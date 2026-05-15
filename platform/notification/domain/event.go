// Package domain holds the notification platform's domain types.
package domain

// EventType is the catalogue of notifiable events. The string value is what
// gets persisted on notification_logs.event_type and shown in admin views.
type EventType string

const (
	// EVENT_EMAIL_VERIFICATION_REQUESTED — emitted by SignupOwnerUseCase
	// when a fresh user needs to confirm their email address.
	EVENT_EMAIL_VERIFICATION_REQUESTED EventType = "EMAIL_VERIFICATION_REQUESTED"

	// EVENT_MEMBER_INVITED — emitted by InviteMemberUseCase. The invitee
	// receives an email with the redeem link.
	EVENT_MEMBER_INVITED EventType = "MEMBER_INVITED"

	// EVENT_INVITE_ACCEPTED — emitted by AcceptInviteUseCase. Notifies
	// the original inviter / owner that the invitee joined.
	EVENT_INVITE_ACCEPTED EventType = "INVITE_ACCEPTED"

	// EVENT_PASSWORD_RESET_REQUESTED — placeholder, wired in once a
	// ForgotPasswordUseCase ships (post-v0.1.0).
	EVENT_PASSWORD_RESET_REQUESTED EventType = "PASSWORD_RESET_REQUESTED"
)

// Channel is the delivery medium of a notification.
type Channel string

const (
	CHANNEL_EMAIL    Channel = "email"
	CHANNEL_SMS      Channel = "sms"
	CHANNEL_WHATSAPP Channel = "whatsapp"
	CHANNEL_IN_APP   Channel = "in_app"
	CHANNEL_PUSH     Channel = "push"
	CHANNEL_WEBHOOK  Channel = "webhook"
)

// Event is the input the Dispatcher consumes. It carries everything the
// channel-specific sender needs to render templates and address the
// recipient, but stays decoupled from the outbox transport type
// (kernel/events.Event) so the notification layer can be swapped for a
// non-outbox driver later.
type Event struct {
	Type        EventType
	RecipientID string         // user_id — empty when the recipient is identified by email only (pre-account)
	Email       string         // explicit recipient email (used when RecipientID is empty)
	TenantID    string         // empty for platform-wide notifications
	Locale      string         // "en" / "fr" — defaults to en when empty
	Payload     map[string]any // template data
}
