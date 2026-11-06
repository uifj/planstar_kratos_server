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
	"github.com/redis/go-redis/v9"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"

	cartV1 "order/api/cart/v1"
	goodsV1 "order/api/goods/v1"
	userV1 "order/api/user/v1"
	"order/internal/biz"
	"order/internal/conf"
	"order/internal/pkg/mq"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewTransaction,
	NewOrderRepo,
	NewPublisher,
	NewUserServiceClient,
	NewCartServiceClient,
	NewGoodsServiceClient,
	NewRegistrar,
)

// Data owns the module's long-lived storage clients (MySQL + Redis).
type Data struct {
	db  *sql.DB
	rdb *redis.Client
}

type contextTxKey struct{}

// NewData opens the MySQL and Redis clients described by the config.
func NewData(c *conf.Data, logger *slog.Logger) (*Data, func(), error) {
	dc := c.GetDatabase()
	if dc.GetDriver() != "mysql" {
		return nil, nil, fmt.Errorf("production database driver must be mysql")
	}
	db, err := sql.Open("mysql", dc.GetSource())
	if err != nil {
		return nil, nil, err
	}
	rc := c.GetRedis()
	rdb := redis.NewClient(&redis.Options{
		Addr:         rc.GetAddr(),
		Password:     rc.GetPassword(),
		DB:           int(rc.GetDb()),
		DialTimeout:  rc.GetDialTimeout().AsDuration(),
		WriteTimeout: rc.GetWriteTimeout().AsDuration(),
		ReadTimeout:  rc.GetReadTimeout().AsDuration(),
	})
	cleanup := func() {
		logger.Info("closing the data resources")
		_ = rdb.Close()
		_ = db.Close()
	}
	return &Data{db: db, rdb: rdb}, cleanup, nil
}

// execQuerier is satisfied by both *sql.DB and *sql.Tx so repositories can run
// raw SQL transparently inside or outside a transaction.
type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// NewTransaction exposes the *Data transaction helper as a biz.Transaction.
func NewTransaction(d *Data) biz.Transaction {
	return d
}

// DB returns the transaction-bound handle when inside ExecTx, else the pool.
func (d *Data) DB(ctx context.Context) execQuerier {
	if tx, ok := ctx.Value(contextTxKey{}).(*sql.Tx); ok {
		return tx
	}
	return d.db
}

// Redis exposes the shared client to repository implementations.
func (d *Data) Redis() *redis.Client { return d.rdb }

// ExecTx runs fn inside a MySQL transaction, committing on success and rolling
// back on error. Nested calls reuse the ambient transaction.
func (d *Data) ExecTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(contextTxKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, contextTxKey{}, tx)
	if err := fn(ctx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("%w (rollback: %v)", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

// NewPublisher builds the RabbitMQ publisher over the configured address.
func NewPublisher(c *conf.Mq) *mq.Publisher {
	if c == nil {
		return nil
	}
	return mq.NewPublisher(c.Addr)
}

// NewUserServiceClient dials the user gRPC service directly. Service discovery
// and tracing are dropped from the target stack; the endpoint is a plain
// host:port address resolved by the gRPC resolver.
func NewUserServiceClient(sr *conf.Service) userV1.UserClient {
	conn, err := kratosgrpc.NewClient(
		context.Background(),
		kratosgrpc.WithEndpoint(sr.GetUser().GetEndpoint()),
		kratosgrpc.WithMiddleware(recovery.Recovery()),
		kratosgrpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	return userV1.NewUserClient(conn)
}

// NewCartServiceClient dials the cart gRPC service directly.
func NewCartServiceClient(sr *conf.Service) cartV1.CartClient {
	conn, err := kratosgrpc.NewClient(
		context.Background(),
		kratosgrpc.WithEndpoint(sr.GetCart().GetEndpoint()),
		kratosgrpc.WithMiddleware(recovery.Recovery()),
		kratosgrpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	return cartV1.NewCartClient(conn)
}

// NewGoodsServiceClient dials the goods gRPC service directly.
func NewGoodsServiceClient(sr *conf.Service) goodsV1.GoodsClient {
	conn, err := kratosgrpc.NewClient(
		context.Background(),
		kratosgrpc.WithEndpoint(sr.GetGoods().GetEndpoint()),
		kratosgrpc.WithMiddleware(recovery.Recovery()),
		kratosgrpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	return goodsV1.NewGoodsClient(conn)
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
