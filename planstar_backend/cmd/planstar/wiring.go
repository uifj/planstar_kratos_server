package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kratos/kratos/contrib/registry/consul/v3"
	"github.com/go-kratos/kratos/v3"
	klog "github.com/go-kratos/kratos/v3/log"
	"github.com/hashicorp/consul/api"
	_ "go.uber.org/automaxprocs"

	"planstar/internal/biz"
	"planstar/internal/data"
	"planstar/internal/platform/conf"
	"planstar/internal/platform/mq"
	"planstar/internal/platform/observability"
	"planstar/internal/platform/ops"
	"planstar/internal/platform/pkg/notify"
	"planstar/internal/platform/pkg/storage"
	"planstar/internal/server"
	"planstar/internal/service"
	"planstar/internal/service/events"
)

// run 是唯一的装配点（ADR-0001 D3 / ADR-0002 §5）：
// 一、可观测与运维闸门  二、数据与共享内核  三、按域用例  四、传输层  五、就绪与生命周期。
// 资源清理按注册逆序 LIFO：后台任务 -> MQ -> Data -> tracing（由 main 的退出码兜底）。
func run() error {
	// 结构化 JSON 日志：Loki/Promtail 直接索引字段，traceId 可关联 Jaeger；
	// 对齐阿里手册「日志规约」禁止 System.out/明文敏感信息的要求。
	logger := klog.NewLogger(observability.NewTraceHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: slog.LevelInfo}))).
		With(slog.String("service.id", id), slog.String("service.name", Name), slog.String("service.version", Version))
	klog.SetDefault(logger)

	c, err := conf.Load(flagconf)
	if err != nil {
		logger.Error("load config failed", "path", flagconf, "err", err)
		return err
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
		return err
	}
	if err := opsAuth.Validate(c.Server.Env); err != nil {
		logger.Error("invalid ops auth config", "err", err)
		return err
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
		return err
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

	// 生产强制接入已购买的阿里云短信（NewSMS 失败即终止启动）；
	// 本地版本使用日志实现，且日志只输出验证码指纹不输出明文。
	smsSender, err := notify.NewSMS(c.Notify.SMS, c.Server.Env, logger)
	if err != nil {
		logger.Error("init sms sender failed", "err", err)
		return err
	}

	// 对象存储：未启用返回 Noop，但装配点显式构造并注入 Service，
	// 避免 conf.Storage.OSS 与 pkg/storage 处于「配置已写、能力未接线」状态（OPEN_ITEMS C2）。
	objectStore, err := storage.NewObjectStorage(c.Storage.OSS, logger)
	if err != nil {
		logger.Error("init object storage failed", "err", err)
		return err
	}

	// 按环境装配认证仓储：仅本地版本允许测试口令；
	// 验证码以 HMAC 摘要入库，密钥由配置或运维令牌派生。
	authRepo := data.NewAuthRepoForEnvironment(
		d,
		c.Auth.TestCredentialPasswordValue(),
		c.Auth.CodeHMACKeyValue(c.Ops.Token),
	)
	// 生产启动断言：库里不允许残留任何测试/弱口令账号。
	if conf.IsProductionLikeEnv(c.Server.Env) {
		guardCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := authRepo.AssertNoTestCredentials(guardCtx, []string{
			conf.LocalTestCredentialPassword,
			c.Auth.TestCredentialPassword,
		})
		cancel()
		if err != nil {
			logger.Error("production credential guard failed", "err", err)
			return err
		}
	}
	authUC := biz.NewAuthUsecase(
		authRepo,
		data.NewAuthCache(d),
		publisher,
		smsSender,
		logger,
		time.Duration(c.Auth.SessionExpireDays)*24*time.Hour,
	)
	catalogRepo := data.NewCatalogRepo(d)
	catalogUC := biz.NewCatalogUsecase(catalogRepo)
	feedUC := biz.NewFeedUsecase(data.NewFeedRepo(d))
	messageUC := biz.NewMessageUsecase(data.NewMessageRepo(d))
	orderUC := biz.NewOrderUsecase(data.NewOrderRepo(d), publisher, logger, conf.QuoteTTL())
	userUC := biz.NewUserUsecase(data.NewUserRepo(d))
	projectUC := biz.NewProjectUsecase(data.NewProjectRepo(d))
	projectIdeaUC := biz.NewProjectIdeaUsecase(data.NewProjectIdeaRepo(d))
	paymentUC := biz.NewPaymentUsecase(data.NewPaymentRepo(d))
	walletUC := biz.NewWalletUsecase(data.NewWalletRepo(d))
	shopUC := biz.NewShopUsecase(data.NewShopRepo(d))
	chatUC := biz.NewChatUsecase(data.NewChatRepo(d))

	// 所有后台 goroutine 绑定到应用生命周期 context，停机时统一取消并等待收尾，
	// 避免优雅停机后同步任务继续访问已关闭的 DB/ES（阿里手册并发处理规约）。
	appCtx, cancelApp := context.WithCancel(context.Background())
	var background sync.WaitGroup
	defer func() {
		cancelApp()
		background.Wait()
	}()

	// ES 读模型：启动时建索引并做一次全量同步，之后周期刷新。
	if d.Elastic() != nil {
		indexCtx, cancelIndex := context.WithTimeout(appCtx, 10*time.Second)
		if err := catalogRepo.EnsureIndex(indexCtx); err != nil {
			logger.Warn("ensure elasticsearch index failed", "err", err)
		}
		cancelIndex()
		background.Add(1)
		go func() {
			defer background.Done()
			catalogRepo.RunSync(appCtx, time.Duration(c.Data.Elastic.SyncInterval)*time.Second, logger)
		}()
	}

	mux := http.NewServeMux()
	svc := service.New(authUC, catalogUC, feedUC, messageUC, orderUC, userUC, projectUC,
		projectIdeaUC, paymentUC, walletUC, shopUC, chatUC, logger, objectStore)
	svc.Register(mux)
	logger.Info("object storage wired", "enabled", svc.ObjectStorageEnabled())

	// 连接池指标在抓取时读取快照，不在后台轮询。
	metrics.RegisterDBStats(d.DBStatsSnapshot)
	// 目录降级计数注册到 /metrics 使用的独立 Registry。
	data.RegisterCatalogMetrics(metrics.Registry)
	// 业务指标（登录、报价、下单）同样注册到 /metrics。
	service.RegisterBizMetrics(metrics.Registry)
	// MQ 消费/发布/死信指标（ADR-0004），与业务指标共用同一独立 Registry。
	mq.RegisterMQMetrics(metrics.Registry)

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
	adminSrv := server.NewAdminServer(c.Server, metrics, readiness, Version, opsAuth, biz.NewReportUsecase(data.NewReportRepo(d)), projectUC, biz.NewOpsUsecase(data.NewOpsRepo(d)), logger)

	// 异步事件消费者：随 kratos.App 一起启停。
	eventHandler := events.NewEventHandler(messageUC, catalogRepo, logger)
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
			// 先关闭就绪闸门，让负载均衡把流量摘走，再取消后台任务。
			readiness.MarkStopping()
			cancelApp()
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
		return err
	}
	return nil
}
