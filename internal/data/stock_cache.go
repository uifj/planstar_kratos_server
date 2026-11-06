package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/redis/go-redis/v9"
)

const stockCacheTTL = 5 * time.Minute

func stockKey(skuID string) string { return "mall:stock:" + skuID }

type stockCache struct{ rdb redis.UniversalClient }

func NewStockCache(d *Data) biz.StockCache { return &stockCache{rdb: d.rdb} }

func (c *stockCache) Get(ctx context.Context, skuID string) (int64, bool, error) {
	n, err := c.rdb.Get(ctx, stockKey(skuID)).Int64()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return n, true, nil
}

func (c *stockCache) Set(ctx context.Context, skuID string, n int64) error {
	return c.rdb.Set(ctx, stockKey(skuID), n, stockCacheTTL).Err()
}

func (c *stockCache) Invalidate(ctx context.Context, skuID string) error {
	return c.rdb.Del(ctx, stockKey(skuID)).Err()
}
