package usecase

import (
	"context"
	"fmt"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxMetadataKeys      = 20
	maxMetadataValueLen  = 256
)

// CreateEndCustomerInput carries all fields for a new end-customer.
type CreateEndCustomerInput struct {
	ExternalID string
	Email      string
	Phone      string
	FirstName  string
	LastName   string
	Tags       []string
	Metadata   map[string]string
}

// CreateEndCustomerOutput wraps the created end-customer.
type CreateEndCustomerOutput struct {
	EndCustomer *domain.EndCustomer
}

// CreateEndCustomerUseCase creates a new end-customer for the authenticated tenant.
type CreateEndCustomerUseCase struct {
	db        *pgxpool.Pool
	repo      domain.EndCustomerRepository
	checker   permission.Checker
	publisher events.Publisher
}

// NewCreateEndCustomerUseCase builds the use case with its dependencies.
func NewCreateEndCustomerUseCase(
	db *pgxpool.Pool,
	repo domain.EndCustomerRepository,
	checker permission.Checker,
	publisher events.Publisher,
) *CreateEndCustomerUseCase {
	return &CreateEndCustomerUseCase{
		db:        db,
		repo:      repo,
		checker:   checker,
		publisher: publisher,
	}
}

// Execute validates, creates, and publishes an end_customer.created event.
func (uc *CreateEndCustomerUseCase) Execute(ctx context.Context, input CreateEndCustomerInput) (CreateEndCustomerOutput, error) {
	if err := uc.checker.Require(ctx, permission.EndCustomerCreate); err != nil {
		return CreateEndCustomerOutput{}, err
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)
	actorID := kernelauth.UserIDFromContext(ctx)

	if input.FirstName == "" {
		return CreateEndCustomerOutput{}, apperrors.BadRequest("first_name is required")
	}
	if input.LastName == "" {
		return CreateEndCustomerOutput{}, apperrors.BadRequest("last_name is required")
	}
	if input.Email == "" && input.Phone == "" {
		return CreateEndCustomerOutput{}, apperrors.BadRequest("at least one of email or phone is required")
	}
	if err := validateMetadata(input.Metadata); err != nil {
		return CreateEndCustomerOutput{}, err
	}

	now := time.Now().UTC()
	ec := &domain.EndCustomer{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		ExternalID: input.ExternalID,
		Email:      input.Email,
		Phone:      input.Phone,
		FirstName:  input.FirstName,
		LastName:   input.LastName,
		Status:     domain.StatusActive,
		Tags:       input.Tags,
		Metadata:   input.Metadata,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if ec.Metadata == nil {
		ec.Metadata = map[string]string{}
	}

	err := execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Create(ctx, ec); err != nil {
			return err
		}
		return uc.publisher.Publish(ctx, events.Event{
			Type:    "end_customer.created",
			Payload: map[string]any{"end_customer_id": ec.ID, "tenant_id": tenantID},
			Meta:    events.EventMeta{TenantID: tenantID, ActorID: actorID, OccurredAt: now},
		})
	})
	if err != nil {
		return CreateEndCustomerOutput{}, err
	}
	return CreateEndCustomerOutput{EndCustomer: ec}, nil
}

// validateMetadata enforces: max 20 keys, max 256 chars per value, non-empty keys.
func validateMetadata(m map[string]string) error {
	if len(m) > maxMetadataKeys {
		return apperrors.BadRequest(fmt.Sprintf("metadata must have at most %d keys", maxMetadataKeys))
	}
	for k, v := range m {
		if k == "" {
			return apperrors.BadRequest("metadata keys must not be empty")
		}
		if len(v) > maxMetadataValueLen {
			return apperrors.BadRequest(fmt.Sprintf("metadata value for key %q exceeds %d characters", k, maxMetadataValueLen))
		}
	}
	return nil
}
