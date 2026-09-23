package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	stderrors "errors"
	"strings"
	"time"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/audit"
	auditdomain "github.com/agile-factory/kernel/platform/audit/domain"
	authdomain "github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/agile-factory/kernel/platform/membership/domain"
	tenantdomain "github.com/agile-factory/kernel/platform/tenant/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// inviteTokenTTL — RFC-0018 §6.
const inviteTokenTTL = 7 * 24 * time.Hour

// InviteMemberOutput bundles the persisted aggregate and the raw token. The
// raw token is what the consumer emails / shows to the invitee — it lives
// only on this response and is never stored in plaintext outside the
// invite_tokens row.
type InviteMemberOutput struct {
	Invite   *domain.InviteToken
	RawToken string
}

// UserLookup is the slice of the auth user repo the invite flow needs. We
// declare a local interface here to keep membership free of an `auth/usecase`
// import — depending only on the producer's domain.
type UserLookup interface {
	FindByEmail(ctx context.Context, email authdomain.Email) (*authdomain.User, error)
}

// InviteMemberUseCase implements RFC-0018 §6: an admin issues an invite
// token bound to (tenantID, email, role). The invitee redeems the token via
// auth.AcceptInvite, which creates the user (if new) and an ACTIVE
// membership in the same transaction.
//
// This use case no longer creates a PENDING membership row. The membership
// only materialises when the invitee actually accepts. That removes the old
// "ghost membership" state that complicated permission checks.
type InviteMemberUseCase struct {
	pool         *pgxpool.Pool
	memberships  domain.MembershipRepository
	tenants      tenantdomain.TenantRepository
	inviteTokens domain.InviteTokenRepository
	users        UserLookup
	publisher    events.Publisher
	auditLogger  audit.EventLogger
}

func NewInviteMemberUseCase(
	pool *pgxpool.Pool,
	memberships domain.MembershipRepository,
	tenants tenantdomain.TenantRepository,
	inviteTokens domain.InviteTokenRepository,
	users UserLookup,
	publisher events.Publisher,
	auditLogger audit.EventLogger,
) *InviteMemberUseCase {
	return &InviteMemberUseCase{
		pool:         pool,
		memberships:  memberships,
		tenants:      tenants,
		inviteTokens: inviteTokens,
		users:        users,
		publisher:    publisher,
		auditLogger:  auditLogger,
	}
}

// Execute checks the caller's authority + invariants, creates the invite
// token, and emits `member.invited`. Atomic via txctx so the event lands
// alongside the row.
func (uc *InviteMemberUseCase) Execute(ctx context.Context, inviterID, tenantID, rawEmail string, role domain.MemberRole) (*InviteMemberOutput, error) {
	email, err := authdomain.NewEmail(strings.TrimSpace(rawEmail))
	if err != nil {
		return nil, err
	}
	if role == domain.MEMBER_ROLE_OWNER {
		return nil, domain.ErrCannotInviteAsOwner
	}

	// Cheap pre-tx check: tenant must exist + be active. Done outside the
	// tx because it doesn't mutate anything and a stale read here merely
	// surfaces as a clean 404 to the caller.
	tenant, err := uc.tenants.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !tenant.IsActive {
		return nil, tenantdomain.ErrTenantNotFound
	}

	// Authority: inviter must be ACTIVE OWNER/ADMIN in tenant.
	inviterMembership, err := uc.memberships.FindByUserAndTenant(ctx, inviterID, tenantID)
	if err != nil {
		if stderrors.Is(err, domain.ErrMembershipNotFound) {
			return nil, domain.ErrInsufficientMemberRole
		}
		return nil, errors.WrapInternal(err)
	}
	if !inviterMembership.IsActive() || !inviterMembership.IsAdmin() {
		return nil, domain.ErrInsufficientMemberRole
	}

	out := &InviteMemberOutput{}

	txErr := txctx.ExecInTx(ctx, uc.pool, func(ctx context.Context) error {
		// If a user with this email already exists and has an active /
		// pending membership in the tenant, the invite is redundant. The
		// partial UNIQUE index (migration 0004) blocks the eventual
		// accept from inserting a duplicate ACTIVE row, but surfacing the
		// conflict at invite time gives a clearer error.
		if existingUser, lookupErr := uc.users.FindByEmail(ctx, email); lookupErr == nil && existingUser != nil {
			if mship, mErr := uc.memberships.FindByUserAndTenant(ctx, existingUser.ID, tenantID); mErr == nil {
				if mship.Status != domain.MEMBER_STATUS_REVOKED {
					return domain.ErrMembershipAlreadyExists
				}
			} else if !stderrors.Is(mErr, domain.ErrMembershipNotFound) {
				return errors.WrapInternal(mErr)
			}
		}

		// Idempotency: a pending invite for the same (tenant, email) is
		// re-used. Resend should bump the timestamp / regenerate token —
		// we revoke the existing pending invite and mint a fresh one.
		existing, findErr := uc.inviteTokens.FindPendingByTenantAndEmail(ctx, tenantID, email.String())
		switch {
		case findErr == nil:
			if err := uc.inviteTokens.Revoke(ctx, existing.ID); err != nil {
				return errors.WrapInternal(err)
			}
		case !stderrors.Is(findErr, domain.ErrInviteTokenNotFound):
			return errors.WrapInternal(findErr)
		}

		raw, err := generateOpaqueToken()
		if err != nil {
			return errors.WrapInternal(err)
		}
		invite, err := domain.NewInviteToken(
			uuid.NewString(),
			tenantID,
			inviterID,
			email.String(),
			role,
			raw,
			inviteTokenTTL,
		)
		if err != nil {
			return err
		}
		if err := uc.inviteTokens.Save(ctx, invite); err != nil {
			return errors.WrapInternal(err)
		}
		out.Invite = invite
		out.RawToken = raw

		// Outbox event — picked up by platform/notification (P1-1) to send
		// the invite email. The handler is responsible for redeem-URL
		// composition; we just provide the raw token + role + identifiers.
		if uc.publisher != nil {
			if err := uc.publisher.Publish(ctx, events.Event{
				Type: "member.invited",
				Payload: map[string]any{
					"tenant_id":    tenantID,
					"inviter_id":   inviterID,
					"email":        email.String(),
					"role":         string(role),
					"token":        raw,
					"expires_at":   invite.ExpiresAt.UTC().Format(time.RFC3339),
				},
				Meta: events.EventMeta{
					TenantID:   tenantID,
					ActorID:    inviterID,
					OccurredAt: time.Now().UTC(),
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

	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   tenantID,
			ActorID:    inviterID,
			Action:     auditdomain.ACTION_MEMBER_INVITED,
			TargetType: "invite_token",
			TargetID:   out.Invite.ID,
			Metadata: map[string]any{
				"email": email.String(),
				"role":  string(role),
			},
		})
	}

	return out, nil
}

// generateOpaqueToken returns a URL-safe random string. 32 bytes of
// crypto/rand → ~43 chars base64url.
func generateOpaqueToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
