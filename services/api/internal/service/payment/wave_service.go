package payment

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
	"time"

	"github.com/google/uuid"
)

// WaveService handles Wave Mobile Money payments (Senegal)
type WaveService struct {
	apiKey        string
	apiURL        string
	webhookSecret string
	httpClient    *http.Client
}

func NewWaveService(apiKey, apiURL, webhookSecret string) *WaveService {
	return &WaveService{
		apiKey:        apiKey,
		apiURL:        apiURL,
		webhookSecret: webhookSecret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// WavePaymentRequest represents a payment initiation request
type WavePaymentRequest struct {
	Amount      int64  `json:"amount"`       // in smallest currency unit (XOF cents)
	Currency    string `json:"currency"`     // "XOF"
	PhoneNumber string `json:"phone_number"` // +221XXXXXXXXX
	Description string `json:"description"`
	Reference   string `json:"reference"` // our booking_id
	ReturnURL   string `json:"return_url"`
	WebhookURL  string `json:"webhook_url"`
}

// WavePaymentResponse from Wave API
type WavePaymentResponse struct {
	ID             string    `json:"id"`
	Status         string    `json:"status"` // pending, success, failed
	TransactionID  string    `json:"transaction_id"`
	PaymentURL     string    `json:"payment_url"` // redirect user here
	QRCodeURL      string    `json:"qr_code_url"`
	ExpiresAt      time.Time `json:"expires_at"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	PhoneNumber    string    `json:"phone_number"`
	Reference      string    `json:"reference"`
	ErrorCode      string    `json:"error_code,omitempty"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

// WaveWebhook represents incoming webhook payload
type WaveWebhook struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transaction_id"`
	Reference     string    `json:"reference"` // our booking_id
	Status        string    `json:"status"`
	Amount        int64     `json:"amount"`
	Currency      string    `json:"currency"`
	PhoneNumber   string    `json:"phone_number"`
	Timestamp     time.Time `json:"timestamp"`
	ErrorCode     string    `json:"error_code,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// InitiatePayment creates a new Wave payment
func (s *WaveService) InitiatePayment(ctx context.Context, req *WavePaymentRequest) (*WavePaymentResponse, error) {
	// Validate
	if req.Amount <= 0 {
		return nil, fmt.Errorf("invalid amount: %d", req.Amount)
	}
	if req.Currency == "" {
		req.Currency = "XOF" // Senegal default
	}

	// Prepare request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.apiURL+"/v1/payments", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("X-Idempotency-Key", uuid.New().String())

	// Execute
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("wave api error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var waveResp WavePaymentResponse
	if err := json.Unmarshal(body, &waveResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &waveResp, nil
}

// GetPaymentStatus retrieves payment status
func (s *WaveService) GetPaymentStatus(ctx context.Context, transactionID string) (*WavePaymentResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", s.apiURL+"/v1/payments/"+transactionID, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wave api error (status %d): %s", resp.StatusCode, string(body))
	}

	var waveResp WavePaymentResponse
	if err := json.Unmarshal(body, &waveResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &waveResp, nil
}

// VerifyWebhookSignature validates incoming webhook authenticity
func (s *WaveService) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

// ParseWebhook parses and validates webhook payload
func (s *WaveService) ParseWebhook(payload []byte, signature string) (*WaveWebhook, error) {
	// Verify signature
	if !s.VerifyWebhookSignature(payload, signature) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	// Parse payload
	var webhook WaveWebhook
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("unmarshal webhook: %w", err)
	}

	return &webhook, nil
}

// PayloadHash generates SHA256 hash of raw payload for idempotency
func PayloadHash(payload []byte) string {
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:])
}

// ==============================================================================
// Additional Wave-specific features
// ==============================================================================

// WaveRefund initiates a refund
type WaveRefundRequest struct {
	TransactionID string `json:"transaction_id"`
	Amount        int64  `json:"amount"`
	Reason        string `json:"reason"`
}

func (s *WaveService) RefundPayment(ctx context.Context, req *WaveRefundRequest) error {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.apiURL+"/v1/refunds", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refund failed (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// ==============================================================================
// Payout to landlord
// ==============================================================================

// WavePayoutRequest for transferring money to landlord
type WavePayoutRequest struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	PhoneNumber string `json:"phone_number"`
	Description string `json:"description"`
	Reference   string `json:"reference"` // payout_id
}

func (s *WaveService) TransferPayout(ctx context.Context, req *WavePayoutRequest) (*WavePaymentResponse, error) {
	if req.Currency == "" {
		req.Currency = "XOF"
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.apiURL+"/v1/transfers", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)
	httpReq.Header.Set("X-Idempotency-Key", req.Reference) // use payout_id for idempotency

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("transfer failed (status %d): %s", resp.StatusCode, string(body))
	}

	var waveResp WavePaymentResponse
	if err := json.Unmarshal(body, &waveResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &waveResp, nil
}

// ==============================================================================
// Reconciliation helpers
// ==============================================================================

// ReconciliationReport for daily reconciliation
type ReconciliationReport struct {
	Date         time.Time
	Transactions []WavePaymentResponse
	TotalAmount  int64
	Count        int
}

// GetTransactions retrieves transactions for a date range (for reconciliation)
func (s *WaveService) GetTransactions(ctx context.Context, startDate, endDate time.Time) (*ReconciliationReport, error) {
	url := fmt.Sprintf("%s/v1/transactions?start_date=%s&end_date=%s",
		s.apiURL,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
	)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(body))
	}

	var transactions []WavePaymentResponse
	if err := json.Unmarshal(body, &transactions); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Calculate total
	var total int64
	for _, tx := range transactions {
		if tx.Status == "success" {
			total += tx.Amount
		}
	}

	return &ReconciliationReport{
		Date:         startDate,
		Transactions: transactions,
		TotalAmount:  total,
		Count:        len(transactions),
	}, nil
}
