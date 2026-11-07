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

	"github.com/XSAM/otelsql"
	"github.com/redis/go-redis/extra/redisotel/v9"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"planstar/internal/conf"
)

var (
	// otelDriverOnce 保证 otelsql 驱动只注册一次（9 个库共享同一驱动）。
	otelDriverOnce sync.Once
	otelDriverName string
)

// mysqlDriver 返回带 OTel 埋点的 MySQL 驱动名；注册失败时回退到原生 mysql。
func mysqlDriver(log *slog.Logger) string {
	otelDriverOnce.Do(func() {
		name, err := otelsql.Register("mysql", otelsql.WithAttributes(semconv.DBSystemMySQL))
		if err != nil {
			otelDriverName = "mysql"
			if log != nil {
				log.Warn("register otelsql driver failed, fallback to mysql", "err", err)
			}
			return
		}
		otelDriverName = name
	})
	return otelDriverName
}

// domains 是 9 个服务库的域名，data 层按库连接。
var domains = []string{
	"user", "catalog", "ticket", "order", "payment",
	"marketing", "community", "content", "meta", "shop",
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
		// Redis 命令埋点：沿用全局 TracerProvider，未开启 tracing 时自动 no-op。
		if err := redisotel.InstrumentTracing(rdb); err != nil {
			log.Warn("instrument redis tracing failed", "err", err)
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
	db, err := sql.Open(mysqlDriver(d.log), dsn)
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

// ElasticIndex 是演出库读模型的查询别名（具体索引由全量重建创建并原子切换）。
func (d *Data) ElasticIndex() string { return d.esIndex }

// ElasticTimeout ES 查询超时。
func (d *Data) ElasticTimeout() time.Duration { return d.esTimeout }

// Health 汇总所有依赖的可用性，供 /readyz 使用（保留旧入口，内部委托给分项探测）。
func (d *Data) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return errors.Join(
		d.PingDatabases(ctx),
		d.PingRedis(ctx),
		d.PingElastic(ctx),
	)
}

// PingDatabases 探测当前已打开的全部 MySQL 连接池，逐库聚合错误（错误文本带库名）。
func (d *Data) PingDatabases(ctx context.Context) error {
	d.mu.Lock()
	dbs := make(map[string]*sql.DB, len(d.dbs))
	for name, db := range d.dbs {
		dbs[name] = db
	}
	d.mu.Unlock()

	var errs []error
	for name, db := range dbs {
		if db == nil {
			continue
		}
		if err := db.PingContext(ctx); err != nil {
			errs = append(errs, fmt.Errorf("mysql %s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

// PingRedis 探测 Redis；未配置时返回 nil（由调用方决定是否跳过）。
func (d *Data) PingRedis(ctx context.Context) error {
	if d.rdb == nil {
		return nil
	}
	if err := d.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	return nil
}

// PingElastic 探测 Elasticsearch；未配置时返回 nil（由调用方决定是否跳过）。
func (d *Data) PingElastic(ctx context.Context) error {
	if d.es == nil {
		return nil
	}
	if _, err := d.es.ClusterHealth().Do(ctx); err != nil {
		return fmt.Errorf("elasticsearch: %w", err)
	}
	return nil
}

// DBStatsSnapshot 在抓取指标时快照各连接池状态，避免长时间持锁。
func (d *Data) DBStatsSnapshot() map[string]sql.DBStats {
	d.mu.Lock()
	defer d.mu.Unlock()
	stats := make(map[string]sql.DBStats, len(d.dbs))
	for name, db := range d.dbs {
		if db == nil {
			continue
		}
		stats[name] = db.Stats()
	}
	return stats
}

// HasRedis 返回是否配置了 Redis。
func (d *Data) HasRedis() bool { return d.rdb != nil }

// HasElastic 返回是否配置了 Elasticsearch。
func (d *Data) HasElastic() bool { return d.es != nil }

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
