// Package notify provides SMS and email sending ports for the admin console.
// Only Noop and log implementations exist today; real provider clients are
// wired in NewSMS / NewEmail when a provider is selected.
package notify

import (
	"context"
	"log/slog"
	"strings"
)

// Config is the notify section of the admin configuration. It is intentionally
// self-contained so this package does not depend on internal/conf.
type Config struct {
	// Production suppresses verification codes from logs.
	Production bool
	SMS        SMSConfig
	Email      EmailConfig
}

// SMSConfig describes the SMS provider.
type SMSConfig struct {
	Enabled      bool
	Provider     string
	Endpoint     string
	AccessKey    string
	AccessSecret string
	SignName     string
	TemplateCode string
}

// EmailConfig describes the SMTP provider.
type EmailConfig struct {
	Enabled  bool
	Provider string
	SMTPHost string
	SMTPPort int32
	Username string
	Password string
	From     string
}

// Message is one templated SMS message.
type Message struct {
	Mobile   string
	Template string
	Params   map[string]string
}

// Email is one email message.
type Email struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// SMSSender sends login codes and templated SMS messages.
type SMSSender interface {
	SendCode(ctx context.Context, mobile, code string) error
	Send(ctx context.Context, msg Message) error
}

// EmailSender sends emails.
type EmailSender interface {
	Send(ctx context.Context, email Email) error
}

// NoopSMS drops every SMS. It is returned when SMS is disabled or when the
// process runs in production without a configured provider.
type NoopSMS struct{}

func (NoopSMS) SendCode(context.Context, string, string) error { return nil }

func (NoopSMS) Send(context.Context, Message) error { return nil }

// NoopEmail drops every email.
type NoopEmail struct{}

func (NoopEmail) Send(context.Context, Email) error { return nil }

// logSMS writes SMS content to the log. Production never receives this
// implementation from NewSMS, and the flag keeps the code out of logs even if
// a caller constructs it directly.
type logSMS struct {
	log        *slog.Logger
	production bool
}

func (s *logSMS) SendCode(ctx context.Context, mobile, code string) error {
	if s.production {
		s.log.InfoContext(ctx, "sms login code sent", "mobile", MaskMobile(mobile))
		return nil
	}
	s.log.InfoContext(ctx, "sms login code (dev)", "mobile", MaskMobile(mobile), "code", code)
	return nil
}

func (s *logSMS) Send(ctx context.Context, msg Message) error {
	if s.production {
		s.log.InfoContext(ctx, "sms sent", "mobile", MaskMobile(msg.Mobile), "template", msg.Template)
		return nil
	}
	s.log.InfoContext(ctx, "sms sent (dev)",
		"mobile", MaskMobile(msg.Mobile), "template", msg.Template, "params", msg.Params)
	return nil
}

// logEmail writes an email summary to the log without message bodies.
type logEmail struct {
	log *slog.Logger
}

func (e *logEmail) Send(ctx context.Context, email Email) error {
	e.log.InfoContext(ctx, "email sent (dev)", "to", MaskEmail(email.To), "subject", email.Subject)
	return nil
}

// NewSMS builds the SMS sender. Production always gets a Noop so verification
// codes never reach the logs; development uses the log sender while no real
// provider client is wired.
func NewSMS(cfg Config, log *slog.Logger) SMSSender {
	if log == nil {
		log = slog.Default()
	}
	if cfg.Production {
		if cfg.SMS.Enabled {
			log.Warn("sms provider not implemented, messages are dropped", "provider", cfg.SMS.Provider)
		}
		return NoopSMS{}
	}
	if !cfg.SMS.Enabled {
		return NoopSMS{}
	}
	log.Warn("sms provider not implemented, falling back to log sender", "provider", cfg.SMS.Provider)
	return &logSMS{log: log, production: cfg.Production}
}

// NewEmail builds the email sender. Production gets a Noop until a real SMTP
// client is wired; development logs the message summary.
func NewEmail(cfg Config, log *slog.Logger) EmailSender {
	if log == nil {
		log = slog.Default()
	}
	if cfg.Production {
		if cfg.Email.Enabled {
			log.Warn("email provider not implemented, messages are dropped", "provider", cfg.Email.Provider)
		}
		return NoopEmail{}
	}
	if !cfg.Email.Enabled {
		return NoopEmail{}
	}
	log.Warn("email provider not implemented, falling back to log sender", "provider", cfg.Email.Provider)
	return &logEmail{log: log}
}

// MaskMobile masks a phone number, for example 138****1234.
func MaskMobile(mobile string) string {
	r := []rune(strings.TrimSpace(mobile))
	switch {
	case len(r) == 0:
		return ""
	case len(r) <= 4:
		return strings.Repeat("*", len(r))
	case len(r) <= 7:
		return string(r[:2]) + strings.Repeat("*", len(r)-2)
	default:
		return string(r[:3]) + "****" + string(r[len(r)-4:])
	}
}

// MaskEmail masks an email address, for example a***@b.com.
func MaskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		r := []rune(email)
		if len(r) <= 1 {
			return "***"
		}
		return string(r[:1]) + "***"
	}
	local := []rune(email[:at])
	if len(local) == 0 {
		return "***" + email[at:]
	}
	return string(local[:1]) + "***" + email[at:]
}
