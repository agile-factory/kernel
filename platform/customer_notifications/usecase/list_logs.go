package usecase

import (
	"context"

	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
)

// ListCustomerNotificationLogsUseCase returns a paginated log of dispatches.
type ListCustomerNotificationLogsUseCase struct {
	repo    domain.NotificationLogRepository
	checker permission.Checker
}

// NewListCustomerNotificationLogsUseCase wires the use case.
func NewListCustomerNotificationLogsUseCase(repo domain.NotificationLogRepository, checker permission.Checker) *ListCustomerNotificationLogsUseCase {
	return &ListCustomerNotificationLogsUseCase{repo: repo, checker: checker}
}

// ListCustomerNotificationLogsInput holds the list parameters.
type ListCustomerNotificationLogsInput struct {
	EndCustomerID string
	Channel       domain.Channel
	Status        domain.DeliveryStatus
	TemplateKey   string
	Cursor        string
	Limit         int
}

// ListCustomerNotificationLogsOutput holds the paginated result.
type ListCustomerNotificationLogsOutput struct {
	Logs       []*domain.CustomerNotificationLog
	NextCursor string
}

// Execute returns the paginated logs for the tenant.
func (uc *ListCustomerNotificationLogsUseCase) Execute(ctx context.Context, tenantID string, in ListCustomerNotificationLogsInput) (ListCustomerNotificationLogsOutput, error) {
	if err := uc.checker.Require(ctx, permission.CustomerNotificationView); err != nil {
		return ListCustomerNotificationLogsOutput{}, err
	}
	logs, cursor, err := uc.repo.List(ctx, domain.ListLogsOptions{
		TenantID:      tenantID,
		EndCustomerID: in.EndCustomerID,
		Channel:       in.Channel,
		Status:        in.Status,
		TemplateKey:   in.TemplateKey,
		Cursor:        in.Cursor,
		Limit:         in.Limit,
	})
	if err != nil {
		return ListCustomerNotificationLogsOutput{}, err
	}
	return ListCustomerNotificationLogsOutput{Logs: logs, NextCursor: cursor}, nil
}
