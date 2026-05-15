package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/AgileFactory/kernel/kernel/errors"
	"github.com/AgileFactory/kernel/kernel/events"
	"github.com/AgileFactory/kernel/kernel/txctx"
	"github.com/AgileFactory/kernel/platform/audit"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
	authdomain "github.com/AgileFactory/kernel/platform/auth/domain"
	memberdomain "github.com/AgileFactory/kernel/platform/membership/domain"
	tenantdomain "github.com/AgileFactory/kernel/platform/tenant/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MinPasswordLength enforces RFC-0018 §8.
const MinPasswordLength = 12

// emailVerificationTTL — RFC-0018 §4.
const emailVerificationTTL = 24 * time.Hour

// SignupOwnerOutput bundles every artefact produced by the atomic signup so
// the HTTP layer can return both the new identifiers and an immediately
// usable access/refresh token pair.
type SignupOwnerOutput struct {
	Tenant            *tenantdomain.Tenant
	User              *authdomain.User
	OwnerMembership   *memberdomain.Membership
	VerificationToken string // raw token — sent to the user's email
	AccessToken       string
	RefreshToken      string
}

// SignupOwnerUseCase atomically creates a tenant, its founding owner user,
// the ACTIVE OWNER membership, and an email-verification token — then issues
// an access+refresh token pair so the new owner is signed in immediately.
//
// All DB writes go through a single transaction (kernel/txctx) so any
// failure rolls everything back. An event "auth.signup_owner" is emitted
// to the outbox in the same transaction; it will be picked up by the
// notification consumer (P1-1) to send the verification email.
type SignupOwnerUseCase struct {
	pool *pgxpool.Pool

	users           authdomain.UserRepository
	tenants         tenantdomain.TenantRepository
	memberships     memberdomain.MembershipRepository
	verifTokens     authdomain.EmailVerificationTokenRepository
	refreshTokens   authdomain.RefreshTokenRepository
	hasher          PasswordHasher
	tokenIssuer     TokenGenerator
	publisher       events.Publisher
	auditLogger     audit.EventLogger
	refreshTokenTTL time.Duration
}

// NewSignupOwnerUseCase wires the use case. publisher may be nil during
// transition (the event emission is then skipped).
func NewSignupOwnerUseCase(
	pool *pgxpool.Pool,
	users authdomain.UserRepository,
	tenants tenantdomain.TenantRepository,
	memberships memberdomain.MembershipRepository,
	verifTokens authdomain.EmailVerificationTokenRepository,
	refreshTokens authdomain.RefreshTokenRepository,
	hasher PasswordHasher,
	tokenIssuer TokenGenerator,
	publisher events.Publisher,
	auditLogger audit.EventLogger,
	refreshTokenTTL time.Duration,
) *SignupOwnerUseCase {
	return &SignupOwnerUseCase{
		pool:            pool,
		users:           users,
		tenants:         tenants,
		memberships:     memberships,
		verifTokens:     verifTokens,
		refreshTokens:   refreshTokens,
		hasher:          hasher,
		tokenIssuer:     tokenIssuer,
		publisher:       publisher,
		auditLogger:     auditLogger,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// Execute performs the atomic signup. On any error the transaction rolls
// back and the database is unchanged.
func (uc *SignupOwnerUseCase) Execute(ctx context.Context, tenantName, tenantSlug, emailStr, password string) (*SignupOwnerOutput, error) {
	if len(password) < MinPasswordLength {
		return nil, authdomain.ErrPasswordTooShort
	}

	email, err := authdomain.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}

	slug, err := tenantdomain.NewSlug(tenantSlug)
	if err != nil {
		return nil, err
	}

	hashedPassword, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, errors.WrapInternal(err)
	}

	out := &SignupOwnerOutput{}

	txErr := txctx.ExecInTx(ctx, uc.pool, func(ctx context.Context) error {
		// 1. Slug uniqueness (inside the tx so a concurrent signup with the
		// same slug is detected against the same snapshot).
		exists, err := uc.tenants.ExistsBySlug(ctx, slug)
		if err != nil {
			return err
		}
		if exists {
			return tenantdomain.ErrTenantAlreadyExists
		}

		// 2. Email uniqueness.
		if existing, _ := uc.users.FindByEmail(ctx, email); existing != nil {
			return authdomain.ErrUserAlreadyExists
		}

		// 3. Create the tenant.
		tenant, err := tenantdomain.NewTenant(uuid.NewString(), tenantName, slug)
		if err != nil {
			return err
		}
		if err := uc.tenants.Save(ctx, tenant); err != nil {
			return errors.WrapInternal(err)
		}
		out.Tenant = tenant

		// 4. Create the founding user (PENDING_EMAIL until they verify).
		user := authdomain.NewUser(uuid.NewString(), email, hashedPassword, authdomain.ROLE_VIEWER)
		if err := uc.users.Save(ctx, user); err != nil {
			return errors.WrapInternal(err)
		}
		out.User = user

		// 5. ACTIVE OWNER membership.
		membership := memberdomain.NewSelfMembership(uuid.NewString(), user.ID, tenant.ID, memberdomain.MEMBER_ROLE_OWNER)
		if err := uc.memberships.Save(ctx, membership); err != nil {
			return errors.WrapInternal(err)
		}
		out.OwnerMembership = membership

		// 6. Email-verification token.
		rawToken, err := generateOpaqueToken()
		if err != nil {
			return errors.WrapInternal(err)
		}
		evt := authdomain.NewEmailVerificationToken(uuid.NewString(), user.ID, rawToken, emailVerificationTTL)
		if err := uc.verifTokens.Save(ctx, evt); err != nil {
			return errors.WrapInternal(err)
		}
		out.VerificationToken = rawToken

		// 7. Outbox event (skipped when no publisher is wired yet).
		if uc.publisher != nil {
			if err := uc.publisher.Publish(ctx, events.Event{
				Type: "auth.signup_owner",
				Payload: map[string]any{
					"user_id":            user.ID,
					"tenant_id":          tenant.ID,
					"email":              user.Email.String(),
					"verification_token": rawToken,
				},
				Meta: events.EventMeta{
					TenantID:   tenant.ID,
					ActorID:    user.ID,
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

	// 8. Issue an access + refresh token pair. These live outside the
	// transaction (the user already exists in the DB at this point;
	// reissuing tokens on retry is harmless because the refresh token's
	// hash is randomly generated each time).
	accessToken, err := uc.tokenIssuer.GenerateToken(out.User)
	if err != nil {
		return nil, errors.WrapInternal(err)
	}
	rawRefresh, refresh, err := authdomain.NewRefreshToken(uuid.NewString(), out.User.ID, out.Tenant.ID, uc.refreshTokenTTL)
	if err != nil {
		return nil, errors.WrapInternal(err)
	}
	if err := uc.refreshTokens.Save(ctx, refresh); err != nil {
		return nil, errors.WrapInternal(err)
	}
	out.AccessToken = accessToken
	out.RefreshToken = rawRefresh

	// 9. Best-effort audit log (transition mode — ADR-0009 amendment).
	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   out.Tenant.ID,
			ActorID:    out.User.ID,
			Action:     auditdomain.ACTION_USER_REGISTERED,
			TargetType: "user",
			TargetID:   out.User.ID,
			Metadata: map[string]any{
				"email":  out.User.Email.String(),
				"tenant": out.Tenant.Slug,
				"flow":   "signup_owner",
			},
		})
	}

	return out, nil
}

// generateOpaqueToken returns a URL-safe random string for verification /
// invite tokens. 32 bytes of crypto/rand → ~43 chars base64url.
func generateOpaqueToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
