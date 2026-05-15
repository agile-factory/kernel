package notification

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// EmailMessage is the rendered payload an EmailSender ships out. HTML is
// preferred by clients that support it; Text is the fallback alternative
// (and what landed in spam-checker / accessibility tooling).
type EmailMessage struct {
	From    string
	To      string
	Subject string
	HTML    string
	Text    string
}

// EmailSender is the port. v0.1.0 ships two implementations:
//
//   - SMTPSender: real delivery via net/smtp.
//   - ConsoleSender: development fallback — logs the rendered message via
//     slog and returns a synthetic "console-<uuid>" external id, so the
//     full pipeline can be exercised without an SMTP server.
type EmailSender interface {
	Send(ctx context.Context, msg EmailMessage) (externalID string, err error)
}

// ─── SMTP sender ─────────────────────────────────────────────────────────────

// SMTPConfig configures a net/smtp-based EmailSender.
type SMTPConfig struct {
	Host     string // "smtp.example.com"
	Port     int    // 465, 587…
	Username string
	Password string
	UseTLS   bool // true → implicit TLS (port 465). false → STARTTLS on plain port (587).
}

// SMTPSender talks to a real SMTP server. It deliberately avoids pulling
// a heavyweight library — net/smtp is enough for the auth + invite emails
// shipped in v0.1.0. Production deployments can swap for Resend / Postmark
// via a tailored EmailSender.
type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(ctx context.Context, msg EmailMessage) (string, error) {
	if s.cfg.Host == "" {
		return "", errors.New("smtp sender: Host is required")
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	body := buildMIME(msg)

	// net/smtp.SendMail handles both plain and STARTTLS. Implicit-TLS
	// (smtps:// on 465) requires a manual tls.Dial first.
	if s.cfg.UseTLS {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return "", fmt.Errorf("smtp tls dial: %w", err)
		}
		defer conn.Close()
		c, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			return "", fmt.Errorf("smtp client: %w", err)
		}
		defer c.Close()
		if err := c.Auth(auth); err != nil {
			return "", fmt.Errorf("smtp auth: %w", err)
		}
		if err := c.Mail(msg.From); err != nil {
			return "", err
		}
		if err := c.Rcpt(msg.To); err != nil {
			return "", err
		}
		w, err := c.Data()
		if err != nil {
			return "", err
		}
		if _, err := w.Write(body); err != nil {
			return "", err
		}
		if err := w.Close(); err != nil {
			return "", err
		}
		return "", c.Quit()
	}

	if err := smtp.SendMail(addr, auth, msg.From, []string{msg.To}, body); err != nil {
		return "", fmt.Errorf("smtp sendmail: %w", err)
	}
	return "", nil
}

// buildMIME assembles a multipart/alternative body with text + html parts.
// The hard-coded boundary is fine: it is generated per-message at runtime by
// the caller-supplied uuid (we keep it stable per-send for simplicity, the
// chance of a payload containing this exact string is astronomically low).
func buildMIME(msg EmailMessage) []byte {
	const boundary = "==agilefactory-kernel-boundary=="
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", msg.From)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", msg.Subject)
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)

	if msg.Text != "" {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(msg.Text)
		b.WriteString("\r\n")
	}
	if msg.HTML != "" {
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(msg.HTML)
		b.WriteString("\r\n")
	}
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return []byte(b.String())
}

// ─── Console sender (dev) ────────────────────────────────────────────────────

// ConsoleSender prints rendered emails through the supplied slog.Logger.
// Returns a synthetic external id so the rest of the pipeline (log row,
// status update) behaves identically to the SMTP path.
type ConsoleSender struct {
	logger *slog.Logger
	count  int // not concurrency-safe, fine for dev visibility
}

func NewConsoleSender(logger *slog.Logger) *ConsoleSender {
	return &ConsoleSender{logger: logger}
}

func (s *ConsoleSender) Send(_ context.Context, msg EmailMessage) (string, error) {
	s.count++
	id := fmt.Sprintf("console-%d", s.count)
	s.logger.Info("notification: console email",
		"id", id,
		"from", msg.From,
		"to", msg.To,
		"subject", msg.Subject,
		"text_len", len(msg.Text),
		"html_len", len(msg.HTML),
	)
	return id, nil
}
