package notify

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strconv"
	"strings"

	"planstar/internal/platform/conf"
)

// sendMailFunc 便于测试注入，生产使用 smtp.SendMail。
type sendMailFunc func(addr string, a smtp.Auth, from string, to []string, msg []byte) error

// smtpEmail 是基于标准库 net/smtp 的邮件实现（OPEN_ITEMS C1）。
// 凭据只来自 conf.Email（由 PLANSTAR_NOTIFY_EMAIL_* 注入），不写入配置文件。
type smtpEmail struct {
	cfg  conf.Email
	log  *slog.Logger
	send sendMailFunc
}

func newSMTPEmail(cfg conf.Email, log *slog.Logger) (*smtpEmail, error) {
	if strings.TrimSpace(cfg.SMTPHost) == "" || cfg.SMTPPort <= 0 || strings.TrimSpace(cfg.From) == "" {
		return nil, fmt.Errorf("邮件配置不完整：需要 smtp_host / smtp_port / from")
	}
	if log == nil {
		log = slog.Default()
	}
	return &smtpEmail{cfg: cfg, log: log, send: smtp.SendMail}, nil
}

// Send 发送一封纯文本或 text/plain + text/html 邮件。
// net/smtp.SendMail 不接受 context，超时由调用方在业务层用 context 控制。
func (s *smtpEmail) Send(ctx context.Context, email Email) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	to := strings.TrimSpace(email.To)
	if to == "" {
		return fmt.Errorf("邮件收件人为空")
	}
	addr := net.JoinHostPort(s.cfg.SMTPHost, strconv.Itoa(s.cfg.SMTPPort))
	var auth smtp.Auth
	if strings.TrimSpace(s.cfg.Username) != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.SMTPHost)
	}
	if err := s.send(addr, auth, s.cfg.From, []string{to}, buildMIME(s.cfg.From, to, email.Subject, email.Text, email.HTML)); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// buildMIME 构造最小可用的 MIME 消息；头部统一做 CRLF 注入过滤。
func buildMIME(from, to, subject, text, html string) []byte {
	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(from) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(to) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	if strings.TrimSpace(html) == "" {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(text)
		return []byte(b.String())
	}
	const boundary = "ps-mail-boundary"
	b.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n\r\n")
	b.WriteString("--" + boundary + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + text + "\r\n")
	b.WriteString("--" + boundary + "\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n" + html + "\r\n")
	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

// sanitizeHeader 移除 CRLF，防止通过 Subject/To 注入额外邮件头。
func sanitizeHeader(v string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(strings.TrimSpace(v))
}
