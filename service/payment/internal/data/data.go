package data

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	consul "github.com/go-kratos/kratos/contrib/registry/consul/v3"
	"github.com/go-kratos/kratos/v3/registry"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/google/wire"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"

	orderV1 "payment/api/service/order/v1"
	"payment/internal/biz"
	"payment/internal/conf"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewPaymentRepo, NewOrderServiceClient, NewRegistrar)

// Data owns the module's long-lived storage client (MySQL).
type Data struct {
	db          *sql.DB
	orderClient orderV1.OrderClient
}

// NewData opens the MySQL client described by the config.
func NewData(c *conf.Data, logger *slog.Logger) (*Data, func(), error) {
	dc := c.GetDatabase()
	if dc.GetDriver() != "mysql" {
		return nil, nil, fmt.Errorf("production database driver must be mysql")
	}
	db, err := sql.Open("mysql", dc.GetSource())
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		logger.Info("closing the data resources")
		_ = db.Close()
	}
	return &Data{db: db}, cleanup, nil
}

// execQuerier is satisfied by both *sql.DB and *sql.Tx so repositories can run
// raw SQL transparently inside or outside a transaction.
type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// DB returns the query handle for the current context.
func (d *Data) DB(ctx context.Context) execQuerier { return d.db }

// NewPaymentRepo builds the payment repository over the shared *Data.
func NewPaymentRepo(data *Data, logger *slog.Logger) biz.PaymentRepo {
	return &paymentRepo{data: data, log: logger}
}

// NewOrderServiceClient dials the order gRPC service directly. Service
// discovery and tracing are dropped from the target stack; the endpoint is a
// plain host:port address resolved by the gRPC resolver.
func NewOrderServiceClient(sr *conf.Service) orderV1.OrderClient {
	conn, err := kratosgrpc.NewClient(
		context.Background(),
		kratosgrpc.WithEndpoint(sr.GetOrder().GetEndpoint()),
		kratosgrpc.WithMiddleware(recovery.Recovery()),
		kratosgrpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	return orderV1.NewOrderClient(conn)
}

// NewRegistrar add consul
func NewRegistrar(conf *conf.Registry) registry.Registrar {
	c := consulAPI.DefaultConfig()
	c.Address = conf.Consul.Address
	c.Scheme = conf.Consul.Scheme
	cli, err := consulAPI.NewClient(c)
	if err != nil {
		panic(err)
	}
	r := consul.New(cli, consul.WithHealthCheck(true))
	return r
}
