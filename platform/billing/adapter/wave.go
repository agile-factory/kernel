package adapter

import (
	"context"
	"errors"

	"github.com/agile-factory/kernel/platform/billing/domain"
)

// WaveAdapter is a stub for the Wave (Senegal / Côte d'Ivoire) payment adapter.
//
// TODO: Implement once Wave for Business API documentation is available.
// Wave does not currently have a public REST API for programmatic checkout creation.
// Contact: Wave Business API team (wavebusiness.com) for access.
type WaveAdapter struct{}

// NewWaveAdapter returns an unimplemented WaveAdapter stub.
func NewWaveAdapter() *WaveAdapter { return &WaveAdapter{} }

func (a *WaveAdapter) CreateCheckoutSession(_ context.Context, _ domain.BillingTenant, _ domain.Plan) (domain.CheckoutResult, error) {
	return domain.CheckoutResult{}, errors.New("wave: CreateCheckoutSession not yet implemented — Wave API access required")
}

func (a *WaveAdapter) HandleWebhook(_ context.Context, _ string, _ []byte) (domain.PaymentEvent, error) {
	return domain.PaymentEvent{}, errors.New("wave: HandleWebhook not yet implemented")
}

func (a *WaveAdapter) RecordPayment(_ context.Context, _ string, _ string, _ int64, _ string) error {
	return nil // no-op
}

// Ensure WaveAdapter implements PaymentProcessor at compile time.
var _ domain.PaymentProcessor = (*WaveAdapter)(nil)
