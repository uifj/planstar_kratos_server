// Package notify 提供短信与邮件的发送抽象。
// 短信支持 Noop、日志与阿里云号码认证（Dypnsapi），邮件暂只内置 Noop 与日志实现。
package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

// logSMS 是本地版本专用的短信实现。
// 依据《阿里巴巴Java开发手册（黄山版）》日志规约「用户敏感信息必须脱敏」，
// 任何环境都不再打印验证码明文，只输出不可逆指纹供对账；本地登录请使用演示账号测试口令。
type logSMS struct {
	log *slog.Logger
}

// codeFingerprint 返回验证码的 SHA-256 前 8 字节指纹（不可逆，仅用于排查是否同一码）。
func codeFingerprint(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:8])
}

func (s *logSMS) SendCode(ctx context.Context, mobile, code string) error {
	s.log.InfoContext(ctx, "sms login code issued (local)",
		"mobile", MaskMobile(mobile), "code_fingerprint", codeFingerprint(code))
	return nil
}

func (s *logSMS) Send(ctx context.Context, msg Message) error {
	// 模板参数可能携带验证码等敏感值，只记录参数名不记录值。
	keys := make([]string, 0, len(msg.Params))
	for k := range msg.Params {
		keys = append(keys, k)
	}
	s.log.InfoContext(ctx, "sms sent (local)",
		"mobile", MaskMobile(msg.Mobile), "template", msg.Template, "param_keys", keys)
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

// NewSMS 按配置构造短信发送器。
// 生产级环境强制启用已购买的阿里云 Dypnsapi：任何「未启用 / 未知 provider /
// 配置缺失」都在启动阶段失败关闭，杜绝静默丢弃验证码导致用户无法登录。
func NewSMS(cfg conf.SMS, env string, log *slog.Logger) (SMSSender, error) {
	if log == nil {
		log = slog.Default()
	}
	production := isProduction(env)
	if !cfg.Enabled {
		if production {
			return nil, fmt.Errorf("生产环境必须启用短信服务：PLANSTAR_NOTIFY_SMS_ENABLED=true 且 PROVIDER=aliyun")
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
		if production {
			return nil, fmt.Errorf("生产环境短信 provider 必须是 aliyun，当前为 %q", cfg.Provider)
		}
		log.Warn("sms provider not implemented, falling back to local log sender", "provider", cfg.Provider)
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

// isProduction 统一委托给 conf 的环境判定（默认拒绝）。
// 除 dev/local/test 外，staging、docker、拼写变体与空值一律按生产级约束处理。
func isProduction(env string) bool {
	return conf.IsProductionLikeEnv(env)
}
