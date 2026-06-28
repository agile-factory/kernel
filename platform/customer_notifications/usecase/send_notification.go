package usecase

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SendCustomerNotificationUseCase dispatches a notification to an end-customer.
// It resolves the end-customer contact, loads the template (tenant override or
// default embedded), renders it, and calls EmailSender or SMSSender.
type SendCustomerNotificationUseCase struct {
	db               *pgxpool.Pool
	logRepo          domain.NotificationLogRepository
	templateRepo     domain.CustomerTemplateRepository
	customerResolver domain.EndCustomerResolver
	brandResolver    domain.BrandResolver
	emailSender      domain.EmailSender
	smsSender        domain.SMSSender
	defaultTemplates map[string]*template.Template // keyed by "key.channel.locale"
}

// NewSendCustomerNotificationUseCase wires the use case.
// defaultTemplates must be pre-parsed at boot; map is read-only after construction.
func NewSendCustomerNotificationUseCase(
	db *pgxpool.Pool,
	logRepo domain.NotificationLogRepository,
	templateRepo domain.CustomerTemplateRepository,
	customerResolver domain.EndCustomerResolver,
	brandResolver domain.BrandResolver,
	emailSender domain.EmailSender,
	smsSender domain.SMSSender,
	defaultTemplates map[string]*template.Template,
) *SendCustomerNotificationUseCase {
	return &SendCustomerNotificationUseCase{
		db:               db,
		logRepo:          logRepo,
		templateRepo:     templateRepo,
		customerResolver: customerResolver,
		brandResolver:    brandResolver,
		emailSender:      emailSender,
		smsSender:        smsSender,
		defaultTemplates: defaultTemplates,
	}
}

// SendCustomerNotificationInput carries the notification request.
type SendCustomerNotificationInput struct {
	TenantID      string
	EndCustomerID string
	TemplateKey   string
	Locale        string           // "" → "fr"
	Channel       domain.Channel   // "" → determined by tenant_settings / contact availability
	ExtraData     map[string]string // merged into TemplateData (OTPCode, ResetLink, etc.)
}

// Execute dispatches the notification.
func (uc *SendCustomerNotificationUseCase) Execute(ctx context.Context, in SendCustomerNotificationInput) error {
	locale := in.Locale
	if locale == "" {
		locale = "fr"
	}

	contact, err := uc.customerResolver.GetContact(ctx, in.EndCustomerID, in.TenantID)
	if err != nil {
		return err
	}

	brand, err := uc.brandResolver.GetBrand(ctx, in.TenantID)
	if err != nil {
		brand = &domain.BrandData{} // non-fatal: send without branding
	}

	ch := in.Channel
	if ch == "" {
		// Default: prefer EMAIL; use SMS when email is absent and phone is available.
		if contact.Email != "" {
			ch = domain.ChannelEmail
		} else if contact.Phone != "" {
			ch = domain.ChannelSMS
		} else {
			ch = domain.ChannelEmail
		}
	}

	data := domain.TemplateData{
		CustomerFirstName:  contact.FirstName,
		CustomerLastName:   contact.LastName,
		TenantAppName:      brand.AppName,
		TenantLogoURL:      brand.LogoURL,
		TenantPrimaryColor: brand.PrimaryColor,
		SupportEmail:       brand.SupportEmail,
	}
	for k, v := range in.ExtraData {
		switch k {
		case "otp_code":
			data.OTPCode = v
		case "reset_link":
			data.ResetLink = v
		case "invoice_amount":
			data.InvoiceAmount = v
		case "invoice_currency":
			data.InvoiceCurrency = v
		}
	}

	tmplKey := templateKey(in.TemplateKey, ch, locale)

	// Try tenant override first.
	subject, body, err := uc.renderTemplate(ctx, in.TenantID, in.TemplateKey, ch, locale, tmplKey, data)
	if err != nil {
		return fmt.Errorf("render template %s: %w", tmplKey, err)
	}

	log := &domain.CustomerNotificationLog{
		TenantID:      in.TenantID,
		EndCustomerID: in.EndCustomerID,
		Channel:       ch,
		TemplateKey:   in.TemplateKey,
		Status:        domain.StatusPending,
	}

	recipient := contact.Email
	if ch == domain.ChannelSMS {
		recipient = contact.Phone
	}
	log.Recipient = recipient

	if err := uc.logRepo.Create(ctx, log); err != nil {
		return err
	}

	now := time.Now()
	var externalID, failureReason string
	var sentAt *time.Time
	status := domain.StatusSent

	switch ch {
	case domain.ChannelEmail:
		if sendErr := uc.emailSender.Send(ctx, recipient, subject, body); sendErr != nil {
			status = domain.StatusFailed
			failureReason = sendErr.Error()
		} else {
			sentAt = &now
		}
	case domain.ChannelSMS:
		extID, sendErr := uc.smsSender.Send(ctx, recipient, body)
		if sendErr != nil {
			status = domain.StatusFailed
			failureReason = sendErr.Error()
		} else {
			externalID = extID
			sentAt = &now
		}
	}

	return uc.logRepo.UpdateStatus(ctx, log.ID, status, externalID, failureReason, sentAt)
}

// renderTemplate loads the template (tenant override then default) and renders it.
// Returns (subject, body, error).
func (uc *SendCustomerNotificationUseCase) renderTemplate(
	ctx context.Context,
	tenantID, key string,
	ch domain.Channel,
	locale, tmplKey string,
	data domain.TemplateData,
) (subject, body string, err error) {
	override, lookupErr := uc.templateRepo.Find(ctx, tenantID, key, ch, locale)
	if lookupErr == nil {
		// Use tenant override.
		if override.SubjectTmpl != "" {
			s, e := renderString(override.SubjectTmpl, data)
			if e != nil {
				return "", "", e
			}
			subject = s
		}
		b, e := renderString(override.BodyTmpl, data)
		if e != nil {
			return "", "", e
		}
		return subject, b, nil
	}

	// Fall back to default embedded template.
	tmpl, ok := uc.defaultTemplates[tmplKey]
	if !ok {
		return "", "", fmt.Errorf("no default template for %s", tmplKey)
	}
	var buf bytes.Buffer
	if e := tmpl.Execute(&buf, data); e != nil {
		return "", "", e
	}
	return subject, buf.String(), nil
}

func templateKey(key string, ch domain.Channel, locale string) string {
	ext := "email.html"
	if ch == domain.ChannelSMS {
		ext = "sms.txt"
	}
	return locale + "/" + key + "." + ext
}

func renderString(tmplStr string, data domain.TemplateData) (string, error) {
	t, err := template.New("").Parse(tmplStr)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
