package usecase

import "github.com/agile-factory/kernel/platform/audit/domain"

type LogEventParams struct {
	TenantID   string
	ActorID    string
	Action     domain.AuditAction
	TargetType string
	TargetID   string
	Metadata   map[string]any
}
