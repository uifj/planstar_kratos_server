package notify

import (
	"context"
	"errors"
	"net/smtp"
	"strings"
	"testing"

	"planstar/internal/platform/conf"
)

func testEmailCfg() conf.Email {
	return conf.Email{
		Enabled:  true,
		Provider: "smtp",
		SMTPHost: "smtp.example.com",
		SMTPPort: 465,
		Username: "user",
		Password: "pass",
		From:     "noreply@example.com",
	}
}

func TestSMTPEmailSendBuildsMessage(t *testing.T) {
	var gotAddr, gotFrom string
	var gotTo []string
	var gotMsg []byte
	s, err := newSMTPEmail(testEmailCfg(), nil)
	if err != nil {
		t.Fatalf("newSMTPEmail: %v", err)
	}
	s.send = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		gotAddr, gotFrom, gotTo, gotMsg = addr, from, to, msg
		return nil
	}
	if err := s.Send(context.Background(), Email{
		To: "u@example.com", Subject: "验证码", Text: "plain", HTML: "<b>html</b>",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotAddr != "smtp.example.com:465" || gotFrom != "noreply@example.com" {
		t.Errorf("addr/from = %q/%q", gotAddr, gotFrom)
	}
	if len(gotTo) != 1 || gotTo[0] != "u@example.com" {
		t.Errorf("to = %v", gotTo)
	}
	msg := string(gotMsg)
	for _, want := range []string{"multipart/alternative", "text/plain", "text/html", "<b>html</b>"} {
		if !strings.Contains(msg, want) {
			t.Errorf("MIME 缺少 %q: %s", want, msg)
		}
	}
}

func TestSMTPEmailSendPropagatesError(t *testing.T) {
	s, _ := newSMTPEmail(testEmailCfg(), nil)
	s.send = func(string, smtp.Auth, string, []string, []byte) error { return errors.New("smtp down") }
	if err := s.Send(context.Background(), Email{To: "u@example.com", Text: "x"}); err == nil {
		t.Fatal("Send 应透传 SMTP 错误")
	}
	if err := s.Send(context.Background(), Email{}); err == nil {
		t.Fatal("空收件人应返回错误")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Send(ctx, Email{To: "u@example.com"}); err == nil {
		t.Fatal("已取消的 ctx 应立即返回")
	}
}

func TestSMTPEmailConfigValidation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*conf.Email)
	}{
		{"缺 host", func(c *conf.Email) { c.SMTPHost = "" }},
		{"端口非法", func(c *conf.Email) { c.SMTPPort = 0 }},
		{"缺 from", func(c *conf.Email) { c.From = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testEmailCfg()
			tc.mutate(&cfg)
			if _, err := newSMTPEmail(cfg, nil); err == nil {
				t.Fatal("配置不完整应返回错误")
			}
		})
	}
}

func TestBuildMIMEPlainAndHeaderInjection(t *testing.T) {
	msg := string(buildMIME("from@example.com", "to@example.com", "hi\r\nBcc: attacker@x", "body", ""))
	// 过滤 CRLF 后注入文本只能留在 Subject 同一行内，无法形成独立邮件头。
	if strings.Contains(msg, "\r\nBcc") || strings.Contains(msg, "\nBcc") {
		t.Errorf("头部注入未过滤: %q", msg)
	}
	if !strings.Contains(msg, "Subject: hiBcc: attacker@x\r\n") {
		t.Errorf("Subject 应保留在同一行: %q", msg)
	}
	if !strings.Contains(msg, "Content-Type: text/plain") || !strings.Contains(msg, "body") {
		t.Errorf("纯文本 MIME 内容异常: %q", msg)
	}
}

func TestNewEmailSelection(t *testing.T) {
	cfg := testEmailCfg()
	if _, ok := NewEmail(cfg, "prod", nil).(*smtpEmail); !ok {
		t.Error("生产环境启用 smtp 时应返回 *smtpEmail")
	}
	cfg.Enabled = false
	if _, ok := NewEmail(cfg, "prod", nil).(NoopEmail); !ok {
		t.Error("生产环境未启用邮件时应返回 NoopEmail")
	}
	if _, ok := NewEmail(cfg, "local", nil).(*logEmail); !ok {
		t.Error("本地未启用邮件时应返回 *logEmail")
	}
}
