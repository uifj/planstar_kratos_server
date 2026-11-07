package notify

import (
	"context"
	"errors"
	"strings"
	"testing"

	"planstar/internal/conf"
)

type fakeVerifyAPI struct {
	calls int
	got   smsVerifyCodeRequest
	code  string
	msg   string
	err   error
}

func (f *fakeVerifyAPI) SendSmsVerifyCode(_ context.Context, req smsVerifyCodeRequest) (string, string, error) {
	f.calls++
	f.got = req
	return f.code, f.msg, f.err
}

func TestAliyunSMSSendCodeMapsRequest(t *testing.T) {
	api := &fakeVerifyAPI{code: "OK", msg: "Success"}
	sender := &aliyunSMS{api: api, signName: "Planstar", templateCode: "SMS_001"}

	if err := sender.SendCode(context.Background(), "13812345678", "246810"); err != nil {
		t.Fatalf("SendCode() error = %v, want nil", err)
	}
	if api.calls != 1 {
		t.Fatalf("SendSmsVerifyCode calls = %d, want 1", api.calls)
	}
	got := api.got
	if got.Mobile != "13812345678" {
		t.Errorf("Mobile = %q", got.Mobile)
	}
	if got.SignName != "Planstar" {
		t.Errorf("SignName = %q", got.SignName)
	}
	if got.TemplateCode != "SMS_001" {
		t.Errorf("TemplateCode = %q", got.TemplateCode)
	}
	if got.TemplateParam != `{"code":"246810"}` {
		t.Errorf("TemplateParam = %q", got.TemplateParam)
	}
	if got.CodeLength != 6 {
		t.Errorf("CodeLength = %d, want 6", got.CodeLength)
	}
	if got.ValidTime != 300 {
		t.Errorf("ValidTime = %d, want 300", got.ValidTime)
	}
}

func TestAliyunSMSSendCodePropagatesResultCode(t *testing.T) {
	api := &fakeVerifyAPI{code: "isv.BUSINESS_LIMIT_CONTROL", msg: "触发流控"}
	sender := &aliyunSMS{api: api, signName: "Planstar", templateCode: "SMS_001"}

	err := sender.SendCode(context.Background(), "13812345678", "246810")
	if err == nil {
		t.Fatal("SendCode() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "isv.BUSINESS_LIMIT_CONTROL") {
		t.Errorf("error missing response code: %v", err)
	}
	if !strings.Contains(err.Error(), "触发流控") {
		t.Errorf("error missing response message: %v", err)
	}
	if strings.Contains(err.Error(), "246810") {
		t.Errorf("error leaked plaintext verification code: %v", err)
	}
}

func TestAliyunSMSSendCodePropagatesAPIError(t *testing.T) {
	api := &fakeVerifyAPI{err: errors.New("network unreachable")}
	sender := &aliyunSMS{api: api, signName: "Planstar", templateCode: "SMS_001"}

	err := sender.SendCode(context.Background(), "13812345678", "246810")
	if err == nil || !strings.Contains(err.Error(), "network unreachable") {
		t.Fatalf("SendCode() error = %v, want wrapped API error", err)
	}
}

func TestAliyunSMSSendRejectsFreeText(t *testing.T) {
	sender := &aliyunSMS{api: &fakeVerifyAPI{code: "OK"}}
	err := sender.Send(context.Background(), Message{Mobile: "13812345678"})
	if err == nil || !strings.Contains(err.Error(), "verification codes only") {
		t.Fatalf("Send() error = %v, want explicit unsupported error", err)
	}
}

func TestNewSMSAliyunValidatesConfig(t *testing.T) {
	valid := conf.SMS{
		Enabled:      true,
		Provider:     "aliyun",
		AccessKey:    "ak",
		AccessSecret: "sk",
		SignName:     "Planstar",
		TemplateCode: "SMS_001",
	}
	cases := []struct {
		name  string
		blank func(*conf.SMS)
		field string
	}{
		{name: "access key", blank: func(c *conf.SMS) { c.AccessKey = " " }, field: "notify.sms.access_key"},
		{name: "access secret", blank: func(c *conf.SMS) { c.AccessSecret = "" }, field: "notify.sms.access_secret"},
		{name: "sign name", blank: func(c *conf.SMS) { c.SignName = "" }, field: "notify.sms.sign_name"},
		{name: "template code", blank: func(c *conf.SMS) { c.TemplateCode = "" }, field: "notify.sms.template_code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := valid
			tc.blank(&cfg)
			sender, err := NewSMS(cfg, "prod", nil)
			if err == nil {
				t.Fatalf("NewSMS() = %T, want config error", sender)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("error = %v, want field %q", err, tc.field)
			}
		})
	}
}

func TestNewSMSAliyunProviderAliases(t *testing.T) {
	base := conf.SMS{
		Enabled:      true,
		AccessKey:    "ak",
		AccessSecret: "sk",
		SignName:     "Planstar",
		TemplateCode: "SMS_001",
	}
	for _, provider := range []string{"aliyun", "aliyun-dypns", " Aliyun-Dypns "} {
		cfg := base
		cfg.Provider = provider
		sender, err := NewSMS(cfg, "prod", nil)
		if err != nil {
			t.Fatalf("NewSMS(provider=%q) error = %v", provider, err)
		}
		sms, ok := sender.(*aliyunSMS)
		if !ok {
			t.Fatalf("NewSMS(provider=%q) = %T, want *aliyunSMS", provider, sender)
		}
		if sms.signName != "Planstar" || sms.templateCode != "SMS_001" {
			t.Errorf("aliyunSMS sign/template = %q/%q", sms.signName, sms.templateCode)
		}
	}
}

func TestNewSMSDisabledKeepsFallback(t *testing.T) {
	cfg := conf.SMS{Enabled: false, Provider: "aliyun"}
	if sender, err := NewSMS(cfg, "prod", nil); err != nil || !isNoopSMS(sender) {
		t.Fatalf("NewSMS(prod, disabled) = %T, %v; want NoopSMS, nil", sender, err)
	}
	if sender, err := NewSMS(cfg, "dev", nil); err != nil || !isLogSMS(sender) {
		t.Fatalf("NewSMS(dev, disabled) = %T, %v; want *logSMS, nil", sender, err)
	}
}

func TestDypnsEndpointDefault(t *testing.T) {
	if got := dypnsEndpoint(conf.SMS{}); got != defaultDypnsEndpoint {
		t.Errorf("dypnsEndpoint(empty) = %q, want %q", got, defaultDypnsEndpoint)
	}
	if got := dypnsEndpoint(conf.SMS{Endpoint: " custom.example.com "}); got != "custom.example.com" {
		t.Errorf("dypnsEndpoint(custom) = %q", got)
	}
}
