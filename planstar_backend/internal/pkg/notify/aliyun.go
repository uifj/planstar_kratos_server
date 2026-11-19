package notify

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dypnsapi "github.com/alibabacloud-go/dypnsapi-20170525/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"

	"planstar/internal/conf"
)

const (
	defaultDypnsEndpoint = "dypnsapi.aliyuncs.com"
	dypnsRequestTimeout  = 5000
	dypnsCodeLength      = 6
	dypnsValidTime       = 300
)

// smsVerifyCodeRequest 是发送验证码的最小参数集，便于测试替换 SDK。
type smsVerifyCodeRequest struct {
	Mobile        string
	SignName      string
	TemplateCode  string
	TemplateParam string
	CodeLength    int64
	ValidTime     int64
}

// smsVerifyCodeAPI 包装 Dypnsapi 调用，测试可注入 fake。
type smsVerifyCodeAPI interface {
	SendSmsVerifyCode(ctx context.Context, req smsVerifyCodeRequest) (code, message string, err error)
}

type aliyunSMS struct {
	api          smsVerifyCodeAPI
	signName     string
	templateCode string
}

func (s *aliyunSMS) SendCode(ctx context.Context, mobile, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	resultCode, resultMessage, err := s.api.SendSmsVerifyCode(ctx, smsVerifyCodeRequest{
		Mobile:        mobile,
		SignName:      s.signName,
		TemplateCode:  s.templateCode,
		TemplateParam: fmt.Sprintf(`{"code":%q}`, code),
		CodeLength:    dypnsCodeLength,
		ValidTime:     dypnsValidTime,
	})
	if err != nil {
		return fmt.Errorf("aliyun sms send verify code: %w", err)
	}
	if resultCode != "OK" {
		return fmt.Errorf("aliyun sms send verify code failed: code=%q message=%q", resultCode, resultMessage)
	}
	return nil
}

// Send 明确拒绝自由文本：Dypnsapi 只发送模板验证码。
func (s *aliyunSMS) Send(context.Context, Message) error {
	return errors.New("aliyun sms provider supports verification codes only")
}

// dypnsClient 是 smsVerifyCodeAPI 的 SDK 适配层。
type dypnsClient struct {
	client *dypnsapi.Client
}

func newDypnsClient(cfg conf.SMS) (smsVerifyCodeAPI, error) {
	client, err := dypnsapi.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(strings.TrimSpace(cfg.AccessKey)),
		AccessKeySecret: tea.String(strings.TrimSpace(cfg.AccessSecret)),
		Endpoint:        tea.String(dypnsEndpoint(cfg)),
	})
	if err != nil {
		return nil, fmt.Errorf("init aliyun dypns client: %w", err)
	}
	return &dypnsClient{client: client}, nil
}

func (c *dypnsClient) SendSmsVerifyCode(ctx context.Context, req smsVerifyCodeRequest) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	resp, err := c.client.SendSmsVerifyCodeWithOptions(&dypnsapi.SendSmsVerifyCodeRequest{
		PhoneNumber:   tea.String(req.Mobile),
		SignName:      tea.String(req.SignName),
		TemplateCode:  tea.String(req.TemplateCode),
		TemplateParam: tea.String(req.TemplateParam),
		CodeLength:    tea.Int64(req.CodeLength),
		ValidTime:     tea.Int64(req.ValidTime),
	}, &util.RuntimeOptions{
		ConnectTimeout: tea.Int(dypnsRequestTimeout),
		ReadTimeout:    tea.Int(dypnsRequestTimeout),
	})
	if err != nil {
		return "", "", err
	}
	if resp == nil || resp.Body == nil {
		return "", "", errors.New("aliyun dypns returned empty response body")
	}
	return tea.StringValue(resp.Body.Code), tea.StringValue(resp.Body.Message), nil
}

func dypnsEndpoint(cfg conf.SMS) string {
	if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
		return endpoint
	}
	return defaultDypnsEndpoint
}

func validateSMSConfig(cfg conf.SMS) error {
	var missing []string
	for _, item := range []struct{ field, value string }{
		{"notify.sms.access_key", cfg.AccessKey},
		{"notify.sms.access_secret", cfg.AccessSecret},
		{"notify.sms.sign_name", cfg.SignName},
		{"notify.sms.template_code", cfg.TemplateCode},
	} {
		if strings.TrimSpace(item.value) == "" {
			missing = append(missing, item.field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("aliyun sms config missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
