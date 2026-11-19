package notify

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"planstar/internal/conf"
)

func newBufferLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func isNoopSMS(sender SMSSender) bool {
	_, ok := sender.(NoopSMS)
	return ok
}

func isLogSMS(sender SMSSender) bool {
	_, ok := sender.(*logSMS)
	return ok
}

// 生产级环境必须 fail-closed：未启用短信或未指定 aliyun provider 一律启动失败，
// 不允许静默丢弃验证码（Noop）导致线上用户收不到登录码。
func TestNewSMSProductionFailsClosed(t *testing.T) {
	for _, env := range []string{"prod", "production", " PROD ", "staging", "docker", ""} {
		for _, enabled := range []bool{false, true} {
			var buf bytes.Buffer
			sender, err := NewSMS(conf.SMS{Enabled: enabled}, env, newBufferLogger(&buf))
			if err == nil {
				t.Fatalf("NewSMS(env=%q, enabled=%v) = %T, want error", env, enabled, sender)
			}
			if sender != nil {
				t.Fatalf("NewSMS(env=%q) returned non-nil sender on error", env)
			}
		}
	}
}

// 本地版本保留 log sender，但不再打印验证码明文（阿里手册日志脱敏规约）。
func TestNewSMSDevUsesLogSenderWithoutPlaintextCode(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		var buf bytes.Buffer
		sender, err := NewSMS(conf.SMS{Enabled: enabled}, "dev", newBufferLogger(&buf))
		if err != nil {
			t.Fatalf("NewSMS(enabled=%v) error = %v", enabled, err)
		}
		if !isLogSMS(sender) {
			t.Fatalf("NewSMS(enabled=%v) = %T, want *logSMS", enabled, sender)
		}
		if err := sender.SendCode(context.Background(), "13812345678", "123456"); err != nil {
			t.Fatalf("SendCode() error = %v, want nil", err)
		}
		out := buf.String()
		if strings.Contains(out, "123456") {
			t.Errorf("log leaked plaintext code: %q", out)
		}
		if !strings.Contains(out, "code_fingerprint=") {
			t.Errorf("log missing code fingerprint: %q", out)
		}
		if !strings.Contains(out, "138****5678") {
			t.Errorf("dev log missing masked mobile: %q", out)
		}
		if strings.Contains(out, "13812345678") {
			t.Errorf("dev log leaked full mobile: %q", out)
		}
	}
}

func TestLogSMSSendMasksMobile(t *testing.T) {
	var buf bytes.Buffer
	sender := &logSMS{log: newBufferLogger(&buf)}
	err := sender.Send(context.Background(), Message{
		Mobile:   "13812345678",
		Template: "login",
		Params:   map[string]string{"code": "123456"},
	})
	if err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	out := buf.String()
	if !strings.Contains(out, "138****5678") {
		t.Errorf("log missing masked mobile: %q", out)
	}
	if strings.Contains(out, "13812345678") {
		t.Errorf("log leaked full mobile: %q", out)
	}
}

func isNoopEmail(sender EmailSender) bool {
	_, ok := sender.(NoopEmail)
	return ok
}

func TestNewEmail(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufferLogger(&buf)

	if got := NewEmail(conf.Email{}, "prod", logger); !isNoopEmail(got) {
		t.Fatalf("NewEmail(prod) = %T, want NoopEmail", got)
	}

	buf.Reset()
	dev := NewEmail(conf.Email{Enabled: true}, "dev", logger)
	if _, ok := dev.(*logEmail); !ok {
		t.Fatalf("NewEmail(dev) = %T, want *logEmail", dev)
	}
	if err := dev.Send(context.Background(), Email{To: "alice@example.com", Subject: "hi"}); err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	out := buf.String()
	if !strings.Contains(out, "a***@example.com") {
		t.Errorf("dev log missing masked email: %q", out)
	}
	if strings.Contains(out, "alice@example.com") {
		t.Errorf("dev log leaked full email: %q", out)
	}

	buf.Reset()
	prod := NewEmail(conf.Email{Enabled: true}, "production", logger)
	if _, ok := prod.(NoopEmail); !ok {
		t.Fatalf("NewEmail(production) = %T, want NoopEmail", prod)
	}
	if err := prod.Send(context.Background(), Email{To: "alice@example.com", Subject: "hi"}); err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	if strings.Contains(buf.String(), "alice@example.com") {
		t.Errorf("production log leaked full email: %q", buf.String())
	}
}

func TestMaskMobile(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "   ", want: ""},
		{in: "138", want: "***"},
		{in: "1381", want: "****"},
		{in: "1381234", want: "13*****"},
		{in: "13812345678", want: "138****5678"},
		{in: " 13812345678 ", want: "138****5678"},
	}
	for _, tc := range cases {
		if got := MaskMobile(tc.in); got != tc.want {
			t.Errorf("MaskMobile(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestMaskEmail(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "alice@example.com", want: "a***@example.com"},
		{in: "  alice@example.com  ", want: "a***@example.com"},
		{in: "noatsign", want: "n***"},
		{in: "", want: "***"},
		// at <= 0 falls through to the "no @" branch.
		{in: "@example.com", want: "@***"},
	}
	for _, tc := range cases {
		if got := MaskEmail(tc.in); got != tc.want {
			t.Errorf("MaskEmail(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsProduction(t *testing.T) {
	cases := []struct {
		env  string
		want bool
	}{
		{env: "prod", want: true},
		{env: "production", want: true},
		{env: " PROD ", want: true},
		{env: "Production", want: true},
		{env: "dev", want: false},
		{env: "local", want: false},
		{env: "test", want: false},
		{env: "", want: true},        // 默认拒绝：未声明环境按生产处理
		{env: "staging", want: true}, // 类生产环境同样受生产约束
		{env: "docker", want: true},
	}
	for _, tc := range cases {
		if got := isProduction(tc.env); got != tc.want {
			t.Errorf("isProduction(%q) = %v, want %v", tc.env, got, tc.want)
		}
	}
}
