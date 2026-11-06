package data

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/olivere/elastic/v7"

	"github.com/go-kratos/kratos/v3/middleware/recovery"
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"

	inventoryV1 "goods/api/service/inventory/v1"
	"goods/internal/biz"
	"goods/internal/conf"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewTransaction,
	NewBrandRepo,
	NewCategoryRepo,
	NewGoodsTypeRepo,
	NewSpecificationRepo,
	NewGoodsAttrRepo,
	NewGoodsRepo,
	NewGoodsSkuRepoRepo,
	NewInventoryRepo,
	NewEsGoodsRepo,
	NewInventoryServiceClient,
)

// Data owns the module's long-lived storage clients (MySQL + Elasticsearch).
type Data struct {
	db       *sql.DB
	EsClient *elastic.Client
}

type contextTxKey struct{}

// execQuerier is satisfied by both *sql.DB and *sql.Tx so repositories can run
// raw SQL transparently inside or outside a transaction.
type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// NewData opens the MySQL and Elasticsearch clients described by the config.
func NewData(c *conf.Data, logger *slog.Logger) (*Data, func(), error) {
	dc := c.GetDatabase()
	if dc.GetDriver() != "mysql" {
		return nil, nil, fmt.Errorf("production database driver must be mysql")
	}
	db, err := sql.Open("mysql", dc.GetSource())
	if err != nil {
		return nil, nil, err
	}
	ec := c.GetElastic()
	es, err := elastic.NewClient(
		elastic.SetURL(ec.GetAddr()),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		logger.Info("closing the data resources")
		_ = db.Close()
	}
	return &Data{db: db, EsClient: es}, cleanup, nil
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

// NewInventoryServiceClient dials the inventory gRPC service directly. Service
// discovery and tracing are dropped from the target stack; the endpoint is a
// plain host:port address resolved by the gRPC resolver.
func NewInventoryServiceClient(sr *conf.Service) inventoryV1.InventoryClient {
	conn, err := kratosgrpc.NewClient(
		context.Background(),
		kratosgrpc.WithEndpoint(sr.GetInventory().GetEndpoint()),
		kratosgrpc.WithMiddleware(recovery.Recovery()),
		kratosgrpc.WithTimeout(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	return inventoryV1.NewInventoryClient(conn)
}
