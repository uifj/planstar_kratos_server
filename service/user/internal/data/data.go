package data

import (
	"context"
	"fmt"
	"log/slog"

	"entgo.io/ent/dialect"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"

	"user/internal/biz"
	"user/internal/conf"
	"user/internal/data/ent"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewUserRepo, NewAddressRepo, NewTransaction)

// Data owns the module's long-lived storage clients.
type Data struct {
	db  *ent.Client
	rdb *redis.Client
}

type contextTxKey struct{}

func NewData(c *conf.Data, logger *slog.Logger) (*Data, func(), error) {
	dc := c.GetDatabase()
	if dc.GetDriver() != "mysql" {
		return nil, nil, fmt.Errorf("production database driver must be mysql")
	}
	drv := dialect.MySQL
	client, err := ent.Open(drv, dc.GetSource())
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
	helper := logger
	cleanup := func() {
		helper.Info("closing the data resources")
		_ = rdb.Close()
		_ = client.Close()
	}
	return &Data{db: client, rdb: rdb}, cleanup, nil
}

// NewForTest wraps a caller-supplied Ent client for repository tests.
func NewForTest(db *ent.Client) *Data { return &Data{db: db} }

// DB returns the transaction-bound client when inside ExecTx, else the base client.
func (d *Data) DB(ctx context.Context) *ent.Client {
	if tx, ok := ctx.Value(contextTxKey{}).(*ent.Tx); ok {
		return tx.Client()
	}
	return d.db
}

// Redis exposes the shared client to repository implementations.
func (d *Data) Redis() *redis.Client { return d.rdb }

func (d *Data) ExecTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(contextTxKey{}).(*ent.Tx); ok {
		return fn(ctx)
	}
	tx, err := d.db.Tx(ctx)
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

func NewTransaction(d *Data) biz.Transaction { return d }
