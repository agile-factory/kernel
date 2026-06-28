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

// DeleteEndCustomerUseCase soft-deletes an end-customer.
type DeleteEndCustomerUseCase struct {
	db        *pgxpool.Pool
	repo      domain.EndCustomerRepository
	checker   permission.Checker
	publisher events.Publisher
}

// NewDeleteEndCustomerUseCase builds the use case.
func NewDeleteEndCustomerUseCase(
	db *pgxpool.Pool,
	repo domain.EndCustomerRepository,
	checker permission.Checker,
	publisher events.Publisher,
) *DeleteEndCustomerUseCase {
	return &DeleteEndCustomerUseCase{
		db:        db,
		repo:      repo,
		checker:   checker,
		publisher: publisher,
	}
}

// Execute soft-deletes the end-customer and publishes end_customer.deleted.
func (uc *DeleteEndCustomerUseCase) Execute(ctx context.Context, id string) error {
	if err := uc.checker.Require(ctx, permission.EndCustomerDelete); err != nil {
		return err
	}
	if id == "" {
		return apperrors.BadRequest("id is required")
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)
	actorID := kernelauth.UserIDFromContext(ctx)
	now := time.Now().UTC()

	return execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.SoftDelete(ctx, tenantID, id); err != nil {
			return err
		}
		return uc.publisher.Publish(ctx, events.Event{
			Type:    "end_customer.deleted",
			Payload: map[string]any{"end_customer_id": id, "tenant_id": tenantID},
			Meta:    events.EventMeta{TenantID: tenantID, ActorID: actorID, OccurredAt: now},
		})
	})
}
