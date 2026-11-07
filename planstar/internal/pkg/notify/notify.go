// Package notify 提供短信与邮件的发送抽象。
// 短信支持 Noop、日志与阿里云号码认证（Dypnsapi），邮件暂只内置 Noop 与日志实现。
package notify

import (
	"context"
	"log/slog"
	"strings"

	"planstar/internal/conf"
)

// Message 是一条短信模板消息。
type Message struct {
	Mobile   string
	Template string
	Params   map[string]string
}

// Email 是一封邮件。
type Email struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// SMSSender 是短信发送端口。
type SMSSender interface {
	SendCode(ctx context.Context, mobile, code string) error
	Send(ctx context.Context, msg Message) error
}

// EmailSender 是邮件发送端口。
type EmailSender interface {
	Send(ctx context.Context, email Email) error
}

// NoopSMS 丢弃全部短信，用于未启用或生产环境（不落明文验证码）。
type NoopSMS struct{}

func (NoopSMS) SendCode(context.Context, string, string) error { return nil }
func (NoopSMS) Send(context.Context, Message) error            { return nil }

// NoopEmail 丢弃全部邮件。
type NoopEmail struct{}

func (NoopEmail) Send(context.Context, Email) error { return nil }

// logSMS 把短信内容写入日志；工厂保证生产环境不会返回该实现。
type logSMS struct {
	log *slog.Logger
}

func (s *logSMS) SendCode(ctx context.Context, mobile, code string) error {
	s.log.InfoContext(ctx, "sms login code (dev)", "mobile", MaskMobile(mobile), "code", code)
	return nil
}

func (s *logSMS) Send(ctx context.Context, msg Message) error {
	s.log.InfoContext(ctx, "sms sent (dev)",
		"mobile", MaskMobile(msg.Mobile), "template", msg.Template, "params", msg.Params)
	return nil
}

// logEmail 把邮件摘要写入日志；工厂保证生产环境不会返回该实现。
type logEmail struct {
	log *slog.Logger
}

func (e *logEmail) Send(ctx context.Context, email Email) error {
	e.log.InfoContext(ctx, "email sent (dev)", "to", MaskEmail(email.To), "subject", email.Subject)
	return nil
}

// NewSMS 按配置构造短信发送器：未启用时保持原有降级行为（生产 Noop、开发日志）；
// 启用 aliyun/aliyun-dypns 时构造真实客户端，其余 provider 仍是未知 provider 处理。
func NewSMS(cfg conf.SMS, env string, log *slog.Logger) (SMSSender, error) {
	if log == nil {
		log = slog.Default()
	}
	if !cfg.Enabled {
		if isProduction(env) {
			return NoopSMS{}, nil
		}
		return &logSMS{log: log}, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "aliyun", "aliyun-dypns":
		if err := validateSMSConfig(cfg); err != nil {
			return nil, err
		}
		api, err := newDypnsClient(cfg)
		if err != nil {
			return nil, err
		}
		log.Info("aliyun sms provider enabled", "endpoint", dypnsEndpoint(cfg))
		return &aliyunSMS{api: api, signName: cfg.SignName, templateCode: cfg.TemplateCode}, nil
	default:
		if isProduction(env) {
			log.Warn("sms provider not implemented, messages are dropped", "provider", cfg.Provider)
			return NoopSMS{}, nil
		}
		log.Warn("sms provider not implemented, falling back to log sender", "provider", cfg.Provider)
		return &logSMS{log: log}, nil
	}
}

// NewEmail 按配置构造邮件发送器：生产环境返回 Noop，开发环境返回日志实现。
func NewEmail(cfg conf.Email, env string, log *slog.Logger) EmailSender {
	if log == nil {
		log = slog.Default()
	}
	if isProduction(env) {
		if cfg.Enabled {
			log.Warn("email provider not implemented, messages are dropped", "provider", cfg.Provider)
		}
		return NoopEmail{}
	}
	if cfg.Enabled {
		log.Warn("email provider not implemented, falling back to log sender", "provider", cfg.Provider)
	}
	return &logEmail{log: log}
}

// MaskMobile 脱敏手机号，例如 138****1234。
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

// MaskEmail 脱敏邮箱，例如 a***@b.com。
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

// isProduction 同时接受 prod 与 production，避免部署环境命名差异导致日志泄露验证码。
func isProduction(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}
