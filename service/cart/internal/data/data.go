package data

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/go-sql-driver/mysql"
	consul "github.com/go-kratos/kratos/contrib/registry/consul/v3"
	"github.com/go-kratos/kratos/v3/registry"
	consulAPI "github.com/hashicorp/consul/api"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"

	"cart/internal/conf"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewCartRepo,
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
