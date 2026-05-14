package usecase

import "github.com/AgileFactory/kernel/platform/audit/domain"

type LogEventParams struct {
	TenantID   string
	ActorID    string
	Action     domain.AuditAction
	TargetType string
	TargetID   string
	Metadata   map[string]any
}
