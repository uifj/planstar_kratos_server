package conf

import (
	"strings"
	"testing"
)

// prodConfig 返回一份满足全部生产 fail-fast 断言的最小配置。
func prodConfig() *Config {
	c := &Config{}
	applyDefaults(c)
	c.Server.Env = EnvProd
	c.Server.HTTP.Addr = "0.0.0.0:8098"
	c.Server.Admin.Addr = "0.0.0.0:9108"
	c.Data.MySQL.Addr = "mysql:3306"
	c.Data.MySQL.User = "planstar_app"
	c.Data.MySQL.Password = "strong-db-password"
	c.Data.Redis.Addr = "redis:6379"
	c.Data.Redis.Password = "strong-redis-password"
	c.Ops.Token = "strong-ops-token"
	c.Ops.AllowedCIDRs = []string{"127.0.0.1/32"}
	secure := true
	c.Auth.CookieSecure = &secure
	c.Notify.SMS.Enabled = true
	c.Notify.SMS.Provider = "aliyun"
	c.Notify.SMS.AccessKey = "ak"
	c.Notify.SMS.AccessSecret = "sk"
	c.Notify.SMS.SignName = "sign"
	c.Notify.SMS.TemplateCode = "SMS_001"
	return c
}

func TestEnvironmentClassificationDefaultsToDeny(t *testing.T) {
	cases := []struct {
		env       string
		wantLocal bool
	}{
		{"local", true}, {"dev", true}, {"test", true}, {" DEV ", true},
		{"prod", false}, {"production", false}, {"staging", false},
		{"docker", false}, {"", false}, {"prodktion", false},
	}
	for _, tc := range cases {
		if got := IsLocalEnv(tc.env); got != tc.wantLocal {
			t.Errorf("IsLocalEnv(%q) = %v, want %v", tc.env, got, tc.wantLocal)
		}
		if got := IsProductionLikeEnv(tc.env); got == tc.wantLocal {
			t.Errorf("IsProductionLikeEnv(%q) = %v, want %v", tc.env, got, !tc.wantLocal)
		}
	}
}

func TestProductionInvariantsAcceptValidConfig(t *testing.T) {
	if err := prodConfig().Validate(); err != nil {
		t.Fatalf("valid production config rejected: %v", err)
	}
}

func TestProductionInvariantsRejectUnsafeConfig(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"测试口令开关", func(c *Config) { c.Auth.AllowTestCredentials = true }, "allow_test_credentials"},
		{"Cookie 非 Secure", func(c *Config) { v := false; c.Auth.CookieSecure = &v }, "cookie_secure"},
		{"运维令牌为空", func(c *Config) { c.Ops.Token = "" }, "ops.token"},
		{"运维来源白名单为空", func(c *Config) { c.Ops.AllowedCIDRs = nil }, "allowed_cidrs"},
		{"DB 口令为空", func(c *Config) { c.Data.MySQL.Password = "" }, "mysql.password"},
		{"DB 使用 root", func(c *Config) { c.Data.MySQL.User = "root" }, "root"},
		{"Redis 口令为空", func(c *Config) { c.Data.Redis.Password = "" }, "redis.password"},
		{"短信未启用", func(c *Config) { c.Notify.SMS.Enabled = false }, "notify.sms.enabled"},
		{"短信 provider 非 aliyun", func(c *Config) { c.Notify.SMS.Provider = "noop" }, "provider"},
		{"短信密钥缺失", func(c *Config) { c.Notify.SMS.AccessSecret = "" }, "access_secret"},
		{"OSS 启用但配置不全", func(c *Config) { c.Storage.OSS.Enabled = true }, "storage.oss.provider"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := prodConfig()
			tc.mutate(c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("expected production config to be rejected")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q should contain %q", err.Error(), tc.want)
			}
		})
	}
}

func TestLocalEnvSkipsProductionInvariants(t *testing.T) {
	c := prodConfig()
	c.Server.Env = EnvLocal
	c.Ops.Token = ""
	c.Ops.AllowedCIDRs = nil
	c.Notify.SMS.Enabled = false
	c.Data.MySQL.User = "root"
	c.Data.MySQL.Password = ""
	if err := c.Validate(); err != nil {
		t.Fatalf("local config should not be subject to production invariants: %v", err)
	}
}

func TestTestCredentialPasswordValueGated(t *testing.T) {
	a := Auth{}
	if got := a.TestCredentialPasswordValue(); got != "" {
		t.Fatalf("disabled test credentials returned %q, want empty", got)
	}
	a.AllowTestCredentials = true
	if got := a.TestCredentialPasswordValue(); got != LocalTestCredentialPassword {
		t.Fatalf("local test password = %q, want %q", got, LocalTestCredentialPassword)
	}
	a.TestCredentialPassword = "custom-local"
	if got := a.TestCredentialPasswordValue(); got != "custom-local" {
		t.Fatalf("override = %q, want custom-local", got)
	}
}

func TestCodeHMACKeyFallback(t *testing.T) {
	if got := (Auth{}).CodeHMACKeyValue("ops-token"); got != "ops-token" {
		t.Fatalf("fallback = %q, want ops-token", got)
	}
	if got := (Auth{CodeHMACKey: "dedicated"}).CodeHMACKeyValue("ops-token"); got != "dedicated" {
		t.Fatalf("explicit key = %q, want dedicated", got)
	}
}
