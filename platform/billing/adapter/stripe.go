// Package adapter contains payment processor adapters.
package adapter

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/billing/domain"
)

// StripeConfig holds credentials and endpoint URLs for Stripe.
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
	SuccessURL    string
	CancelURL     string
}

// StripeAdapter implements domain.PaymentProcessor using Stripe Checkout Sessions.
// It uses only stdlib net/http — no official Stripe SDK dependency.
type StripeAdapter struct {
	cfg    StripeConfig
	client *http.Client
}

// NewStripeAdapter returns a new StripeAdapter. cfg.SecretKey must be non-empty.
func NewStripeAdapter(cfg StripeConfig) *StripeAdapter {
	return &StripeAdapter{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateCheckoutSession creates a Stripe Checkout Session for the given tenant and plan.
// Returns a CheckoutResult with the Stripe Session URL.
func (a *StripeAdapter) CreateCheckoutSession(_ context.Context, tenant domain.BillingTenant, plan domain.Plan) (domain.CheckoutResult, error) {
	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("customer_email", tenant.Email)
	form.Set("success_url", a.cfg.SuccessURL)
	form.Set("cancel_url", a.cfg.CancelURL)
	// Line item: price must be a Stripe Price ID. We store ExternalID in plan.ID for now.
	// In production, plan.ID should map to a Stripe Price ID via tenant_settings.
	form.Set("line_items[0][price]", plan.ID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("metadata[tenant_id]", tenant.ID)
	form.Set("metadata[plan_name]", plan.Name)

	req, err := http.NewRequest("POST", "https://api.stripe.com/v1/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return domain.CheckoutResult{}, fmt.Errorf("stripe: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.cfg.SecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return domain.CheckoutResult{}, fmt.Errorf("stripe: checkout session: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return domain.CheckoutResult{}, fmt.Errorf("stripe: checkout session: status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return domain.CheckoutResult{}, fmt.Errorf("stripe: decode response: %w", err)
	}
	return domain.CheckoutResult{RedirectURL: result.URL, ExternalID: result.ID}, nil
}

// HandleWebhook verifies the Stripe-Signature header and parses the event.
// Stripe sends: Stripe-Signature: t=<timestamp>,v1=<sig>
func (a *StripeAdapter) HandleWebhook(_ context.Context, signature string, body []byte) (domain.PaymentEvent, error) {
	if err := verifyStripeSignature(a.cfg.WebhookSecret, signature, body); err != nil {
		return domain.PaymentEvent{}, apperrors.BadRequest("stripe: invalid webhook signature")
	}

	var ev struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				Metadata struct {
					TenantID string `json:"tenant_id"`
				} `json:"metadata"`
				AmountTotal int64  `json:"amount_total"`
				Currency    string `json:"currency"`
				ID          string `json:"id"`
			} `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		return domain.PaymentEvent{}, apperrors.BadRequest("stripe: invalid webhook payload")
	}

	var evType domain.PaymentEventType
	switch ev.Type {
	case "checkout.session.completed":
		evType = domain.PaymentSucceeded
	case "invoice.payment_failed":
		evType = domain.PaymentFailed
	case "invoice.payment_succeeded":
		evType = domain.SubscriptionRenewed
	default:
		// Unknown event type — return a no-op event; caller should ignore.
		return domain.PaymentEvent{Type: domain.PaymentEventType(ev.Type)}, nil
	}

	return domain.PaymentEvent{
		Type:     evType,
		TenantID: ev.Data.Object.Metadata.TenantID,
		Amount:   ev.Data.Object.AmountTotal,
		Currency: strings.ToUpper(ev.Data.Object.Currency),
		Ref:      ev.Data.Object.ID,
	}, nil
}

// RecordPayment is a no-op for Stripe — payment is recorded via webhook.
func (a *StripeAdapter) RecordPayment(_ context.Context, _, _ string, _ int64, _ string) error {
	return nil
}

// verifyStripeSignature checks the Stripe-Signature header.
// Format: t=<timestamp>,v1=<hex_sig>[,v1=<hex_sig>...]
func verifyStripeSignature(secret, header string, payload []byte) error {
	parts := strings.Split(header, ",")
	var timestamp string
	var sigs []string
	for _, part := range parts {
		if strings.HasPrefix(part, "t=") {
			timestamp = strings.TrimPrefix(part, "t=")
		} else if strings.HasPrefix(part, "v1=") {
			sigs = append(sigs, strings.TrimPrefix(part, "v1="))
		}
	}
	if timestamp == "" || len(sigs) == 0 {
		return fmt.Errorf("malformed signature header")
	}

	// Reject replays older than 5 minutes.
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	if time.Since(time.Unix(ts, 0)) > 5*time.Minute {
		return fmt.Errorf("webhook timestamp too old")
	}

	signedPayload := timestamp + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range sigs {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return nil
		}
	}
	return fmt.Errorf("signature mismatch")
}

// Ensure StripeAdapter implements PaymentProcessor at compile time.
var _ domain.PaymentProcessor = (*StripeAdapter)(nil)

// bufReader is a helper that re-reads from a []byte.
var _ = bytes.NewReader // avoid unused import
