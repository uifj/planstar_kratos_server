package data

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"inventory/internal/biz"
	"inventory/internal/domain"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/redis/go-redis/v9"
)

const stockKeyPrefix = "stock:sku:"

func stockKey(skuID int64) string {
	return stockKeyPrefix + strconv.FormatInt(skuID, 10)
}

// placeholders returns "?,?,?" repeated n times for an IN (…) clause.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func toArgs(ids []int64) []any {
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	return args
}

type inventoryRepo struct {
	data *Data
	log  *slog.Logger
}

func NewInventoryRepo(data *Data, logger *slog.Logger) biz.InventoryRepo {
	return &inventoryRepo{data: data, log: logger}
}

func (r *inventoryRepo) Query(ctx context.Context, skuIds []int64) ([]*domain.Inventory, error) {
	if len(skuIds) == 0 {
		return nil, nil
	}
	res := make([]*domain.Inventory, 0, len(skuIds))
	var missing []int64
	for _, id := range skuIds {
		inv, ok, err := r.queryCache(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok {
			res = append(res, inv)
			continue
		}
		missing = append(missing, id)
	}
	if len(missing) > 0 {
		rows, err := r.data.DB(ctx).QueryContext(ctx,
			"SELECT sku_id, inventory, locked FROM inventories WHERE sku_id IN ("+placeholders(len(missing))+")",
			toArgs(missing)...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var inv domain.Inventory
			if err := rows.Scan(&inv.SkuID, &inv.Inventory, &inv.Locked); err != nil {
				return nil, err
			}
			res = append(res, &inv)
			_ = r.cacheSet(ctx, &inv)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (r *inventoryRepo) TryLock(ctx context.Context, orderSn string, items []*domain.SkuItem) error {
	for _, item := range items {
		ok, err := r.tryLockRedis(ctx, item)
		if err != nil {
			return err
		}
		if !ok {
			return kerrors.New(400, "STOCK_NOT_ENOUGH", "库存不足")
		}
	}
	err := r.data.ExecTx(ctx, func(ctx context.Context) error {
		now := time.Now()
		for _, item := range items {
			var invID, invTotal, invLocked int64
			err := r.data.DB(ctx).QueryRowContext(ctx,
				"SELECT id, inventory, locked FROM inventories WHERE sku_id = ? FOR UPDATE", item.SkuID).
				Scan(&invID, &invTotal, &invLocked)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return kerrors.New(400, "STOCK_NOT_FOUND", "SKU 库存不存在")
				}
				return err
			}
			if invTotal-invLocked < int64(item.Num) {
				return kerrors.New(400, "STOCK_NOT_ENOUGH", "库存不足")
			}

			var existing int64
			err = r.data.DB(ctx).QueryRowContext(ctx,
				"SELECT id FROM inventory_locks WHERE order_sn = ? AND sku_id = ?", orderSn, item.SkuID).Scan(&existing)
			if err == nil {
				continue // 幂等：同一订单同一 SKU 已预占
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"UPDATE inventories SET locked = locked + ?, version = version + 1, updated_at = ? WHERE id = ?",
				item.Num, now, invID); err != nil {
				return err
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"INSERT INTO inventory_locks (order_sn, sku_id, num, status, created_at, updated_at) VALUES (?,?,?,?,?,?)",
				orderSn, item.SkuID, item.Num, 1, now, now); err != nil {
				return err
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"INSERT INTO inventory_flows (order_sn, sku_id, `change`, `type`, created_at) VALUES (?,?,?,?,?)",
				orderSn, item.SkuID, int64(item.Num), "lock", now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		for _, item := range items {
			r.rollbackRedisLock(ctx, item)
		}
		return err
	}
	for _, item := range items {
		_ = r.cacheInvalidate(ctx, item.SkuID)
	}
	return nil
}

func (r *inventoryRepo) ConfirmDeduct(ctx context.Context, orderSn string, items []*domain.SkuItem) error {
	err := r.data.ExecTx(ctx, func(ctx context.Context) error {
		now := time.Now()
		for _, item := range items {
			var lockID int64
			err := r.data.DB(ctx).QueryRowContext(ctx,
				"SELECT id FROM inventory_locks WHERE order_sn = ? AND sku_id = ? AND status = 1", orderSn, item.SkuID).
				Scan(&lockID)
			if err != nil {
				return kerrors.New(400, "LOCK_NOT_FOUND", "预占记录不存在")
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"UPDATE inventories SET inventory = inventory - ?, locked = locked - ?, version = version + 1, updated_at = ? WHERE sku_id = ?",
				item.Num, item.Num, now, item.SkuID); err != nil {
				return err
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"UPDATE inventory_locks SET status = 2 WHERE id = ?", lockID); err != nil {
				return err
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"INSERT INTO inventory_flows (order_sn, sku_id, `change`, `type`, created_at) VALUES (?,?,?,?,?)",
				orderSn, item.SkuID, -int64(item.Num), "deduct", now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, item := range items {
		_ = r.cacheInvalidate(ctx, item.SkuID)
	}
	return nil
}

func (r *inventoryRepo) Release(ctx context.Context, orderSn string, items []*domain.SkuItem) error {
	err := r.data.ExecTx(ctx, func(ctx context.Context) error {
		now := time.Now()
		for _, item := range items {
			var lockID int64
			err := r.data.DB(ctx).QueryRowContext(ctx,
				"SELECT id FROM inventory_locks WHERE order_sn = ? AND sku_id = ? AND status = 1", orderSn, item.SkuID).
				Scan(&lockID)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"UPDATE inventories SET locked = locked - ?, version = version + 1, updated_at = ? WHERE sku_id = ?",
				item.Num, now, item.SkuID); err != nil {
				return err
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"UPDATE inventory_locks SET status = 3 WHERE id = ?", lockID); err != nil {
				return err
			}
			if _, err := r.data.DB(ctx).ExecContext(ctx,
				"INSERT INTO inventory_flows (order_sn, sku_id, `change`, `type`, created_at) VALUES (?,?,?,?,?)",
				orderSn, item.SkuID, int64(item.Num), "release", now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, item := range items {
		_ = r.cacheInvalidate(ctx, item.SkuID)
	}
	return nil
}

func (r *inventoryRepo) IsConsumed(ctx context.Context, eventID string) (bool, error) {
	var one int
	err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT 1 FROM consumed_event WHERE event_id = ?", eventID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *inventoryRepo) MarkConsumed(ctx context.Context, eventID, orderSn string) error {
	_, err := r.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO consumed_event (event_id, order_sn, consumed_at) VALUES (?,?,?)",
		eventID, orderSn, time.Now())
	return err
}

func (r *inventoryRepo) queryCache(ctx context.Context, skuID int64) (*domain.Inventory, bool, error) {
	key := stockKey(skuID)
	values, err := r.data.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, false, err
	}
	if len(values) == 0 {
		return nil, false, nil
	}
	total, err1 := strconv.ParseInt(values["total"], 10, 64)
	locked, err2 := strconv.ParseInt(values["locked"], 10, 64)
	if err1 != nil || err2 != nil {
		return nil, false, nil
	}
	return &domain.Inventory{SkuID: skuID, Inventory: total, Locked: locked}, true, nil
}

func (r *inventoryRepo) cacheSet(ctx context.Context, inv *domain.Inventory) error {
	return r.data.rdb.HSet(ctx, stockKey(inv.SkuID), "total", inv.Inventory, "locked", inv.Locked).Err()
}

func (r *inventoryRepo) cacheInvalidate(ctx context.Context, skuID int64) error {
	return r.data.rdb.Del(ctx, stockKey(skuID)).Err()
}

func (r *inventoryRepo) tryLockRedis(ctx context.Context, item *domain.SkuItem) (bool, error) {
	script := redis.NewScript(`
local total = tonumber(redis.call('HGET', KEYS[1], 'total') or '-1')
local locked = tonumber(redis.call('HGET', KEYS[1], 'locked') or '0')
local num = tonumber(ARGV[1])
if total >= 0 and total - locked >= num then
  redis.call('HSET', KEYS[1], 'locked', locked + num)
  return 1
end
return 0
`)
	for attempt := 0; attempt < 2; attempt++ {
		res, err := script.Run(ctx, r.data.rdb, []string{stockKey(item.SkuID)}, item.Num).Int()
		if err != nil {
			return false, err
		}
		if res == 1 {
			return true, nil
		}
		var inv domain.Inventory
		if err := r.data.DB(ctx).QueryRowContext(ctx,
			"SELECT sku_id, inventory, locked FROM inventories WHERE sku_id = ?", item.SkuID).
			Scan(&inv.SkuID, &inv.Inventory, &inv.Locked); err != nil {
			return false, err
		}
		if err := r.cacheSet(ctx, &inv); err != nil {
			return false, err
		}
	}
	return false, nil
}

func (r *inventoryRepo) rollbackRedisLock(ctx context.Context, item *domain.SkuItem) {
	_ = r.data.rdb.HIncrBy(ctx, stockKey(item.SkuID), "locked", -int64(item.Num)).Err()
	_ = r.cacheInvalidate(ctx, item.SkuID)
}
