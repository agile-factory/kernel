package domain

import "context"

// PaymentProcessor is the port for payment processing.
// Adapters: StripeAdapter (international), WaveAdapter (CEDEAO XOF).
type PaymentProcessor interface {
	CreateCheckoutSession(ctx context.Context, tenant BillingTenant, plan Plan) (CheckoutResult, error)
	HandleWebhook(ctx context.Context, signature string, body []byte) (PaymentEvent, error)
	RecordPayment(ctx context.Context, tenantID string, ref string, amount int64, currency string) error
}

// BillingTenant carries the tenant fields needed by payment adapters.
type BillingTenant struct {
	ID    string
	Name  string
	Email string
}

// CheckoutResult is returned by CreateCheckoutSession.
type CheckoutResult struct {
	RedirectURL string
	ExternalID  string // Stripe Session ID, Wave Order ID, etc.
}

// PaymentEventType classifies inbound webhook events.
type PaymentEventType string

const (
	PaymentSucceeded    PaymentEventType = "PAYMENT_SUCCEEDED"
	PaymentFailed       PaymentEventType = "PAYMENT_FAILED"
	SubscriptionRenewed PaymentEventType = "SUBSCRIPTION_RENEWED"
)

// PaymentEvent is produced by HandleWebhook.
type PaymentEvent struct {
	Type     PaymentEventType
	TenantID string
	Amount   int64
	Currency string
	Ref      string
}
