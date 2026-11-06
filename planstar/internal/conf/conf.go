package conf

import (
	"os"

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
	HTTP  HTTP `yaml:"http"`
	Admin HTTP `yaml:"admin"`
}

type HTTP struct {
	Addr string `yaml:"addr"`
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

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Auth.SessionExpireDays == 0 {
		c.Auth.SessionExpireDays = 30
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
	// 敏感信息优先由环境变量注入，避免把真实口令写进配置文件。
	override(&c.Data.MySQL.Password, "PLANSTAR_MYSQL_PASSWORD")
	override(&c.Data.Redis.Password, "PLANSTAR_REDIS_PASSWORD")
	override(&c.MQ.Addr, "PLANSTAR_MQ_ADDR")
	override(&c.Trace.Endpoint, "PLANSTAR_TRACE_ENDPOINT")
	return &c, nil
}

func override(dst *string, env string) {
	if v := os.Getenv(env); v != "" {
		*dst = v
	}
}
