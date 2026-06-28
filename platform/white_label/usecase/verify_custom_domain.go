package usecase

import (
	"context"
	"errors"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/white_label/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VerifyCustomDomainUseCase performs a DNS TXT lookup to verify ownership of
// the tenant's custom domain.
type VerifyCustomDomainUseCase struct {
	db          *pgxpool.Pool
	repo        domain.BrandRepository
	checker     permission.Checker
	publisher   events.Publisher
	dnsResolver DNSResolver
}

// NewVerifyCustomDomainUseCase builds the use case.
func NewVerifyCustomDomainUseCase(
	db *pgxpool.Pool,
	repo domain.BrandRepository,
	checker permission.Checker,
	publisher events.Publisher,
	dnsResolver DNSResolver,
) *VerifyCustomDomainUseCase {
	return &VerifyCustomDomainUseCase{
		db:          db,
		repo:        repo,
		checker:     checker,
		publisher:   publisher,
		dnsResolver: dnsResolver,
	}
}

// Execute looks up the DNS TXT record and updates the domain verification status.
//
// States after Execute:
//   - VERIFIED: TXT record found and matches the stored token.
//   - TIMEOUT:  DNS query timed out — retryable.
//   - FAILED:   TXT record found but token mismatch — requires reconfiguration.
func (uc *VerifyCustomDomainUseCase) Execute(ctx context.Context) error {
	if err := uc.checker.Require(ctx, permission.BrandWrite); err != nil {
		return err
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)
	actorID := kernelauth.UserIDFromContext(ctx)

	brand, err := uc.repo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return err
	}

	if brand.CustomDomainStatus != domain.DomainPending && brand.CustomDomainStatus != domain.DomainTimeout {
		return domain.ErrDomainNotPending
	}

	if brand.CustomDomain == "" {
		return apperrors.BadRequest("no custom domain configured")
	}

	// DNS lookup with a 5-second timeout (RFC-0035 §6).
	dnsCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	records, dnsErr := uc.dnsResolver.LookupTXT(dnsCtx, "_kernel-verify."+brand.CustomDomain)

	if dnsErr != nil {
		// Network or timeout error — transient, mark as TIMEOUT so callers retry.
		_ = uc.repo.UpdateDomainStatus(ctx, tenantID, domain.DomainTimeout, nil)
		return apperrors.BadRequest("DNS lookup timed out or failed — please retry. " + dnsErr.Error())
	}

	// Check if expected token appears in any TXT record.
	for _, r := range records {
		if r == brand.CustomDomainToken {
			now := time.Now().UTC()
			err = execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
				if err := uc.repo.UpdateDomainStatus(ctx, tenantID, domain.DomainVerified, &now); err != nil {
					return err
				}
				return uc.publisher.Publish(ctx, events.Event{
					Type:    "white_label.domain_verified",
					Payload: map[string]any{"tenant_id": tenantID, "custom_domain": brand.CustomDomain},
					Meta:    events.EventMeta{TenantID: tenantID, ActorID: actorID, OccurredAt: now},
				})
			})
			return err
		}
	}

	// TXT record found but token mismatch — not retryable without reconfiguration.
	_ = uc.repo.UpdateDomainStatus(ctx, tenantID, domain.DomainFailed, nil)
	return errors.New("DNS TXT record does not match expected token — please add '_kernel-verify." +
		brand.CustomDomain + "' TXT record with value: " + brand.CustomDomainToken)
}
