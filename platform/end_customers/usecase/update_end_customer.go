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

// UpdateEndCustomerInput carries all mutable fields.
// Nil pointer fields are left unchanged; slice/map nil means "no change".
type UpdateEndCustomerInput struct {
	ID         string
	ExternalID *string
	Email      *string
	Phone      *string
	FirstName  *string
	LastName   *string
	Tags       []string
	TagsSet    bool // true when Tags should be replaced (even to empty slice)
	Metadata   map[string]string
	MetadataSet bool // true when Metadata should be replaced
}

// UpdateEndCustomerOutput wraps the updated end-customer.
type UpdateEndCustomerOutput struct {
	EndCustomer *domain.EndCustomer
}

// UpdateEndCustomerUseCase applies a partial update to an end-customer.
type UpdateEndCustomerUseCase struct {
	db        *pgxpool.Pool
	repo      domain.EndCustomerRepository
	checker   permission.Checker
	publisher events.Publisher
}

// NewUpdateEndCustomerUseCase builds the use case.
func NewUpdateEndCustomerUseCase(
	db *pgxpool.Pool,
	repo domain.EndCustomerRepository,
	checker permission.Checker,
	publisher events.Publisher,
) *UpdateEndCustomerUseCase {
	return &UpdateEndCustomerUseCase{
		db:        db,
		repo:      repo,
		checker:   checker,
		publisher: publisher,
	}
}

// Execute fetches, applies the patch, persists, and publishes end_customer.updated.
func (uc *UpdateEndCustomerUseCase) Execute(ctx context.Context, input UpdateEndCustomerInput) (UpdateEndCustomerOutput, error) {
	if err := uc.checker.Require(ctx, permission.EndCustomerUpdate); err != nil {
		return UpdateEndCustomerOutput{}, err
	}
	if input.ID == "" {
		return UpdateEndCustomerOutput{}, apperrors.BadRequest("id is required")
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)
	actorID := kernelauth.UserIDFromContext(ctx)

	ec, err := uc.repo.FindByID(ctx, tenantID, input.ID)
	if err != nil {
		return UpdateEndCustomerOutput{}, err
	}

	if input.ExternalID != nil {
		ec.ExternalID = *input.ExternalID
	}
	if input.Email != nil {
		ec.Email = *input.Email
	}
	if input.Phone != nil {
		ec.Phone = *input.Phone
	}
	if input.FirstName != nil {
		if *input.FirstName == "" {
			return UpdateEndCustomerOutput{}, apperrors.BadRequest("first_name must not be empty")
		}
		ec.FirstName = *input.FirstName
	}
	if input.LastName != nil {
		if *input.LastName == "" {
			return UpdateEndCustomerOutput{}, apperrors.BadRequest("last_name must not be empty")
		}
		ec.LastName = *input.LastName
	}
	if input.TagsSet {
		ec.Tags = input.Tags
	}
	if input.MetadataSet {
		if err := validateMetadata(input.Metadata); err != nil {
			return UpdateEndCustomerOutput{}, err
		}
		ec.Metadata = input.Metadata
	}

	now := time.Now().UTC()
	ec.UpdatedAt = now

	err = execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Update(ctx, ec); err != nil {
			return err
		}
		return uc.publisher.Publish(ctx, events.Event{
			Type:    "end_customer.updated",
			Payload: map[string]any{"end_customer_id": ec.ID, "tenant_id": tenantID},
			Meta:    events.EventMeta{TenantID: tenantID, ActorID: actorID, OccurredAt: now},
		})
	})
	if err != nil {
		return UpdateEndCustomerOutput{}, err
	}
	return UpdateEndCustomerOutput{EndCustomer: ec}, nil
}
