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
	Ops      Ops      `yaml:"ops"`
	Notify   Notify   `yaml:"notify"`
	Storage  Storage  `yaml:"storage"`
}

// Notify 是短信与邮件通知配置；默认关闭，真实服务商客户端按 provider 接入。
type Notify struct {
	SMS   SMS   `yaml:"sms"`
	Email Email `yaml:"email"`
}

// SMS 是短信服务商配置。
type SMS struct {
	Enabled      bool   `yaml:"enabled"`
	Provider     string `yaml:"provider"`
	Endpoint     string `yaml:"endpoint"`
	AccessKey    string `yaml:"access_key"`
	AccessSecret string `yaml:"access_secret"`
	SignName     string `yaml:"sign_name"`
	TemplateCode string `yaml:"template_code"`
}

// Email 是 SMTP 邮件配置。
type Email struct {
	Enabled  bool   `yaml:"enabled"`
	Provider string `yaml:"provider"`
	SMTPHost string `yaml:"smtp_host"`
	SMTPPort int    `yaml:"smtp_port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

// Storage 是对象存储配置。
type Storage struct {
	OSS OSS `yaml:"oss"`
}

// OSS 是对象存储（阿里云 OSS / S3 等）配置。
type OSS struct {
	Enabled           bool   `yaml:"enabled"`
	Provider          string `yaml:"provider"`
	Endpoint          string `yaml:"endpoint"`
	Region            string `yaml:"region"`
	Bucket            string `yaml:"bucket"`
	AccessKey         string `yaml:"access_key"`
	AccessSecret      string `yaml:"access_secret"`
	CDNHost           string `yaml:"cdn_host"`
	PresignTTLSeconds int    `yaml:"presign_ttl_seconds"`
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
	// CookieSecure 显式控制 psSession 的 Secure 属性；未配置时按 server.env 推导（生产级环境=true）。
	CookieSecure *bool `yaml:"cookie_secure"`
	// SendCodeCooldownSeconds 是同一手机号发送验证码的冷却秒数。
	SendCodeCooldownSeconds int `yaml:"send_code_cooldown_seconds"`
	// SendCodeIPLimitPerMinute 是同一 IP 每分钟发送验证码的最大次数。
	SendCodeIPLimitPerMinute int `yaml:"send_code_ip_limit_per_minute"`
	// LoginFailCaptchaThreshold 是触发图形验证码的连续登录失败次数。
	LoginFailCaptchaThreshold int `yaml:"login_fail_captcha_threshold"`
	// AllowTestCredentials 仅允许本地版本开启：开启后新注册用户可被下发测试口令。
	// 生产级环境必须为 false，Validate() 会强制失败关闭（见 validateProductionInvariants）。
	AllowTestCredentials bool `yaml:"allow_test_credentials"`
	// TestCredentialPassword 仅在 AllowTestCredentials=true 时生效；为空时使用 LocalTestCredentialPassword。
	// 禁止写入任何生产配置文件，只能通过 PLANSTAR_AUTH_TEST_CREDENTIAL_PASSWORD 注入。
	TestCredentialPassword string `yaml:"test_credential_password"`
	// CodeHMACKey 是短信验证码入库摘要的 HMAC 密钥；为空时复用 ops.token，
	// 生产环境由 PLANSTAR_AUTH_CODE_HMAC_KEY 显式注入。
	CodeHMACKey string `yaml:"code_hmac_key"`
}

// CodeHMACKeyValue 返回验证码摘要密钥，未配置时回退到运维令牌（生产必填）。
func (a Auth) CodeHMACKeyValue(fallback string) string {
	if v := strings.TrimSpace(a.CodeHMACKey); v != "" {
		return v
	}
	return fallback
}

// CookieSecureEnabled 返回是否对会话 Cookie 启用 Secure（默认配置在 Load 中补齐）。
func (a Auth) CookieSecureEnabled() bool {
	return a.CookieSecure != nil && *a.CookieSecure
}

// TestCredentialPasswordValue 返回本地版本使用的一次性测试口令。
// 未开启 AllowTestCredentials 时返回空串，data 层据此把 password_hash 置空。
func (a Auth) TestCredentialPasswordValue() string {
	if !a.AllowTestCredentials {
		return ""
	}
	if v := strings.TrimSpace(a.TestCredentialPassword); v != "" {
		return v
	}
	return LocalTestCredentialPassword
}

// ---------------------------------------------------------------------------
// 环境判定：默认拒绝（deny-by-default）。除显式声明的本地环境外，
// 一律按生产级环境施加约束，避免 staging / docker / 拼写变体绕过安全基线。
// 对应《阿里巴巴Java开发手册（黄山版）》「安全规约 / 工程规约」的
// 「配置外置 + 环境隔离 + 上线前校验」要求。
// ---------------------------------------------------------------------------

const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"

	// LocalTestCredentialPassword 是本地/演示版本演示账号的唯一测试口令常量。
	// 代码里禁止再出现其它硬编码口令字面量；生产环境不允许引用该值。
	LocalTestCredentialPassword = "123456"
)

var localEnvNames = map[string]struct{}{
	"local": {}, "dev": {}, "development": {}, "test": {}, "testing": {},
}

// IsLocalEnv 判断是否为可放松安全基线的本地/测试环境。
func IsLocalEnv(env string) bool {
	_, ok := localEnvNames[strings.ToLower(strings.TrimSpace(env))]
	return ok
}

// IsProductionLikeEnv 判断是否必须施加生产级安全约束（默认拒绝）。
func IsProductionLikeEnv(env string) bool { return !IsLocalEnv(env) }

type Trace struct {
	Endpoint string `yaml:"endpoint"`
	// Insecure 控制 OTLP/gRPC 是否走明文连接；未配置时默认 true（内网 collector）。
	Insecure    *bool   `yaml:"insecure"`
	Env         string  `yaml:"env"`
	SampleRatio float64 `yaml:"sample_ratio"`
}

// InsecureEnabled 返回是否使用明文 gRPC（默认开启，适用内网 collector）。
func (t Trace) InsecureEnabled() bool {
	return t.Insecure == nil || *t.Insecure
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

// Ops 是运维探针与指标端点配置。
type Ops struct {
	// Token 由 PLANSTAR_OPS_TOKEN 注入，供后续运维端点鉴权使用。
	Token string `yaml:"token"`
	// AllowedCIDRs 由 PLANSTAR_OPS_ALLOWED_CIDRS 逗号分隔注入。
	AllowedCIDRs             []string  `yaml:"allowed_cidrs"`
	ProbeCacheTTLSeconds     int       `yaml:"probe_cache_ttl_seconds"`
	ProbeTimeoutSeconds      int       `yaml:"probe_timeout_seconds"`
	ProbeTotalTimeoutSeconds int       `yaml:"probe_total_timeout_seconds"`
	MetricsMaxInflight       int       `yaml:"metrics_max_inflight"`
	MetricsTimeoutSeconds    int       `yaml:"metrics_timeout_seconds"`
	Readiness                Readiness `yaml:"readiness"`
}

// Readiness 描述各依赖的就绪探测参数。
type Readiness struct {
	MySQL         ProbeConfig `yaml:"mysql"`
	Redis         ProbeConfig `yaml:"redis"`
	Elasticsearch ProbeConfig `yaml:"elasticsearch"`
	RabbitMQ      ProbeConfig `yaml:"rabbitmq"`
	Consul        ProbeConfig `yaml:"consul"`
}

// ProbeConfig 是单个依赖探测项；Critical 用指针区分「未配置」与「显式 false」。
type ProbeConfig struct {
	Critical       *bool `yaml:"critical"`
	TimeoutSeconds int   `yaml:"timeout_seconds"`
}

// IsCritical 返回是否关键依赖；未配置时使用调用方给的默认值。
func (p ProbeConfig) IsCritical(fallback bool) bool {
	if p.Critical != nil {
		return *p.Critical
	}
	return fallback
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
	// PLANSTAR_ENV 需在 applyDefaults 之前生效，保证 Cookie Secure 等安全默认值按真实环境推导。
	override(&c.Server.Env, "PLANSTAR_ENV")
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
	if v := os.Getenv("PLANSTAR_TRACE_INSECURE"); v != "" {
		insecure, err := strconv.ParseBool(strings.TrimSpace(v))
		if err != nil {
			return nil, fmt.Errorf("PLANSTAR_TRACE_INSECURE 必须是 true/false/1/0: %q", v)
		}
		c.Trace.Insecure = &insecure
	}
	override(&c.Registry.Consul.Address, "PLANSTAR_CONSUL_ADDRESS")
	override(&c.Ops.Token, "PLANSTAR_OPS_TOKEN")
	// 测试口令开关仅由环境变量注入；配置文件里写死也不会在生产通过校验。
	if err := overrideBool(&c.Auth.AllowTestCredentials, "PLANSTAR_AUTH_ALLOW_TEST_CREDENTIALS"); err != nil {
		return nil, err
	}
	override(&c.Auth.TestCredentialPassword, "PLANSTAR_AUTH_TEST_CREDENTIAL_PASSWORD")
	override(&c.Auth.CodeHMACKey, "PLANSTAR_AUTH_CODE_HMAC_KEY")
	// 通知与对象存储默认关闭，密钥仅通过环境变量注入。
	if err := overrideBool(&c.Notify.SMS.Enabled, "PLANSTAR_NOTIFY_SMS_ENABLED"); err != nil {
		return nil, err
	}
	override(&c.Notify.SMS.Provider, "PLANSTAR_NOTIFY_SMS_PROVIDER")
	override(&c.Notify.SMS.Endpoint, "PLANSTAR_NOTIFY_SMS_ENDPOINT")
	override(&c.Notify.SMS.AccessKey, "PLANSTAR_NOTIFY_SMS_ACCESS_KEY")
	override(&c.Notify.SMS.AccessSecret, "PLANSTAR_NOTIFY_SMS_ACCESS_SECRET")
	override(&c.Notify.SMS.SignName, "PLANSTAR_NOTIFY_SMS_SIGN_NAME")
	override(&c.Notify.SMS.TemplateCode, "PLANSTAR_NOTIFY_SMS_TEMPLATE_CODE")
	if err := overrideBool(&c.Notify.Email.Enabled, "PLANSTAR_NOTIFY_EMAIL_ENABLED"); err != nil {
		return nil, err
	}
	override(&c.Notify.Email.Provider, "PLANSTAR_NOTIFY_EMAIL_PROVIDER")
	override(&c.Notify.Email.SMTPHost, "PLANSTAR_NOTIFY_EMAIL_SMTP_HOST")
	if err := overrideInt(&c.Notify.Email.SMTPPort, "PLANSTAR_NOTIFY_EMAIL_SMTP_PORT"); err != nil {
		return nil, err
	}
	override(&c.Notify.Email.Username, "PLANSTAR_NOTIFY_EMAIL_USERNAME")
	override(&c.Notify.Email.Password, "PLANSTAR_NOTIFY_EMAIL_PASSWORD")
	override(&c.Notify.Email.From, "PLANSTAR_NOTIFY_EMAIL_FROM")
	if err := overrideBool(&c.Storage.OSS.Enabled, "PLANSTAR_STORAGE_OSS_ENABLED"); err != nil {
		return nil, err
	}
	override(&c.Storage.OSS.Provider, "PLANSTAR_STORAGE_OSS_PROVIDER")
	override(&c.Storage.OSS.Endpoint, "PLANSTAR_STORAGE_OSS_ENDPOINT")
	override(&c.Storage.OSS.Region, "PLANSTAR_STORAGE_OSS_REGION")
	override(&c.Storage.OSS.Bucket, "PLANSTAR_STORAGE_OSS_BUCKET")
	override(&c.Storage.OSS.AccessKey, "PLANSTAR_STORAGE_OSS_ACCESS_KEY")
	override(&c.Storage.OSS.AccessSecret, "PLANSTAR_STORAGE_OSS_ACCESS_SECRET")
	override(&c.Storage.OSS.CDNHost, "PLANSTAR_STORAGE_OSS_CDN_HOST")
	if err := overrideInt(&c.Storage.OSS.PresignTTLSeconds, "PLANSTAR_STORAGE_OSS_PRESIGN_TTL_SECONDS"); err != nil {
		return nil, err
	}
	if v := os.Getenv("PLANSTAR_OPS_ALLOWED_CIDRS"); v != "" {
		cidrs := make([]string, 0, 4)
		for _, item := range strings.Split(v, ",") {
			if item = strings.TrimSpace(item); item != "" {
				cidrs = append(cidrs, item)
			}
		}
		c.Ops.AllowedCIDRs = cidrs
	}
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
	if c.Trace.Endpoint != "" && !validTraceEndpoint(c.Trace.Endpoint) {
		return fmt.Errorf("trace.endpoint 必须是 host:port 或带 scheme 的 URL: %q", c.Trace.Endpoint)
	}
	// 请求超时不小于连接写超时时，504 可能来不及写回客户端。
	if c.Server.HTTP.RequestTimeoutSeconds > 0 && c.Server.HTTP.WriteTimeoutSeconds > 0 &&
		c.Server.HTTP.RequestTimeoutSeconds >= c.Server.HTTP.WriteTimeoutSeconds {
		return fmt.Errorf("server.http.request_timeout_seconds(%d) 必须小于 write_timeout_seconds(%d)",
			c.Server.HTTP.RequestTimeoutSeconds, c.Server.HTTP.WriteTimeoutSeconds)
	}
	return c.validateProductionInvariants()
}

// validateProductionInvariants 是生产上线前的 fail-fast 断言。
// 依据《阿里巴巴Java开发手册（黄山版）》安全规约「敏感信息不得硬编码 / 生产禁用测试账号」
// 与工程规约「配置外置、启动即校验」，任何一条不满足都在启动阶段终止，而不是运行期暴露。
func (c *Config) validateProductionInvariants() error {
	if !IsProductionLikeEnv(c.Server.Env) {
		return nil
	}
	fail := func(format string, args ...any) error {
		return fmt.Errorf("生产配置校验失败（server.env=%s）: %s", c.Server.Env, fmt.Sprintf(format, args...))
	}
	if c.Auth.AllowTestCredentials {
		return fail("auth.allow_test_credentials 必须为 false，生产禁止下发测试口令")
	}
	if !c.Auth.CookieSecureEnabled() {
		return fail("auth.cookie_secure 必须为 true")
	}
	if strings.TrimSpace(c.Ops.Token) == "" {
		return fail("ops.token 不能为空（PLANSTAR_OPS_TOKEN）")
	}
	if len(c.Ops.AllowedCIDRs) == 0 {
		return fail("ops.allowed_cidrs 不能为空（PLANSTAR_OPS_ALLOWED_CIDRS）")
	}
	if strings.TrimSpace(c.Data.MySQL.Password) == "" {
		return fail("data.mysql.password 不能为空（PLANSTAR_MYSQL_PASSWORD）")
	}
	if strings.EqualFold(strings.TrimSpace(c.Data.MySQL.User), "root") {
		return fail("data.mysql.user 禁止使用 root，必须配置最小权限账号")
	}
	if strings.TrimSpace(c.Data.Redis.Password) == "" {
		return fail("data.redis.password 不能为空（PLANSTAR_REDIS_PASSWORD）")
	}
	// 已购买的阿里云短信是生产唯一允许的验证码通道；Noop/Log 会静默丢弃验证码。
	if !c.Notify.SMS.Enabled {
		return fail("notify.sms.enabled 必须为 true（生产必须接入阿里云短信）")
	}
	if p := strings.ToLower(strings.TrimSpace(c.Notify.SMS.Provider)); p != "aliyun" && p != "aliyun-dypns" {
		return fail("notify.sms.provider 必须是 aliyun")
	}
	for _, item := range []struct{ field, value string }{
		{"notify.sms.access_key", c.Notify.SMS.AccessKey},
		{"notify.sms.access_secret", c.Notify.SMS.AccessSecret},
		{"notify.sms.sign_name", c.Notify.SMS.SignName},
		{"notify.sms.template_code", c.Notify.SMS.TemplateCode},
	} {
		if strings.TrimSpace(item.value) == "" {
			return fail("%s 不能为空", item.field)
		}
	}
	// OSS 是可选项；一旦启用就必须完整配置，避免运行期才报错。
	if c.Storage.OSS.Enabled {
		if p := strings.ToLower(strings.TrimSpace(c.Storage.OSS.Provider)); p != "aliyun" && p != "oss" {
			return fail("storage.oss.provider 必须是 aliyun")
		}
		for _, item := range []struct{ field, value string }{
			{"storage.oss.endpoint", c.Storage.OSS.Endpoint},
			{"storage.oss.bucket", c.Storage.OSS.Bucket},
			{"storage.oss.access_key", c.Storage.OSS.AccessKey},
			{"storage.oss.access_secret", c.Storage.OSS.AccessSecret},
		} {
			if strings.TrimSpace(item.value) == "" {
				return fail("%s 不能为空", item.field)
			}
		}
	}
	return nil
}

// validTraceEndpoint 接受带 scheme 的 URL，或 gRPC 使用的 host:port。
func validTraceEndpoint(endpoint string) bool {
	if u, err := url.Parse(endpoint); err == nil && u.Scheme != "" && u.Host != "" {
		return true
	}
	host, port, err := net.SplitHostPort(endpoint)
	if err != nil || host == "" {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
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
		// 默认拒绝：未显式声明环境时按生产级环境处理，缺配置直接启动失败。
		c.Server.Env = EnvProd
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
		secure := IsProductionLikeEnv(c.Server.Env)
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
	if c.Trace.Insecure == nil {
		insecure := true
		c.Trace.Insecure = &insecure
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
	// 运维探针默认值：整体缓存 2s、单检查 2s、总超时 3s。
	if c.Ops.ProbeCacheTTLSeconds <= 0 {
		c.Ops.ProbeCacheTTLSeconds = 2
	}
	if c.Ops.ProbeTimeoutSeconds <= 0 {
		c.Ops.ProbeTimeoutSeconds = 2
	}
	if c.Ops.ProbeTotalTimeoutSeconds <= 0 {
		c.Ops.ProbeTotalTimeoutSeconds = 3
	}
	if c.Ops.MetricsMaxInflight <= 0 {
		c.Ops.MetricsMaxInflight = 8
	}
	if c.Ops.MetricsTimeoutSeconds <= 0 {
		c.Ops.MetricsTimeoutSeconds = 5
	}
	for _, probe := range []*ProbeConfig{
		&c.Ops.Readiness.MySQL,
		&c.Ops.Readiness.Redis,
		&c.Ops.Readiness.Elasticsearch,
		&c.Ops.Readiness.RabbitMQ,
		&c.Ops.Readiness.Consul,
	} {
		if probe.TimeoutSeconds <= 0 {
			probe.TimeoutSeconds = c.Ops.ProbeTimeoutSeconds
		}
	}
	if c.Notify.Email.SMTPPort <= 0 {
		c.Notify.Email.SMTPPort = 465
	}
	if c.Storage.OSS.PresignTTLSeconds <= 0 {
		c.Storage.OSS.PresignTTLSeconds = 900
	}
}

func override(dst *string, env string) {
	if v := os.Getenv(env); v != "" {
		*dst = v
	}
}

func overrideBool(dst *bool, env string) error {
	v := os.Getenv(env)
	if v == "" {
		return nil
	}
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	if err != nil {
		return fmt.Errorf("%s 必须是 true/false/1/0: %q", env, v)
	}
	*dst = b
	return nil
}

func overrideInt(dst *int, env string) error {
	v := os.Getenv(env)
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fmt.Errorf("%s 必须是整数: %q", env, v)
	}
	*dst = n
	return nil
}
