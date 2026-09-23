package usecase

import (
	"context"
	stderrors "errors"
	"time"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/audit"
	auditdomain "github.com/agile-factory/kernel/platform/audit/domain"
	authdomain "github.com/agile-factory/kernel/platform/auth/domain"
	memberdomain "github.com/agile-factory/kernel/platform/membership/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AcceptInviteOutput is what /api/v1/auth/accept-invite returns once a
// pending invite is redeemed: the user (possibly freshly created), the
// resulting ACTIVE membership, and an immediately usable token pair.
type AcceptInviteOutput struct {
	User         *authdomain.User
	Membership   *memberdomain.Membership
	AccessToken  string
	RefreshToken string
}

// AcceptInviteUseCase consumes an invite token and produces an ACTIVE
// membership with the pre-decided role. If the email already corresponds
// to an existing user, that user is reused — but only after the caller
// proves ownership of the account with its current password (possession of
// the invite token alone is not enough: the inviter may have seen it).
// Otherwise a new VERIFIED user is created (the invite token's redemption is itself
// the proof of email ownership, per RFC-0018 §1).
type AcceptInviteUseCase struct {
	runInTx txRunner

	users           authdomain.UserRepository
	memberships     memberdomain.MembershipRepository
	inviteTokens    memberdomain.InviteTokenRepository
	refreshTokens   authdomain.RefreshTokenRepository
	hasher          PasswordHasherVerifier
	tokenIssuer     TokenGenerator
	publisher       events.Publisher
	auditLogger     audit.EventLogger
	refreshTokenTTL time.Duration
}

func NewAcceptInviteUseCase(
	pool *pgxpool.Pool,
	users authdomain.UserRepository,
	memberships memberdomain.MembershipRepository,
	inviteTokens memberdomain.InviteTokenRepository,
	refreshTokens authdomain.RefreshTokenRepository,
	hasher PasswordHasherVerifier,
	tokenIssuer TokenGenerator,
	publisher events.Publisher,
	auditLogger audit.EventLogger,
	refreshTokenTTL time.Duration,
) *AcceptInviteUseCase {
	return &AcceptInviteUseCase{
		runInTx:         poolTxRunner(pool),
		users:           users,
		memberships:     memberships,
		inviteTokens:    inviteTokens,
		refreshTokens:   refreshTokens,
		hasher:          hasher,
		tokenIssuer:     tokenIssuer,
		publisher:       publisher,
		auditLogger:     auditLogger,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// Execute redeems the invite token. For a new user, password becomes the
// account password. For an existing user, password must be the account's
// current password; otherwise ErrInvalidCredentials is returned.
func (uc *AcceptInviteUseCase) Execute(ctx context.Context, rawToken, password string) (*AcceptInviteOutput, error) {
	out := &AcceptInviteOutput{}

	txErr := uc.runInTx(ctx, func(ctx context.Context) error {
		invite, err := uc.inviteTokens.FindByToken(ctx, rawToken)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if invite.RevokedAt != nil {
			return memberdomain.ErrInviteTokenRevoked
		}
		if invite.ConsumedAt != nil {
			return memberdomain.ErrInviteTokenAlreadyConsumed
		}
		if now.After(invite.ExpiresAt) {
			return memberdomain.ErrInviteTokenExpired
		}

		// 1. Resolve or create the invited user.
		email, err := authdomain.NewEmail(invite.Email)
		if err != nil {
			return err
		}
		user, err := uc.users.FindByEmail(ctx, email)
		switch {
		case err == nil:
			// Existing account: the invite token proves the inviter picked
			// this email, not that the caller owns the account. Require the
			// account password so a token leak cannot become a takeover.
			if password == "" {
				return authdomain.ErrInvalidCredentials
			}
			if verr := uc.hasher.Verify(user.HashedPassword, password); verr != nil {
				return authdomain.ErrInvalidCredentials
			}
			if !user.IsActive() {
				return authdomain.ErrUserNotActive
			}
		case stderrors.Is(err, authdomain.ErrUserNotFound):
			if len(password) < MinPasswordLength {
				return authdomain.ErrPasswordTooShort
			}
			hashed, hashErr := uc.hasher.Hash(password)
			if hashErr != nil {
				return errors.WrapInternal(hashErr)
			}
			user = authdomain.NewVerifiedUser(uuid.NewString(), email, hashed, authdomain.ROLE_VIEWER)
			if err := uc.users.Save(ctx, user); err != nil {
				return errors.WrapInternal(err)
			}
		default:
			return errors.WrapInternal(err)
		}
		out.User = user

		// 2. Membership (ACTIVE, with the role baked into the invite).
		membership := memberdomain.NewSelfMembership(uuid.NewString(), user.ID, invite.TenantID, invite.Role)
		// NewSelfMembership marks the membership as having no inviter; carry
		// over the invite's invited_by so the audit trail is accurate.
		membership.InvitedBy = invite.InvitedBy
		if err := uc.memberships.Save(ctx, membership); err != nil {
			return errors.WrapInternal(err)
		}
		out.Membership = membership

		// 3. Mark the invite token consumed.
		if err := uc.inviteTokens.Consume(ctx, invite.ID); err != nil {
			return errors.WrapInternal(err)
		}

		// 4. Outbox event.
		if uc.publisher != nil {
			if err := uc.publisher.Publish(ctx, events.Event{
				Type: "auth.invite_accepted",
				Payload: map[string]any{
					"user_id":       user.ID,
					"tenant_id":     invite.TenantID,
					"membership_id": membership.ID,
					"role":          string(invite.Role),
				},
				Meta: events.EventMeta{
					TenantID:   invite.TenantID,
					ActorID:    user.ID,
					OccurredAt: now,
				},
			}); err != nil {
				return errors.WrapInternal(err)
			}
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	// 5. Token issuance (post-tx; same reasoning as SignupOwner).
	accessToken, err := uc.tokenIssuer.GenerateToken(out.User)
	if err != nil {
		return nil, errors.WrapInternal(err)
	}
	rawRefresh, refresh, err := authdomain.NewRefreshToken(uuid.NewString(), out.User.ID, out.Membership.TenantID, uc.refreshTokenTTL)
	if err != nil {
		return nil, errors.WrapInternal(err)
	}
	if err := uc.refreshTokens.Save(ctx, refresh); err != nil {
		return nil, errors.WrapInternal(err)
	}
	out.AccessToken = accessToken
	out.RefreshToken = rawRefresh

	// 6. Best-effort audit log (transition mode).
	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   out.Membership.TenantID,
			ActorID:    out.User.ID,
			Action:     auditdomain.ACTION_MEMBER_ACCEPTED,
			TargetType: "membership",
			TargetID:   out.Membership.ID,
			Metadata: map[string]any{
				"role": string(out.Membership.Role),
				"flow": "accept_invite",
			},
		})
	}

	return out, nil
}
