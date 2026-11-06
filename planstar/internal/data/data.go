package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/olivere/elastic/v7"
	"github.com/redis/go-redis/v9"

	"planstar/internal/conf"
)

// domains 是 9 个服务库的域名，data 层按库连接。
var domains = []string{
	"user", "catalog", "ticket", "order", "payment",
	"marketing", "community", "content", "meta",
}

// Data 持有 9 个 planstar_* 库的连接池，以及 Redis / Elasticsearch 长连接。
type Data struct {
	mu    sync.Mutex
	dbs   map[string]*sql.DB
	mysql conf.MySQL
	rdb   *redis.Client
	es    *elastic.Client

	esIndex   string
	esTimeout time.Duration
	log       *slog.Logger
}

func NewData(c conf.Data, log *slog.Logger) (*Data, error) {
	d := &Data{
		dbs:       make(map[string]*sql.DB, len(domains)),
		mysql:     c.MySQL,
		esIndex:   c.Elastic.IndexPrefix + "-activities",
		esTimeout: time.Duration(c.Elastic.RequestTimeout) * time.Second,
		log:       log,
	}
	for _, domain := range domains {
		if _, err := d.DB(domain); err != nil {
			return nil, err
		}
	}
	if c.Redis.Addr != "" {
		rdb := redis.NewClient(&redis.Options{
			Addr:         c.Redis.Addr,
			Password:     c.Redis.Password,
			DB:           c.Redis.DB,
			DialTimeout:  time.Second,
			ReadTimeout:  300 * time.Millisecond,
			WriteTimeout: 300 * time.Millisecond,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("redis %s: %w", c.Redis.Addr, err)
		}
		d.rdb = rdb
	}
	if c.Elastic.Addr != "" {
		es, err := elastic.NewClient(
			elastic.SetURL(c.Elastic.Addr),
			elastic.SetSniff(false),
			elastic.SetHealthcheck(false),
			elastic.SetMaxRetries(2),
			elastic.SetHealthcheckTimeoutStartup(5*time.Second),
		)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch %s: %w", c.Elastic.Addr, err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := es.ClusterHealth().Do(ctx); err != nil {
			return nil, fmt.Errorf("elasticsearch %s: %w", c.Elastic.Addr, err)
		}
		d.es = es
	}
	return d, nil
}

func (d *Data) DB(domain string) (*sql.DB, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if db, ok := d.dbs[domain]; ok && db != nil {
		return db, nil
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/planstar_%s?parseTime=true&charset=%s&loc=Local",
		d.mysql.User, d.mysql.Password, d.mysql.Addr, domain, d.mysql.Charset)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("planstar_%s: %w", domain, err)
	}
	d.dbs[domain] = db
	return db, nil
}

// Redis 返回共享 Redis 客户端（未配置时为 nil）。
func (d *Data) Redis() *redis.Client { return d.rdb }

// Elastic 返回共享 Elasticsearch 客户端（未配置时为 nil）。
func (d *Data) Elastic() *elastic.Client { return d.es }

// ElasticIndex 是演出库读模型的索引名。
func (d *Data) ElasticIndex() string { return d.esIndex }

// ElasticTimeout ES 查询超时。
func (d *Data) ElasticTimeout() time.Duration { return d.esTimeout }

// Health 汇总所有依赖的可用性，供 /readyz 使用。
func (d *Data) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var errs []error
	d.mu.Lock()
	dbs := make([]*sql.DB, 0, len(d.dbs))
	for _, db := range d.dbs {
		dbs = append(dbs, db)
	}
	d.mu.Unlock()
	for _, db := range dbs {
		if err := db.PingContext(ctx); err != nil {
			errs = append(errs, err)
			break
		}
	}
	if d.rdb != nil {
		if err := d.rdb.Ping(ctx).Err(); err != nil {
			errs = append(errs, fmt.Errorf("redis: %w", err))
		}
	}
	if d.es != nil {
		if _, err := d.es.ClusterHealth().Do(ctx); err != nil {
			errs = append(errs, fmt.Errorf("elasticsearch: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (d *Data) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, db := range d.dbs {
		if db != nil {
			_ = db.Close()
		}
	}
	d.dbs = nil
	if d.rdb != nil {
		_ = d.rdb.Close()
	}
}
