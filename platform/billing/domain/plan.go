package domain

import (
	"context"
	"time"
)

// Plan is a SaaS subscription tier.
type Plan struct {
	ID            string
	Name          string
	PriceCents    int64
	Currency      string
	Interval      string // "MONTHLY" | "YEARLY"
	MaxUsers      *int64
	MaxStorage    *int64
	MaxAPICalls   *int64
	CanExport     bool
	CanWhiteLabel bool
	CanMerchant   bool
	CreatedAt     time.Time
}

// PlanRepository is the storage port for plans.
type PlanRepository interface {
	FindByID(ctx context.Context, id string) (*Plan, error)
	FindAll(ctx context.Context) ([]*Plan, error)
}
