package main

import (
	"context"
	"time"

	"github.com/hashicorp/consul/api"

	"planstar/internal/conf"
	"planstar/internal/data"
	"planstar/internal/mq"
	"planstar/internal/ops"
)

// aggregateProbeStatus 汇总探针结果：任一关键依赖失败为 fail，非关键失败为 degraded。
func aggregateProbeStatus(results []ops.CheckResult) string {
	status := ops.StatusOK
	for _, res := range results {
		if res.Status != ops.StatusFail {
			continue
		}
		if res.Critical {
			return ops.StatusFail
		}
		status = ops.StatusDegraded
	}
	return status
}

// buildOpsChecks 按实际配置构造依赖探针；未配置的依赖 Probe 保持 nil，就绪引擎会标记为 skipped。
// MySQL/Redis 为 critical，ES/RabbitMQ/Consul 为 optional，未配置时对应 Probe 为 nil。
func buildOpsChecks(c *conf.Config, d *data.Data, pub *mq.Publisher, consulCli *api.Client) []ops.Check {
	timeout := func(sec int) time.Duration {
		if sec <= 0 {
			sec = c.Ops.ProbeTimeoutSeconds
		}
		return time.Duration(sec) * time.Second
	}

	var redisProbe, elasticProbe, rabbitProbe, consulProbe func(context.Context) error
	if d.HasRedis() {
		redisProbe = d.PingRedis
	}
	if d.HasElastic() {
		elasticProbe = d.PingElastic
	}
	if c.MQ.Addr != "" {
		rabbitProbe = pub.Health
	}
	if consulCli != nil {
		consulProbe = func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			_, err := consulCli.Status().Leader()
			return err
		}
	}

	return []ops.Check{
		{Name: "mysql", Critical: c.Ops.Readiness.MySQL.IsCritical(true), Timeout: timeout(c.Ops.Readiness.MySQL.TimeoutSeconds), Probe: d.PingDatabases},
		{Name: "redis", Critical: c.Ops.Readiness.Redis.IsCritical(true), Timeout: timeout(c.Ops.Readiness.Redis.TimeoutSeconds), Probe: redisProbe},
		{Name: "elasticsearch", Critical: c.Ops.Readiness.Elasticsearch.IsCritical(false), Timeout: timeout(c.Ops.Readiness.Elasticsearch.TimeoutSeconds), Probe: elasticProbe},
		{Name: "rabbitmq", Critical: c.Ops.Readiness.RabbitMQ.IsCritical(false), Timeout: timeout(c.Ops.Readiness.RabbitMQ.TimeoutSeconds), Probe: rabbitProbe},
		{Name: "consul", Critical: c.Ops.Readiness.Consul.IsCritical(false), Timeout: timeout(c.Ops.Readiness.Consul.TimeoutSeconds), Probe: consulProbe},
	}
}
