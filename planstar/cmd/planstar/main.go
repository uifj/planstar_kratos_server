package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/contrib/registry/consul/v3"
	"github.com/go-kratos/kratos/v3"
	klog "github.com/go-kratos/kratos/v3/log"
	"github.com/hashicorp/consul/api"
	_ "go.uber.org/automaxprocs"

	"planstar/internal/biz"
	"planstar/internal/conf"
	"planstar/internal/data"
	"planstar/internal/mq"
	"planstar/internal/observability"
	"planstar/internal/ops"
	"planstar/internal/server"
	"planstar/internal/service"
)

var (
	Name    = "planstar.api"
	Version = "planstar.api.v1"
	// flagconf 是配置路径。
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "configs/config.yaml", "config path, eg: -conf configs/config.yaml")
}

func main() {
	flag.Parse()

	logger := klog.NewLogger(observability.NewTraceHandler(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo}))).
		With(slog.String("service.id", id), slog.String("service.name", Name), slog.String("service.version", Version))
	klog.SetDefault(logger)

	// Kratos 文档约定：Run 返回错误视为进程级失败，返回非零退出码；
	// 该 defer 最先注册，保证已注册的清理函数（关闭 MySQL/MQ/tracing）先执行。
	exitCode := 0
	defer func() {
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()

	c, err := conf.Load(flagconf)
	if err != nil {
		logger.Error("load config failed", "path", flagconf, "err", err)
		os.Exit(1)
	}

	// 指标与运维鉴权在装配早期构建：鉴权配置错误直接失败关闭。
	metrics := observability.NewMetrics(Version, c.Server.Env)
	metrics.SetScrapeLimits(
		c.Ops.MetricsMaxInflight,
		time.Duration(c.Ops.MetricsTimeoutSeconds)*time.Second,
	)
	opsAuth, err := server.NewOpsAuth(c.Ops, logger, metrics)
	if err != nil {
		logger.Error("init ops auth failed", "err", err)
		os.Exit(1)
	}
	if err := opsAuth.Validate(c.Server.Env); err != nil {
		logger.Error("invalid ops auth config", "err", err)
		os.Exit(1)
	}
	if !opsAuth.Enabled() {
		logger.Warn("ops auth disabled: PLANSTAR_OPS_TOKEN 未配置")
	}

	// 链路追踪：endpoint 为空时自动降级为 no-op。
	shutdownTrace, err := observability.InitTracer(c.Trace.Endpoint, c.Trace.InsecureEnabled(), c.Trace.Env, c.Trace.SampleRatio, Name, Version, logger)
	if err != nil {
		logger.Error("init tracer failed", "err", err)
	} else {
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = shutdownTrace(ctx)
		}()
	}

	d, err := data.NewData(c.Data, logger)
	if err != nil {
		logger.Error("init data clients failed", "err", err)
		os.Exit(1)
	}
	defer d.Close()

	// 事件发布：MQ 不可用时退化为 no-op，不影响同步链路。
	var publisher biz.EventPublisher = biz.NopEventPublisher{}
	mqPublisher := mq.NewPublisher(c.MQ.Addr, c.MQ.Exchange)
	if c.MQ.Addr != "" {
		publisher = mqPublisher
	}
	defer mqPublisher.Close()

	// Consul 客户端同时用于服务注册与 consul 探针，只创建一次。
	var consulCli *api.Client
	if c.Registry.Consul.Address != "" {
		cli, err := api.NewClient(&api.Config{
			Address: c.Registry.Consul.Address,
			Scheme:  c.Registry.Consul.Scheme,
		})
		if err != nil {
			logger.Warn("init consul client failed", "err", err)
		} else {
			consulCli = cli
		}
	}

	authUC := biz.NewAuthUsecase(
		data.NewAuthRepo(d),
		data.NewAuthCache(d),
		publisher,
		logger,
		time.Duration(c.Auth.SessionExpireDays)*24*time.Hour,
	)
	catalogRepo := data.NewCatalogRepo(d)
	catalogUC := biz.NewCatalogUsecase(catalogRepo)
	feedUC := biz.NewFeedUsecase(data.NewFeedRepo(d))
	messageUC := biz.NewMessageUsecase(data.NewMessageRepo(d))
	orderUC := biz.NewOrderUsecase(data.NewOrderRepo(d), publisher, logger)
	userUC := biz.NewUserUsecase(data.NewUserRepo(d))
	projectUC := biz.NewProjectUsecase(data.NewProjectRepo(d))
	paymentUC := biz.NewPaymentUsecase(data.NewPaymentRepo(d))
	chatUC := biz.NewChatUsecase(data.NewChatRepo(d))

	// ES 读模型：启动时建索引并做一次全量同步，之后周期刷新。
	if d.Elastic() != nil {
		if err := catalogRepo.EnsureIndex(context.Background()); err != nil {
			logger.Warn("ensure elasticsearch index failed", "err", err)
		}
		go catalogRepo.RunSync(context.Background(), time.Duration(c.Data.Elastic.SyncInterval)*time.Second, logger)
	}

	mux := http.NewServeMux()
	service.New(authUC, catalogUC, feedUC, messageUC, orderUC, userUC, projectUC, paymentUC, chatUC, logger).Register(mux)

	// 连接池指标在抓取时读取快照，不在后台轮询。
	metrics.RegisterDBStats(d.DBStatsSnapshot)
	// 目录降级计数注册到 /metrics 使用的独立 Registry。
	data.RegisterCatalogMetrics(metrics.Registry)
	// 业务指标（登录、报价、下单）同样注册到 /metrics。
	service.RegisterBizMetrics(metrics.Registry)

	// 就绪探针：未配置的依赖 Probe 为 nil，会被标记为 skipped。
	// 聚合状态变化时写一条审计日志，便于与告警时间线对齐。
	var probeStatus atomic.Value
	readiness := ops.NewReadiness(buildOpsChecks(c, d, mqPublisher, consulCli), ops.Options{
		Service:      Name,
		Version:      Version,
		TTL:          time.Duration(c.Ops.ProbeCacheTTLSeconds) * time.Second,
		TotalTimeout: time.Duration(c.Ops.ProbeTotalTimeoutSeconds) * time.Second,
		OnResult: func(results []ops.CheckResult) {
			metrics.ObserveProbe("readiness", results, false)
			status := aggregateProbeStatus(results)
			if prev, _ := probeStatus.Load().(string); prev == status {
				return
			}
			probeStatus.Store(status)
			logger.InfoContext(context.Background(), "ops audit",
				"audit", true,
				"actor", "system",
				"action", "ops.probe.state",
				"target", "readyz",
				"result", status,
			)
			metrics.IncOpsAudit("ops.probe.state", status)
		},
	})

	hs := server.NewHTTPServer(c.Server, mux, metrics, logger)
	adminSrv := server.NewAdminServer(c.Server, metrics, readiness, Version, opsAuth, biz.NewReportUsecase(data.NewReportRepo(d)), projectUC, logger)

	// 异步事件消费者：随 kratos.App 一起启停。
	eventHandler := service.NewEventHandler(messageUC, catalogRepo, logger)
	mqSrv := mq.NewServer(c.MQ, eventHandler.Handle, logger)

	opts := []kratos.Option{
		kratos.ID(id + "-" + Name),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{"env": c.Trace.Env}),
		kratos.Logger(logger),
		kratos.Server(hs, adminSrv, mqSrv),
		// 优雅停机：给在途请求与 MQ 消费者留出收尾时间，并限制注册中心注销耗时。
		kratos.StopTimeout(15 * time.Second),
		kratos.RegistrarTimeout(5 * time.Second),
		kratos.BeforeStart(func(ctx context.Context) error {
			logger.InfoContext(ctx, "application starting",
				"http_addr", c.Server.HTTP.Addr, "admin_addr", c.Server.Admin.Addr)
			return nil
		}),
		kratos.AfterStart(func(ctx context.Context) error {
			readiness.MarkStarted()
			logger.InfoContext(ctx, "application started", "service", Name, "version", Version)
			return nil
		}),
		kratos.BeforeStop(func(ctx context.Context) error {
			// 先关闭就绪闸门，让负载均衡把流量摘走，再执行优雅停机。
			readiness.MarkStopping()
			logger.InfoContext(ctx, "application stopping", "service", Name)
			return nil
		}),
		kratos.AfterStop(func(ctx context.Context) error {
			logger.InfoContext(ctx, "application stopped", "service", Name)
			return nil
		}),
	}

	// Consul 服务注册：仅注册业务端口，运维端口不进服务发现。
	if c.Registry.Consul.Address != "" {
		host := c.Registry.Consul.ServiceHost
		if _, port, err := net.SplitHostPort(c.Server.HTTP.Addr); err == nil {
			apiURL := &url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}
			opts = append(opts, kratos.Endpoint(apiURL))
		}
		if consulCli != nil {
			opts = append(opts, kratos.Registrar(consul.New(consulCli, consul.WithHealthCheck(true))))
			logger.Info("consul registry enabled", "address", c.Registry.Consul.Address)
		}
	}

	app := kratos.New(opts...)
	if err := app.Run(); err != nil {
		logger.Error("application exited with error", "err", err)
		exitCode = 1
	}
}

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
