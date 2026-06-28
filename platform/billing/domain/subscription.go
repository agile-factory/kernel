package domain

import (
	"context"
	"time"
)

// SubscriptionStatus is the lifecycle state of a subscription.
type SubscriptionStatus string

const (
	StatusTrialing  SubscriptionStatus = "TRIALING"
	StatusActive    SubscriptionStatus = "ACTIVE"
	StatusPastDue   SubscriptionStatus = "PAST_DUE"
	StatusCanceled  SubscriptionStatus = "CANCELED"
)

// Subscription tracks a tenant's SaaS plan subscription.
type Subscription struct {
	ID                  string
	TenantID            string
	PlanID              string
	Status              SubscriptionStatus
	CurrentPeriodStart  time.Time
	CurrentPeriodEnd    time.Time
	CanceledAt          *time.Time
	PaymentMethod       string // "STRIPE" | "WAVE" | "ORANGE_MONEY" | "FREE_MONEY"
	ExternalID          string // Stripe Subscription ID
	MobileMoneyRef      string
	CreatedAt           time.Time
}

// SubscriptionRepository is the storage port for subscriptions.
type SubscriptionRepository interface {
	FindByTenantID(ctx context.Context, tenantID string) (*Subscription, error)
	Upsert(ctx context.Context, sub *Subscription) error
}
