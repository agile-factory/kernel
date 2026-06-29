// Package infrastructure provides the external-dependency implementations
// for the customer_notifications platform.
package infrastructure

import (
	"context"
	"log/slog"
)

// NoopSMSAdapter logs SMS sends without actually sending.
// Used in development and test environments.
type NoopSMSAdapter struct {
	logger *slog.Logger
}

// NewNoopSMSAdapter creates a NoopSMSAdapter.
func NewNoopSMSAdapter(logger *slog.Logger) *NoopSMSAdapter {
	return &NoopSMSAdapter{logger: logger}
}

// Send logs the message and returns a synthetic external ID.
func (a *NoopSMSAdapter) Send(ctx context.Context, to, body string) (string, error) {
	a.logger.InfoContext(ctx, "noop SMS send", "to", to, "body_len", len(body))
	return "noop-sms-" + to, nil
}
