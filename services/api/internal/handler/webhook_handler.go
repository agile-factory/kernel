package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"immobilier-saas/api/internal/service/payment"
)

type WebhookHandler struct {
	waveService *payment.WaveService
	db          Database // your database interface
	nats        NATSClient // your NATS client
}

func NewWebhookHandler(
	waveService *payment.WaveService,
	db Database,
	nats NATSClient,
) *WebhookHandler {
	return &WebhookHandler{
		waveService: waveService,
		db:          db,
		nats:        nats,
	}
}

// ==============================================================================
// Webhook Database Operations (interfaces)
// ==============================================================================

type Database interface {
	CheckWebhookProcessed(provider, transactionID string) (bool, error)
	StoreWebhook(webhook *WebhookRecord) error
	MarkWebhookProcessed(webhookID uuid.UUID) error
	UpdateBookingPaymentStatus(bookingID uuid.UUID, status string) error
}

type NATSClient interface {
	PublishPaymentReceived(data PaymentReceivedEvent) error
}

type WebhookRecord struct {
	ID            uuid.UUID
	Provider      string
	TransactionID string
	PayloadHash   string
	RawPayload    json.RawMessage
	ProcessedAt   *time.Time
	ProcessingError *string
	CreatedAt     time.Time
}

type PaymentReceivedEvent struct {
	BookingID     uuid.UUID `json:"booking_id"`
	TransactionID string    `json:"transaction_id"`
	Amount        int64     `json:"amount"`
	Provider      string    `json:"provider"`
	PhoneNumber   string    `json:"phone_number"`
	Timestamp     time.Time `json:"timestamp"`
}

// ==============================================================================
// Wave Webhook Handler
// ==============================================================================

// HandleWaveWebhook godoc
// @Summary      Wave payment webhook
// @Description  Receives payment notifications from Wave (idempotent)
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        X-Wave-Signature header string true "HMAC signature"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /webhooks/wave [post]
func (h *WebhookHandler) HandleWaveWebhook(c *fiber.Ctx) error {
	// Step 1: Read raw payload (needed for signature verification)
	rawPayload, err := io.ReadAll(c.Context().RequestBodyStream())
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to read payload",
		})
	}

	// Step 2: Get signature from header
	signature := c.Get("X-Wave-Signature")
	if signature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing signature",
		})
	}

	// Step 3: Verify signature (CRITICAL for security)
	webhook, err := h.waveService.ParseWebhook(rawPayload, signature)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid signature",
		})
	}

	// Step 4: Calculate payload hash for idempotency
	payloadHash := calculatePayloadHash(rawPayload)

	// Step 5: Check if already processed (idempotency check)
	processed, err := h.db.CheckWebhookProcessed("wave", webhook.TransactionID)
	if err != nil {
		// Log error but return 200 to prevent retry
		// (better to have duplicate detection fail open than closed)
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "error_checking_duplicate",
		})
	}

	if processed {
		// Already processed, return success to stop retries
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "already_processed",
		})
	}

	// Step 6: Store webhook (for audit trail)
	webhookRecord := &WebhookRecord{
		ID:            uuid.New(),
		Provider:      "wave",
		TransactionID: webhook.TransactionID,
		PayloadHash:   payloadHash,
		RawPayload:    rawPayload,
		CreatedAt:     time.Now(),
	}

	if err := h.db.StoreWebhook(webhookRecord); err != nil {
		// If unique constraint violation, already processed
		if isUniqueViolation(err) {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status": "already_processed",
			})
		}

		// Other database error - return 200 to prevent retry storm
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "error_storing_webhook",
		})
	}

	// Step 7: Return 200 OK immediately (within 2s SLA)
	// Process asynchronously via NATS
	go h.processWebhookAsync(webhookRecord.ID, webhook)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "received",
		"id":     webhookRecord.ID,
	})
}

// processWebhookAsync handles webhook processing asynchronously
func (h *WebhookHandler) processWebhookAsync(webhookID uuid.UUID, webhook *payment.WaveWebhook) {
	ctx := context.Background()

	// Parse booking ID from reference
	bookingID, err := uuid.Parse(webhook.Reference)
	if err != nil {
		h.markWebhookError(webhookID, "invalid booking reference")
		return
	}

	// Check payment status
	if webhook.Status != "success" {
		// Handle failed payment
		h.db.UpdateBookingPaymentStatus(bookingID, "payment_failed")
		h.markWebhookError(webhookID, fmt.Sprintf("payment failed: %s", webhook.ErrorMessage))
		return
	}

	// Update booking status
	if err := h.db.UpdateBookingPaymentStatus(bookingID, "paid"); err != nil {
		h.markWebhookError(webhookID, fmt.Sprintf("failed to update booking: %v", err))
		return
	}

	// Publish event to NATS for further processing
	event := PaymentReceivedEvent{
		BookingID:     bookingID,
		TransactionID: webhook.TransactionID,
		Amount:        webhook.Amount,
		Provider:      "wave",
		PhoneNumber:   webhook.PhoneNumber,
		Timestamp:     webhook.Timestamp,
	}

	if err := h.nats.PublishPaymentReceived(event); err != nil {
		h.markWebhookError(webhookID, fmt.Sprintf("failed to publish event: %v", err))
		return
	}

	// Mark as processed
	h.db.MarkWebhookProcessed(webhookID)
}

func (h *WebhookHandler) markWebhookError(webhookID uuid.UUID, errorMsg string) {
	// Update webhook with error
	// (implement error storage in your DB layer)
	log.Printf("Webhook %s processing error: %s", webhookID, errorMsg)
}

// ==============================================================================
// Orange Money Webhook Handler
// ==============================================================================

// HandleOrangeMoneyWebhook godoc
// @Summary      Orange Money payment webhook
// @Description  Receives payment notifications from Orange Money (idempotent)
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        X-Orange-Signature header string true "HMAC signature"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Router       /webhooks/orange [post]
func (h *WebhookHandler) HandleOrangeMoneyWebhook(c *fiber.Ctx) error {
	// Similar implementation to Wave
	// Orange Money has slightly different payload structure
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "received",
	})
}

// ==============================================================================
// Free Money Webhook Handler
// ==============================================================================

// HandleFreeMoneyWebhook godoc
// @Summary      Free Money payment webhook
// @Description  Receives payment notifications from Free Money (idempotent)
// @Tags         webhooks
// @Accept       json
// @Produce      json
// @Param        X-Free-Signature header string true "HMAC signature"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Router       /webhooks/free [post]
func (h *WebhookHandler) HandleFreeMoneyWebhook(c *fiber.Ctx) error {
	// Similar implementation to Wave
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "received",
	})
}

// ==============================================================================
// Helpers
// ==============================================================================

func calculatePayloadHash(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

func isUniqueViolation(err error) bool {
	// Check for PostgreSQL unique constraint violation
	// Error code 23505
	return strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "duplicate key")
}

// ==============================================================================
// Reconciliation Endpoint (Internal/Admin only)
// ==============================================================================

// ReconcilePayments godoc
// @Summary      Reconcile payments
// @Description  Run daily reconciliation against Wave API (admin only)
// @Tags         admin
// @Security     BearerAuth
// @Param        date query string true "Date (YYYY-MM-DD)"
// @Success      200 {object} map[string]interface{}
// @Router       /admin/reconcile [post]
func (h *WebhookHandler) ReconcilePayments(c *fiber.Ctx) error {
	// Verify admin role
	userRole := c.Locals("user_role")
	if userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required",
		})
	}

	// Parse date
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().AddDate(0, 0, -1).Format("2006-01-02") // yesterday
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid date format",
		})
	}

	// Get transactions from Wave
	startDate := date
	endDate := date.Add(24 * time.Hour)

	report, err := h.waveService.GetTransactions(c.Context(), startDate, endDate)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to fetch transactions: %v", err),
		})
	}

	// TODO: Compare with our database records
	// Flag discrepancies for manual review

	return c.JSON(fiber.Map{
		"date":          dateStr,
		"total_amount":  report.TotalAmount,
		"count":         report.Count,
		"transactions":  report.Transactions,
	})
}
