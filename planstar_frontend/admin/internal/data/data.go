package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/types/known/durationpb"

	"planstar/admin/internal/conf"
	"planstar/admin/internal/data/ent"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewAdminRepo, NewRevocationStore, NewAttemptLimiter, NewConsoleRepo, NewOpsClient, NewAuditRepo)

const (
	defaultMaxOpenConns    = 20
	defaultMaxIdleConns    = 10
	defaultConnMaxLifetime = 30 * time.Minute
	defaultRedisAddr       = "127.0.0.1:6379"
	startupPingTimeout     = 5 * time.Second
)

// Data holds the long-lived storage clients shared by every repo. Repos receive
// *Data and never construct clients of their own.
type Data struct {
	// db is the ent client used by the generated repository builders.
	db *ent.Client
	// readDB is a plain database/sql pool for queries and writes that do not
	// need ent (health checks, audit rows).
	readDB *sql.DB
	// redis backs login throttling, sessions and short-lived caches.
	redis *redis.Client
}

// NewData creates the storage clients. It returns an error instead of exiting
// the process so the caller decides how to report a bad configuration.
func NewData(c *conf.Data, s *conf.Server) (*Data, func(), error) {
	dc := c.GetDatabase()
	if dc.GetAutoMigrate() && isProduction(s.GetEnv()) {
		return nil, nil, fmt.Errorf(
			"data.database.auto_migrate must be false when server.env is %q; apply planstar/sql migrations instead",
			s.GetEnv(),
		)
	}

	db, err := ent.Open("mysql", dc.GetSource())
	if err != nil {
		return nil, nil, fmt.Errorf("open ent client: %w", err)
	}
	if dc.GetDebug() {
		db = db.Debug()
	}

	readDB, err := sql.Open("mysql", dc.GetSource())
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("open mysql pool: %w", err)
	}
	readDB.SetMaxOpenConns(poolInt(dc.GetMaxOpenConns(), defaultMaxOpenConns))
	readDB.SetMaxIdleConns(poolInt(dc.GetMaxIdleConns(), defaultMaxIdleConns))
	readDB.SetConnMaxLifetime(poolDuration(dc.GetConnMaxLifetime(), defaultConnMaxLifetime))

	pingCtx, cancel := context.WithTimeout(context.Background(), startupPingTimeout)
	defer cancel()
	if err := readDB.PingContext(pingCtx); err != nil {
		_ = readDB.Close()
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping mysql: %w", err)
	}

	rc := newRedisClient(c.GetRedis())

	// Auto migration is a convenience for local development only; production
	// schema changes are applied through the reviewed SQL migrations.
	if dc.GetAutoMigrate() {
		if err := db.Schema.Create(context.Background()); err != nil {
			_ = rc.Close()
			_ = readDB.Close()
			_ = db.Close()
			return nil, nil, fmt.Errorf("auto migrate schema: %w", err)
		}
	}

	d := &Data{
		db:     db,
		readDB: readDB,
		redis:  rc,
	}
	return d, d.Cleanup, nil
}

// DB exposes the ent client used by the repositories.
func (d *Data) DB() *ent.Client {
	return d.db
}

// SQL exposes the plain database/sql pool for queries and writes that are
// awkward to express with ent.
func (d *Data) SQL() *sql.DB {
	return d.readDB
}

// Redis exposes the shared Redis client.
func (d *Data) Redis() *redis.Client {
	return d.redis
}

// Ready reports whether both MySQL and Redis accept commands. It backs the
// /readyz probe; startup only pings MySQL, so a temporarily unavailable Redis
// surfaces here instead of preventing the process from starting.
func (d *Data) Ready(ctx context.Context) error {
	var errs []error
	if err := d.readDB.PingContext(ctx); err != nil {
		errs = append(errs, fmt.Errorf("mysql not ready: %w", err))
	}
	if err := d.redis.Ping(ctx).Err(); err != nil {
		errs = append(errs, fmt.Errorf("redis not ready: %w", err))
	}
	return errors.Join(errs...)
}

// Cleanup closes every long-lived client. It is safe to call from the Wire
// cleanup hook and releases resources in reverse creation order.
func (d *Data) Cleanup() {
	_ = d.redis.Close()
	_ = d.readDB.Close()
	_ = d.db.Close()
}

func newRedisClient(c *conf.Data_Redis) *redis.Client {
	network := c.GetNetwork()
	if network == "" {
		network = "tcp"
	}
	addr := c.GetAddr()
	if addr == "" {
		addr = defaultRedisAddr
	}
	return redis.NewClient(&redis.Options{
		Network:      network,
		Addr:         addr,
		Password:     c.GetPassword(),
		DB:           int(c.GetDb()),
		ReadTimeout:  c.GetReadTimeout().AsDuration(),
		WriteTimeout: c.GetWriteTimeout().AsDuration(),
	})
}

func poolInt(v, fallback int32) int {
	if v <= 0 {
		return int(fallback)
	}
	return int(v)
}

func poolDuration(d *durationpb.Duration, fallback time.Duration) time.Duration {
	if v := d.AsDuration(); v > 0 {
		return v
	}
	return fallback
}

// isProduction mirrors conf.Validate's environment check so auto_migrate can
// never run against a production database.
func isProduction(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}
