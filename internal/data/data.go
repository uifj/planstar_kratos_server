package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/data/ent"

	"github.com/go-kratos/kratos/v3/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/olivere/elastic/v7"
	"github.com/redis/go-redis/v9"
)

// ProviderSet is data providers. Every biz port is implemented here.
var ProviderSet = wire.NewSet(
	NewData,
	NewTodoRepo,
	NewUserRepo,
	NewAddressRepo,
	NewPermissionRepo,
	NewGoodsRepo,
	NewCartRepo,
	NewOrderRepo,
	NewInventoryRepo,
	NewPaymentRepo,
	NewOutboxRepo,
	NewConsumedEventRepo,
	NewPasswordHasher,
	NewTokenManager,
	NewCaptchaStore,
	NewGoodsSearch,
	NewStockCache,
	NewRateLimiter,
	NewCircuitBreakerFactory,
	NewAuthKeyProvider,
)

// Data holds the long-lived storage clients shared by repos.
type Data struct {
	db      *ent.Client
	rdb     redis.UniversalClient
	es      *elastic.Client
	esIndex string
}

// NewData opens all three clients and returns a single cleanup closure so
// Wire can own their lifecycle. Redis and Elasticsearch clients are lazy:
// construction never dials, so a failing dependency degrades gracefully.
func NewData(c *conf.Data, m *conf.Mall) (*Data, func(), error) {
	dc := c.GetDatabase()
	db, err := ent.Open(dc.GetDriver(), dc.GetSource())
	if err != nil {
		return nil, nil, err
	}
	if dc.GetDebug() {
		db = db.Debug()
	}
	if dc.GetAutoMigrate() {
		if err := db.Schema.Create(context.Background()); err != nil {
			db.Close()
			return nil, nil, err
		}
	}

	redisAddr := "127.0.0.1:6379"
	if rc := c.GetRedis(); rc != nil && rc.GetAddr() != "" {
		redisAddr = rc.GetAddr()
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	esAddrs := []string{"http://127.0.0.1:9200"}
	esIndex := "goods"
	if m != nil {
		if s := m.GetSearch(); s != nil {
			if len(s.GetAddresses()) > 0 {
				esAddrs = s.GetAddresses()
			}
			if s.GetGoodsIndex() != "" {
				esIndex = s.GetGoodsIndex()
			}
		}
	}
	esClient, err := elastic.NewClient(
		elastic.SetURL(esAddrs...),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)
	if err != nil {
		_ = rdb.Close()
		db.Close()
		return nil, nil, err
	}

	d := &Data{db: db, rdb: rdb, es: esClient}
	d.esIndex = esIndex

	cleanup := func() {
		log.Info("closing the data resources")
		if err := db.Close(); err != nil {
			log.Error("failed closing the database", "err", err)
		}
		if err := rdb.Close(); err != nil {
			log.Error("failed closing redis", "err", err)
		}
		esClient.Stop()
	}
	return d, cleanup, nil
}
