package usecase

import "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/domain"

type LogEventParams struct {
	TenantID   string
	ActorID    string
	Action     domain.AuditAction
	TargetType string
	TargetID   string
	Metadata   map[string]any
}
