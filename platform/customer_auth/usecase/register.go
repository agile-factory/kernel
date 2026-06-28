package usecase

import (
	"context"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerRegisterUseCase creates a password credential for an end-customer.
// Typically called after the tenant enables the B2B2C portal and the
// end-customer chooses a password.
type CustomerRegisterUseCase struct {
	db     *pgxpool.Pool
	repo   domain.CredentialRepository
	hasher Hasher
	pub    events.Publisher
}

// NewCustomerRegisterUseCase wires the use case.
func NewCustomerRegisterUseCase(db *pgxpool.Pool, repo domain.CredentialRepository, hasher Hasher, pub events.Publisher) *CustomerRegisterUseCase {
	return &CustomerRegisterUseCase{db: db, repo: repo, hasher: hasher, pub: pub}
}

// CustomerRegisterInput holds the registration parameters.
type CustomerRegisterInput struct {
	EndCustomerID string
	TenantID      string
	Password      string
}

// Execute registers a credential. Returns ErrPasswordTooShort or
// ErrCredentialAlreadyExists on validation/conflict.
func (uc *CustomerRegisterUseCase) Execute(ctx context.Context, in CustomerRegisterInput) error {
	if len(in.Password) < 12 {
		return domain.ErrPasswordTooShort
	}
	return execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		hash, err := uc.hasher.Hash(in.Password)
		if err != nil {
			return err
		}
		cred := &domain.CustomerCredential{
			EndCustomerID: in.EndCustomerID,
			TenantID:      in.TenantID,
			PasswordHash:  hash,
		}
		if err := uc.repo.Create(ctx, cred); err != nil {
			return err
		}
		return uc.pub.Publish(ctx, events.Event{
			Type: "customer_auth.registered",
			Payload: map[string]any{
				"end_customer_id": in.EndCustomerID,
				"tenant_id":       in.TenantID,
			},
			Meta: events.EventMeta{TenantID: in.TenantID, OccurredAt: time.Now()},
		})
	})
}
