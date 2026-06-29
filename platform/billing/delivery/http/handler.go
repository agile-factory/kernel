// Package http exposes the billing platform's HTTP surface.
package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/billing/domain"
	"github.com/agile-factory/kernel/platform/billing/usecase"
)

// BillingHandler handles HTTP requests for the billing surface.
type BillingHandler struct {
	listPlansUC    *usecase.ListPlansUseCase
	getSubUC       *usecase.GetSubscriptionUseCase
	processor      domain.PaymentProcessor
	checker        permission.Checker
}

// NewBillingHandler wires the handler.
func NewBillingHandler(
	listPlansUC *usecase.ListPlansUseCase,
	getSubUC *usecase.GetSubscriptionUseCase,
	processor domain.PaymentProcessor,
	checker permission.Checker,
) *BillingHandler {
	return &BillingHandler{
		listPlansUC: listPlansUC,
		getSubUC:    getSubUC,
		processor:   processor,
		checker:     checker,
	}
}

// planResponse is the JSON shape for a plan.
type planResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	PriceCents    int64  `json:"priceCents"`
	Currency      string `json:"currency"`
	Interval      string `json:"interval"`
	CanExport     bool   `json:"canExport"`
	CanWhiteLabel bool   `json:"canWhiteLabel"`
	CanMerchant   bool   `json:"canMerchant"`
}

func toPlanResponse(p *domain.Plan) planResponse {
	return planResponse{
		ID:            p.ID,
		Name:          p.Name,
		PriceCents:    p.PriceCents,
		Currency:      p.Currency,
		Interval:      p.Interval,
		CanExport:     p.CanExport,
		CanWhiteLabel: p.CanWhiteLabel,
		CanMerchant:   p.CanMerchant,
	}
}

// ListPlans handles GET /api/v1/plans — public, no auth required.
func (h *BillingHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	out, err := h.listPlansUC.Execute(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	resp := make([]planResponse, len(out.Plans))
	for i, p := range out.Plans {
		resp[i] = toPlanResponse(p)
	}
	response.JSON(w, http.StatusOK, resp)
}

// subscriptionResponse is the JSON shape for a subscription.
type subscriptionResponse struct {
	ID                 string `json:"id"`
	TenantID           string `json:"tenantId"`
	PlanID             string `json:"planId"`
	Status             string `json:"status"`
	CurrentPeriodStart string `json:"currentPeriodStart"`
	CurrentPeriodEnd   string `json:"currentPeriodEnd"`
	PaymentMethod      string `json:"paymentMethod"`
}

func toSubResponse(s *domain.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID:                 s.ID,
		TenantID:           s.TenantID,
		PlanID:             s.PlanID,
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart.Format(time.RFC3339),
		CurrentPeriodEnd:   s.CurrentPeriodEnd.Format(time.RFC3339),
		PaymentMethod:      s.PaymentMethod,
	}
}

// GetSubscription handles GET /api/v1/tenants/{tenantId}/subscription.
func (h *BillingHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	callerID := kernelauth.UserIDFromContext(r.Context())
	if callerID == "" {
		response.Error(w, apperrors.Unauthorized("authentication required"))
		return
	}
	if err := h.checker.Require(r.Context(), permission.TenantView); err != nil {
		response.Error(w, err)
		return
	}
	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		response.Error(w, apperrors.BadRequest("tenantId path parameter is required"))
		return
	}
	out, err := h.getSubUC.Execute(r.Context(), usecase.GetSubscriptionInput{TenantID: tenantID})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toSubResponse(out.Subscription))
}

// StripeWebhook handles POST /api/v1/billing/webhooks/stripe.
func (h *BillingHandler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		response.Error(w, apperrors.BadRequest("cannot read body"))
		return
	}
	sig := r.Header.Get("Stripe-Signature")
	ev, err := h.processor.HandleWebhook(r.Context(), sig, body)
	if err != nil {
		response.Error(w, err)
		return
	}
	// Process event: update subscription status based on event type.
	_ = ev // TODO: dispatch to subscription update use case in P1-4 follow-up
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"received":true}`))
}

// WaveWebhook handles POST /api/v1/billing/webhooks/wave.
func (h *BillingHandler) WaveWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		response.Error(w, apperrors.BadRequest("cannot read body"))
		return
	}
	sig := r.Header.Get("X-Wave-Signature")
	ev, err := h.processor.HandleWebhook(r.Context(), sig, body)
	if err != nil {
		response.Error(w, err)
		return
	}
	_ = ev
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"received":true}`))
}

// ensure unused import removed
var _ = json.Marshal
