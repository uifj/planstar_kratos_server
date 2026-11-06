package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent/inventory"
	"github.com/go-kratos/kratos-layout/internal/data/ent/inventorylock"
)

type inventoryRepo struct{ data *Data }

func NewInventoryRepo(d *Data) biz.InventoryRepo { return &inventoryRepo{data: d} }

func (r *inventoryRepo) EnsureSku(ctx context.Context, skuID string, total int64) error {
	po, err := r.data.db.Inventory.Query().Where(inventory.SkuIDEQ(skuID)).Only(ctx)
	if isNotFound(err) {
		_, err = r.data.db.Inventory.Create().SetSkuID(skuID).SetTotal(total).Save(ctx)
		return err
	}
	if err != nil {
		return err
	}
	_, err = r.data.db.Inventory.UpdateOneID(po.ID).SetTotal(total).Save(ctx)
	return err
}

func (r *inventoryRepo) Available(ctx context.Context, skuID string) (int64, error) {
	po, err := r.data.db.Inventory.Query().Where(inventory.SkuIDEQ(skuID)).Only(ctx)
	if err != nil {
		return 0, err
	}
	return po.Total - po.Locked - po.Sold, nil
}

// TryLock takes a FOR UPDATE row lock on each inventory row and increments
// locked with a version CAS. All lines live in one transaction, so a partial
// lock can never be committed.
func (r *inventoryRepo) TryLock(ctx context.Context, orderSn string, items []biz.StockItem) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	for _, it := range items {
		inv, err := tx.Inventory.Query().Where(inventory.SkuIDEQ(it.SkuID)).ForUpdate().Only(ctx)
		if err != nil {
			_ = tx.Rollback()
			return biz.ErrInventoryNotEnough
		}
		if inv.Total-inv.Locked-inv.Sold < int64(it.Num) {
			_ = tx.Rollback()
			return biz.ErrInventoryNotEnough
		}
		n, err := tx.Inventory.Update().
			Where(inventory.IDEQ(inv.ID), inventory.VersionEQ(inv.Version)).
			AddLocked(int64(it.Num)).AddVersion(1).Save(ctx)
		if err != nil || n == 0 {
			_ = tx.Rollback()
			return biz.ErrInventoryNotEnough
		}
		if _, err = tx.InventoryLock.Create().
			SetOrderSn(orderSn).SetSkuID(it.SkuID).SetNum(it.Num).
			SetStatus(biz.LockHeld).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = tx.InventoryFlow.Create().
			SetSkuID(it.SkuID).SetOrderSn(orderSn).SetNum(it.Num).
			SetFlowType(biz.FlowLock).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *inventoryRepo) ConfirmDeduct(ctx context.Context, orderSn string, items []biz.StockItem) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	for _, it := range items {
		lock, err := tx.InventoryLock.Query().
			Where(inventorylock.OrderSnEQ(orderSn), inventorylock.SkuIDEQ(it.SkuID), inventorylock.StatusEQ(biz.LockHeld)).Only(ctx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		inv, err := tx.Inventory.Query().Where(inventory.SkuIDEQ(it.SkuID)).ForUpdate().Only(ctx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		n, err := tx.Inventory.Update().
			Where(inventory.IDEQ(inv.ID), inventory.VersionEQ(inv.Version)).
			AddLocked(-int64(it.Num)).AddSold(int64(it.Num)).AddVersion(1).Save(ctx)
		if err != nil || n == 0 {
			_ = tx.Rollback()
			return biz.ErrInventoryNotEnough
		}
		if _, err = tx.InventoryLock.UpdateOneID(lock.ID).SetStatus(biz.LockConfirmed).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = tx.InventoryFlow.Create().
			SetSkuID(it.SkuID).SetOrderSn(orderSn).SetNum(it.Num).
			SetFlowType(biz.FlowConfirm).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *inventoryRepo) Release(ctx context.Context, orderSn string) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	locks, err := tx.InventoryLock.Query().
		Where(inventorylock.OrderSnEQ(orderSn), inventorylock.StatusEQ(biz.LockHeld)).All(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, l := range locks {
		inv, err := tx.Inventory.Query().Where(inventory.SkuIDEQ(l.SkuID)).ForUpdate().Only(ctx)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = tx.Inventory.Update().
			Where(inventory.IDEQ(inv.ID), inventory.VersionEQ(inv.Version)).
			AddLocked(-int64(l.Num)).AddVersion(1).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = tx.InventoryLock.UpdateOneID(l.ID).SetStatus(biz.LockReleased).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err = tx.InventoryFlow.Create().
			SetSkuID(l.SkuID).SetOrderSn(orderSn).SetNum(l.Num).
			SetFlowType(biz.FlowRelease).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
