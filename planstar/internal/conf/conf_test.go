package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// baseConfig 返回补齐默认值后的最小可启动配置。
func baseConfig() *Config {
	c := &Config{}
	applyDefaults(c)
	c.Server.HTTP.Addr = "0.0.0.0:8098"
	c.Data.MySQL.Addr = "127.0.0.1:3306"
	c.Data.MySQL.User = "root"
	return c
}

func TestValidateOK(t *testing.T) {
	if err := baseConfig().Validate(); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateRejectsInvalid(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"http addr", func(c *Config) { c.Server.HTTP.Addr = "8098" }, "server.http.addr"},
		{"http port", func(c *Config) { c.Server.HTTP.Addr = "0.0.0.0:70000" }, "1-65535"},
		{"mysql addr", func(c *Config) { c.Data.MySQL.Addr = "" }, "data.mysql.addr"},
		{"mysql user", func(c *Config) { c.Data.MySQL.User = " " }, "data.mysql.user"},
		{"elastic url", func(c *Config) { c.Data.Elastic.Addr = "elasticsearch:9200" }, "data.elastic.addr"},
		{"mq url", func(c *Config) { c.MQ.Addr = "rabbitmq:5672" }, "mq.addr"},
		{"consul addr", func(c *Config) { c.Registry.Consul.Address = "consul" }, "registry.consul.address"},
		{"trace url", func(c *Config) { c.Trace.Endpoint = "jaeger:14268" }, "trace.endpoint"},
		{"timeout order", func(c *Config) { c.Server.HTTP.RequestTimeoutSeconds = 30 }, "必须小于"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := baseConfig()
			tc.mutate(c)
			err := c.Validate()
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestLoadAppliesEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  http:
    addr: 0.0.0.0:8098
  admin:
    addr: 0.0.0.0:9108
data:
  mysql:
    addr: 127.0.0.1:3306
    user: root
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PLANSTAR_MYSQL_ADDR", "mysql:3307")
	t.Setenv("PLANSTAR_MYSQL_USER", "planstar")
	t.Setenv("PLANSTAR_HTTP_ADDR", "0.0.0.0:18098")
	t.Setenv("PLANSTAR_ADMIN_ADDR", "0.0.0.0:19108")

	c, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if c.Data.MySQL.Addr != "mysql:3307" || c.Data.MySQL.User != "planstar" {
		t.Fatalf("mysql override not applied: %+v", c.Data.MySQL)
	}
	if c.Server.HTTP.Addr != "0.0.0.0:18098" || c.Server.Admin.Addr != "0.0.0.0:19108" {
		t.Fatalf("server addr override not applied: %+v", c.Server)
	}
	if Loaded() != c {
		t.Fatalf("Loaded() should return the latest config")
	}
}
