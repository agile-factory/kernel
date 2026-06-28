// Package http provides the HTTP delivery layer for the customer_notifications platform.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
	"github.com/agile-factory/kernel/platform/customer_notifications/usecase"
)

// CustomerNotificationsHandler handles HTTP requests for the customer_notifications surface.
type CustomerNotificationsHandler struct {
	upsertUC   *usecase.UpsertCustomerTemplateUseCase
	deleteUC   *usecase.DeleteCustomerTemplateUseCase
	listLogsUC *usecase.ListCustomerNotificationLogsUseCase
}

// NewCustomerNotificationsHandler wires the handler.
func NewCustomerNotificationsHandler(
	upsertUC *usecase.UpsertCustomerTemplateUseCase,
	deleteUC *usecase.DeleteCustomerTemplateUseCase,
	listLogsUC *usecase.ListCustomerNotificationLogsUseCase,
) *CustomerNotificationsHandler {
	return &CustomerNotificationsHandler{
		upsertUC:   upsertUC,
		deleteUC:   deleteUC,
		listLogsUC: listLogsUC,
	}
}

// ── DTOs ─────────────────────────────────────────────────────────────────────

type upsertTemplateRequest struct {
	SubjectTmpl string `json:"subject_tmpl"`
	BodyTmpl    string `json:"body_tmpl"`
	Locale      string `json:"locale"`
}

type logResponse struct {
	ID            string  `json:"id"`
	TenantID      string  `json:"tenant_id"`
	EndCustomerID string  `json:"end_customer_id,omitempty"`
	Channel       string  `json:"channel"`
	Recipient     string  `json:"recipient"`
	TemplateKey   string  `json:"template_key"`
	Status        string  `json:"status"`
	ExternalID    string  `json:"external_id,omitempty"`
	FailureReason string  `json:"failure_reason,omitempty"`
	SentAt        *string `json:"sent_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

func toLogResponse(l *domain.CustomerNotificationLog) logResponse {
	r := logResponse{
		ID:            l.ID,
		TenantID:      l.TenantID,
		EndCustomerID: l.EndCustomerID,
		Channel:       string(l.Channel),
		Recipient:     l.Recipient,
		TemplateKey:   l.TemplateKey,
		Status:        string(l.Status),
		ExternalID:    l.ExternalID,
		FailureReason: l.FailureReason,
		CreatedAt:     l.CreatedAt.Format(time.RFC3339),
	}
	if l.SentAt != nil {
		s := l.SentAt.Format(time.RFC3339)
		r.SentAt = &s
	}
	return r
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// UpsertTemplate handles PUT /api/v1/tenants/{id}/customer-templates/{key}/{channel}
func (h *CustomerNotificationsHandler) UpsertTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	key := r.PathValue("key")
	ch := domain.Channel(r.PathValue("channel"))

	var body upsertTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}
	if body.BodyTmpl == "" {
		response.Error(w, kerrors.BadRequest("body_tmpl is required"))
		return
	}
	locale := body.Locale
	if locale == "" {
		locale = "fr"
	}

	if err := h.upsertUC.Execute(r.Context(), tenantID, usecase.UpsertCustomerTemplateInput{
		Key:         key,
		Channel:     ch,
		Locale:      locale,
		SubjectTmpl: body.SubjectTmpl,
		BodyTmpl:    body.BodyTmpl,
	}); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// DeleteTemplate handles DELETE /api/v1/tenants/{id}/customer-templates/{key}/{channel}
func (h *CustomerNotificationsHandler) DeleteTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	key := r.PathValue("key")
	ch := domain.Channel(r.PathValue("channel"))
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "fr"
	}

	if err := h.deleteUC.Execute(r.Context(), tenantID, key, ch, locale); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// ListLogs handles GET /api/v1/tenants/{id}/customer-notification-logs
func (h *CustomerNotificationsHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := kernelauth.TenantIDFromContext(r.Context())
	if tenantID == "" {
		tenantID = r.PathValue("id")
	}
	q := r.URL.Query()

	out, err := h.listLogsUC.Execute(r.Context(), tenantID, usecase.ListCustomerNotificationLogsInput{
		EndCustomerID: q.Get("end_customer_id"),
		Channel:       domain.Channel(q.Get("channel")),
		Status:        domain.DeliveryStatus(q.Get("status")),
		TemplateKey:   q.Get("template_key"),
		Cursor:        q.Get("cursor"),
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	resp := make([]logResponse, len(out.Logs))
	for i, l := range out.Logs {
		resp[i] = toLogResponse(l)
	}
	result := map[string]any{"data": resp}
	if out.NextCursor != "" {
		result["next_cursor"] = out.NextCursor
	}
	response.JSON(w, http.StatusOK, result)
}
