// Package domain holds the customer_notifications platform's types and contracts.
package domain

import "time"

// Channel identifies the delivery channel for a notification.
type Channel string

const (
	ChannelEmail Channel = "EMAIL"
	ChannelSMS   Channel = "SMS"
)

// DeliveryStatus tracks the lifecycle of a notification attempt.
type DeliveryStatus string

const (
	StatusPending DeliveryStatus = "PENDING"
	StatusSent    DeliveryStatus = "SENT"
	StatusFailed  DeliveryStatus = "FAILED"
	StatusBounced DeliveryStatus = "BOUNCED"
)

// CustomerNotificationLog records a dispatch attempt.
// Body is intentionally omitted (Lucas amendment) — OTP codes and payment
// amounts must not be stored in plaintext in the database.
type CustomerNotificationLog struct {
	ID             string
	TenantID       string
	EndCustomerID  string
	Channel        Channel
	Recipient      string        // email address or E.164 phone number
	TemplateKey    string        // e.g. "otp_reset", "welcome", "invoice_paid"
	Status         DeliveryStatus
	ExternalID     string        // ID returned by the provider
	FailureReason  string
	SentAt         *time.Time
	CreatedAt      time.Time
}
