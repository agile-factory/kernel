package usecase

import (
	"context"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SuspendEndCustomerUseCase transitions an end-customer from ACTIVE to SUSPENDED.
type SuspendEndCustomerUseCase struct {
	db        *pgxpool.Pool
	repo      domain.EndCustomerRepository
	checker   permission.Checker
	publisher events.Publisher
}

// NewSuspendEndCustomerUseCase builds the use case.
func NewSuspendEndCustomerUseCase(
	db *pgxpool.Pool,
	repo domain.EndCustomerRepository,
	checker permission.Checker,
	publisher events.Publisher,
) *SuspendEndCustomerUseCase {
	return &SuspendEndCustomerUseCase{
		db:        db,
		repo:      repo,
		checker:   checker,
		publisher: publisher,
	}
}

// Execute fetches, calls Suspend(), persists, and publishes end_customer.suspended.
func (uc *SuspendEndCustomerUseCase) Execute(ctx context.Context, id string) error {
	if err := uc.checker.Require(ctx, permission.EndCustomerSuspend); err != nil {
		return err
	}
	if id == "" {
		return apperrors.BadRequest("id is required")
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)
	actorID := kernelauth.UserIDFromContext(ctx)

	ec, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if err := ec.Suspend(); err != nil {
		return err
	}

	now := time.Now().UTC()
	ec.UpdatedAt = now

	return execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Update(ctx, ec); err != nil {
			return err
		}
		return uc.publisher.Publish(ctx, events.Event{
			Type:    "end_customer.suspended",
			Payload: map[string]any{"end_customer_id": ec.ID, "tenant_id": tenantID},
			Meta:    events.EventMeta{TenantID: tenantID, ActorID: actorID, OccurredAt: now},
		})
	})
}
