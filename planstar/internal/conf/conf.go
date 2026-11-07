package conf

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 是服务的运行时配置（config.yaml）。
type Config struct {
	Server   Server   `yaml:"server"`
	Data     Data     `yaml:"data"`
	Auth     Auth     `yaml:"auth"`
	Trace    Trace    `yaml:"trace"`
	MQ       MQ       `yaml:"mq"`
	Registry Registry `yaml:"registry"`
}

type Server struct {
	// Env 是运行环境（dev/prod），用于推导 Cookie Secure 等安全默认值。
	Env   string `yaml:"env"`
	HTTP  HTTP   `yaml:"http"`
	Admin HTTP   `yaml:"admin"`
}

type HTTP struct {
	Addr string `yaml:"addr"`
	// ReadTimeoutSeconds / WriteTimeoutSeconds / IdleTimeoutSeconds 是连接级超时。
	ReadTimeoutSeconds  int `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds int `yaml:"write_timeout_seconds"`
	IdleTimeoutSeconds  int `yaml:"idle_timeout_seconds"`
	// RequestTimeoutSeconds 是单请求 context 超时，超时返回 504。
	RequestTimeoutSeconds int `yaml:"request_timeout_seconds"`
	// MaxBodyBytes 是请求体上限（字节），upload 前缀单独放宽。
	MaxBodyBytes int64 `yaml:"max_body_bytes"`
	// CORSAllowOrigins 是跨域白名单；为空时仅允许同源请求。
	CORSAllowOrigins []string `yaml:"cors_allow_origins"`
}

type Data struct {
	MySQL   MySQL   `yaml:"mysql"`
	Redis   Redis   `yaml:"redis"`
	Elastic Elastic `yaml:"elastic"`
}

type MySQL struct {
	Addr     string `yaml:"addr"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Charset  string `yaml:"charset"`
}

type Redis struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type Elastic struct {
	Addr           string `yaml:"addr"`
	IndexPrefix    string `yaml:"index_prefix"`
	SyncInterval   int    `yaml:"sync_interval_seconds"`
	RequestTimeout int    `yaml:"request_timeout_seconds"`
}

type Auth struct {
	SessionExpireDays int `yaml:"session_expire_days"`
	// CookieSecure 显式控制 psSession 的 Secure 属性；未配置时按 server.env 推导（prod=true）。
	CookieSecure *bool `yaml:"cookie_secure"`
	// SendCodeCooldownSeconds 是同一手机号发送验证码的冷却秒数。
	SendCodeCooldownSeconds int `yaml:"send_code_cooldown_seconds"`
	// SendCodeIPLimitPerMinute 是同一 IP 每分钟发送验证码的最大次数。
	SendCodeIPLimitPerMinute int `yaml:"send_code_ip_limit_per_minute"`
	// LoginFailCaptchaThreshold 是触发图形验证码的连续登录失败次数。
	LoginFailCaptchaThreshold int `yaml:"login_fail_captcha_threshold"`
}

// CookieSecureEnabled 返回是否对会话 Cookie 启用 Secure（默认配置在 Load 中补齐）。
func (a Auth) CookieSecureEnabled() bool {
	return a.CookieSecure != nil && *a.CookieSecure
}

type Trace struct {
	Endpoint    string  `yaml:"endpoint"`
	Env         string  `yaml:"env"`
	SampleRatio float64 `yaml:"sample_ratio"`
}

type MQ struct {
	Addr     string `yaml:"addr"`
	Exchange string `yaml:"exchange"`
	Queue    string `yaml:"queue"`
}

type Registry struct {
	Consul Consul `yaml:"consul"`
}

type Consul struct {
	Address string `yaml:"address"`
	Scheme  string `yaml:"scheme"`
	// ServiceHost 是注册到 Consul 的可达地址（默认 127.0.0.1，容器内按需配置主机名）。
	ServiceHost string `yaml:"service_host"`
}

// loaded 保存最近一次 Load 的配置，供不便接收 conf 的适配层读取。
var loaded *Config

// Load 读取并补齐配置默认值（敏感项可由 PLANSTAR_* 环境变量覆盖）。
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	applyDefaults(&c)
	// 敏感信息与部署相关地址优先由环境变量注入，避免把真实口令写进配置文件。
	override(&c.Server.HTTP.Addr, "PLANSTAR_HTTP_ADDR")
	override(&c.Server.Admin.Addr, "PLANSTAR_ADMIN_ADDR")
	override(&c.Data.MySQL.Addr, "PLANSTAR_MYSQL_ADDR")
	override(&c.Data.MySQL.User, "PLANSTAR_MYSQL_USER")
	override(&c.Data.MySQL.Password, "PLANSTAR_MYSQL_PASSWORD")
	override(&c.Data.Redis.Addr, "PLANSTAR_REDIS_ADDR")
	override(&c.Data.Redis.Password, "PLANSTAR_REDIS_PASSWORD")
	override(&c.Data.Elastic.Addr, "PLANSTAR_ES_ADDR")
	override(&c.MQ.Addr, "PLANSTAR_MQ_ADDR")
	override(&c.Trace.Endpoint, "PLANSTAR_TRACE_ENDPOINT")
	override(&c.Registry.Consul.Address, "PLANSTAR_CONSUL_ADDRESS")
	if err := c.Validate(); err != nil {
		return nil, err
	}
	loaded = &c
	return &c, nil
}

// Validate 校验启动必需的配置项，遵循 Kratos v3「配置在启动阶段失败关闭」的原则。
// 在 applyDefaults 与环境变量覆盖之后调用，避免把错误配置带进运行期。
func (c *Config) Validate() error {
	for _, item := range []struct{ field, addr string }{
		{"server.http.addr", c.Server.HTTP.Addr},
		{"server.admin.addr", c.Server.Admin.Addr},
		{"data.mysql.addr", c.Data.MySQL.Addr},
	} {
		if err := validateHostPort(item.field, item.addr); err != nil {
			return err
		}
	}
	if strings.TrimSpace(c.Data.MySQL.User) == "" {
		return fmt.Errorf("data.mysql.user 不能为空")
	}
	if c.Data.Redis.Addr != "" {
		if err := validateHostPort("data.redis.addr", c.Data.Redis.Addr); err != nil {
			return err
		}
	}
	if c.Data.Elastic.Addr != "" {
		u, err := url.Parse(c.Data.Elastic.Addr)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("data.elastic.addr 必须是带 scheme 的 URL: %q", c.Data.Elastic.Addr)
		}
	}
	if c.MQ.Addr != "" {
		u, err := url.Parse(c.MQ.Addr)
		if err != nil || (u.Scheme != "amqp" && u.Scheme != "amqps") || u.Host == "" {
			return fmt.Errorf("mq.addr 必须是合法的 amqp(s) URL: %q", c.MQ.Addr)
		}
	}
	if c.Registry.Consul.Address != "" {
		if err := validateHostPort("registry.consul.address", c.Registry.Consul.Address); err != nil {
			return err
		}
	}
	if c.Trace.Endpoint != "" {
		u, err := url.Parse(c.Trace.Endpoint)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("trace.endpoint 必须是带 scheme 的 URL: %q", c.Trace.Endpoint)
		}
	}
	// 请求超时不小于连接写超时时，504 可能来不及写回客户端。
	if c.Server.HTTP.RequestTimeoutSeconds > 0 && c.Server.HTTP.WriteTimeoutSeconds > 0 &&
		c.Server.HTTP.RequestTimeoutSeconds >= c.Server.HTTP.WriteTimeoutSeconds {
		return fmt.Errorf("server.http.request_timeout_seconds(%d) 必须小于 write_timeout_seconds(%d)",
			c.Server.HTTP.RequestTimeoutSeconds, c.Server.HTTP.WriteTimeoutSeconds)
	}
	return nil
}

// validateHostPort 校验监听/连接地址格式，不做 DNS 解析（容器启动早期域名可能尚未就绪）。
func validateHostPort(field, addr string) error {
	if strings.TrimSpace(addr) == "" {
		return fmt.Errorf("%s 不能为空", field)
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%s 不是合法的 host:port: %q", field, addr)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("%s 端口必须在 1-65535 之间: %q", field, addr)
	}
	return nil
}

// Loaded 返回最近一次 Load 的配置；未加载时返回带安全默认值的配置（不会返回 nil）。
func Loaded() *Config {
	if loaded != nil {
		return loaded
	}
	c := &Config{}
	applyDefaults(c)
	return c
}

// applyDefaults 补齐安全默认值，不破坏已有配置字段。
func applyDefaults(c *Config) {
	if c.Server.Env == "" {
		c.Server.Env = "dev"
	}
	if c.Server.HTTP.ReadTimeoutSeconds <= 0 {
		c.Server.HTTP.ReadTimeoutSeconds = 10
	}
	if c.Server.HTTP.WriteTimeoutSeconds <= 0 {
		c.Server.HTTP.WriteTimeoutSeconds = 30
	}
	if c.Server.HTTP.IdleTimeoutSeconds <= 0 {
		c.Server.HTTP.IdleTimeoutSeconds = 60
	}
	if c.Server.HTTP.RequestTimeoutSeconds <= 0 {
		c.Server.HTTP.RequestTimeoutSeconds = 15
	}
	if c.Server.HTTP.MaxBodyBytes <= 0 {
		c.Server.HTTP.MaxBodyBytes = 1 << 20
	}
	if c.Auth.SessionExpireDays == 0 {
		c.Auth.SessionExpireDays = 30
	}
	if c.Auth.CookieSecure == nil {
		secure := c.Server.Env == "prod"
		c.Auth.CookieSecure = &secure
	}
	if c.Auth.SendCodeCooldownSeconds <= 0 {
		c.Auth.SendCodeCooldownSeconds = 60
	}
	if c.Auth.SendCodeIPLimitPerMinute <= 0 {
		c.Auth.SendCodeIPLimitPerMinute = 10
	}
	if c.Auth.LoginFailCaptchaThreshold <= 0 {
		c.Auth.LoginFailCaptchaThreshold = 3
	}
	if c.Data.MySQL.Charset == "" {
		c.Data.MySQL.Charset = "utf8mb4"
	}
	if c.Trace.SampleRatio <= 0 || c.Trace.SampleRatio > 1 {
		c.Trace.SampleRatio = 1
	}
	if c.Trace.Env == "" {
		c.Trace.Env = "dev"
	}
	if c.Data.Elastic.IndexPrefix == "" {
		c.Data.Elastic.IndexPrefix = "planstar"
	}
	if c.Data.Elastic.SyncInterval <= 0 {
		c.Data.Elastic.SyncInterval = 300
	}
	if c.Data.Elastic.RequestTimeout <= 0 {
		c.Data.Elastic.RequestTimeout = 3
	}
	if c.MQ.Exchange == "" {
		c.MQ.Exchange = "planstar.exchange"
	}
	if c.MQ.Queue == "" {
		c.MQ.Queue = "planstar.api.events"
	}
	if c.Server.Admin.Addr == "" {
		c.Server.Admin.Addr = "0.0.0.0:9108"
	}
	if c.Registry.Consul.Scheme == "" {
		c.Registry.Consul.Scheme = "http"
	}
	if c.Registry.Consul.ServiceHost == "" {
		c.Registry.Consul.ServiceHost = "127.0.0.1"
	}
}

func override(dst *string, env string) {
	if v := os.Getenv(env); v != "" {
		*dst = v
	}
}
