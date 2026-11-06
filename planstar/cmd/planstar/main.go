package main

import (
	"context"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
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

	c, err := conf.Load(flagconf)
	if err != nil {
		panic(err)
	}

	// 链路追踪：endpoint 为空时自动降级为 no-op。
	shutdownTrace, err := observability.InitTracer(c.Trace.Endpoint, c.Trace.Env, c.Trace.SampleRatio, Name, logger)
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
		panic(err)
	}
	defer d.Close()

	// 事件发布：MQ 不可用时退化为 no-op，不影响同步链路。
	var publisher biz.EventPublisher = biz.NopEventPublisher{}
	mqPublisher := mq.NewPublisher(c.MQ.Addr, c.MQ.Exchange)
	if c.MQ.Addr != "" {
		publisher = mqPublisher
	}
	defer mqPublisher.Close()

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

	// ES 读模型：启动时建索引并做一次全量同步，之后周期刷新。
	if d.Elastic() != nil {
		if err := catalogRepo.EnsureIndex(context.Background()); err != nil {
			logger.Warn("ensure elasticsearch index failed", "err", err)
		}
		go catalogRepo.RunSync(context.Background(), time.Duration(c.Data.Elastic.SyncInterval)*time.Second, logger)
	}

	mux := http.NewServeMux()
	service.New(authUC, catalogUC, feedUC, messageUC, orderUC, userUC, logger).Register(mux)

	metrics := observability.NewMetrics()
	hs := server.NewHTTPServer(c.Server, mux, metrics, logger)
	adminSrv := server.NewAdminServer(c.Server, metrics.Handler(), d.Health, logger)

	// 异步事件消费者：随 kratos.App 一起启停。
	eventHandler := service.NewEventHandler(messageUC, logger)
	mqSrv := mq.NewServer(c.MQ, eventHandler.Handle, logger)

	opts := []kratos.Option{
		kratos.ID(id + "-" + Name),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{"env": c.Trace.Env}),
		kratos.Logger(logger),
		kratos.Server(hs, adminSrv, mqSrv),
	}

	// Consul 服务注册：仅注册业务端口，运维端口不进服务发现。
	if c.Registry.Consul.Address != "" {
		host := c.Registry.Consul.ServiceHost
		if _, port, err := net.SplitHostPort(c.Server.HTTP.Addr); err == nil {
			apiURL := &url.URL{Scheme: "http", Host: net.JoinHostPort(host, port)}
			opts = append(opts, kratos.Endpoint(apiURL))
		}
		cli, err := api.NewClient(&api.Config{
			Address: c.Registry.Consul.Address,
			Scheme:  c.Registry.Consul.Scheme,
		})
		if err != nil {
			logger.Warn("init consul client failed", "err", err)
		} else {
			opts = append(opts, kratos.Registrar(consul.New(cli, consul.WithHealthCheck(true))))
			logger.Info("consul registry enabled", "address", c.Registry.Consul.Address)
		}
	}

	app := kratos.New(opts...)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
