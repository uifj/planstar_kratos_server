package notify

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

// newBufferLogger returns a text logger writing into buf.
func newBufferLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewSMSDisabledReturnsNoop(t *testing.T) {
	cases := []struct {
		name       string
		production bool
	}{
		{name: "dev", production: false},
		{name: "production", production: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			sender := NewSMS(Config{Production: tc.production}, newBufferLogger(&buf))
			if _, ok := sender.(NoopSMS); !ok {
				t.Fatalf("NewSMS() = %T, want NoopSMS", sender)
			}
			if err := sender.SendCode(context.Background(), "13812345678", "123456"); err != nil {
				t.Fatalf("SendCode() error = %v, want nil", err)
			}
			if err := sender.Send(context.Background(), Message{Mobile: "13812345678", Template: "login"}); err != nil {
				t.Fatalf("Send() error = %v, want nil", err)
			}
			if strings.Contains(buf.String(), "123456") {
				t.Fatalf("noop sender wrote code to log: %q", buf.String())
			}
		})
	}
}

func TestNewSMSDevEnabledLogsCode(t *testing.T) {
	var buf bytes.Buffer
	sender := NewSMS(Config{SMS: SMSConfig{Enabled: true}}, newBufferLogger(&buf))
	if _, ok := sender.(*logSMS); !ok {
		t.Fatalf("NewSMS() = %T, want *logSMS", sender)
	}
	if err := sender.SendCode(context.Background(), "13812345678", "123456"); err != nil {
		t.Fatalf("SendCode() error = %v, want nil", err)
	}
	out := buf.String()
	if !strings.Contains(out, "123456") {
		t.Errorf("log output missing plaintext code: %q", out)
	}
	if !strings.Contains(out, "138****5678") {
		t.Errorf("log output missing masked mobile: %q", out)
	}
	if strings.Contains(out, "13812345678") {
		t.Errorf("log output leaked full mobile: %q", out)
	}
}

func TestNewSMSProductionEnabledNeverLogsCode(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufferLogger(&buf)
	sender := NewSMS(Config{Production: true, SMS: SMSConfig{Enabled: true}}, logger)
	if _, ok := sender.(NoopSMS); !ok {
		t.Fatalf("NewSMS() = %T, want NoopSMS in production", sender)
	}
	if err := sender.SendCode(context.Background(), "13812345678", "246810"); err != nil {
		t.Fatalf("SendCode() error = %v, want nil", err)
	}
	out := buf.String()
	if strings.Contains(out, "246810") {
		t.Errorf("production log leaked plaintext code: %q", out)
	}
	if strings.Contains(out, "13812345678") {
		t.Errorf("production log leaked full mobile: %q", out)
	}

	// logSMS guards itself even when constructed directly in production mode.
	buf.Reset()
	raw := &logSMS{log: logger, production: true}
	if err := raw.SendCode(context.Background(), "13812345678", "246810"); err != nil {
		t.Fatalf("SendCode() error = %v, want nil", err)
	}
	out = buf.String()
	if strings.Contains(out, "246810") {
		t.Errorf("production logSMS leaked plaintext code: %q", out)
	}
	if !strings.Contains(out, "138****5678") {
		t.Errorf("production logSMS missing masked mobile: %q", out)
	}
	if strings.Contains(out, "13812345678") {
		t.Errorf("production logSMS leaked full mobile: %q", out)
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

func isNoopEmail(sender EmailSender) bool {
	_, ok := sender.(NoopEmail)
	return ok
}

func TestNewEmail(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufferLogger(&buf)

	if got := NewEmail(Config{}, logger); !isNoopEmail(got) {
		t.Fatalf("NewEmail() = %T, want NoopEmail when disabled", got)
	}

	// Disabled senders stay silent.
	buf.Reset()
	noop := NewEmail(Config{}, logger)
	if err := noop.Send(context.Background(), Email{To: "alice@example.com", Subject: "hi"}); err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	if buf.Len() != 0 {
		t.Errorf("noop email sender wrote to log: %q", buf.String())
	}

	// Enabled development logs a masked address only.
	buf.Reset()
	dev := NewEmail(Config{Email: EmailConfig{Enabled: true}}, logger)
	if _, ok := dev.(*logEmail); !ok {
		t.Fatalf("NewEmail() = %T, want *logEmail", dev)
	}
	if err := dev.Send(context.Background(), Email{To: "alice@example.com", Subject: "hi"}); err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	out := buf.String()
	if !strings.Contains(out, "a***@example.com") {
		t.Errorf("log output missing masked email: %q", out)
	}
	if strings.Contains(out, "alice@example.com") {
		t.Errorf("log output leaked full email: %q", out)
	}

	// Production always returns NoopEmail, even when enabled.
	buf.Reset()
	prod := NewEmail(Config{Production: true, Email: EmailConfig{Enabled: true}}, logger)
	if _, ok := prod.(NoopEmail); !ok {
		t.Fatalf("NewEmail() = %T, want NoopEmail in production", prod)
	}
	if err := prod.Send(context.Background(), Email{To: "alice@example.com", Subject: "hi"}); err != nil {
		t.Fatalf("Send() error = %v, want nil", err)
	}
	if strings.Contains(buf.String(), "alice@example.com") {
		t.Errorf("production log leaked full email: %q", buf.String())
	}
}
