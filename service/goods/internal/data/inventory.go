package data

import (
	"context"

	"github.com/go-kratos/kratos/v3/errors"
	"log/slog"

	"goods/internal/biz"
	"goods/internal/domain"
)

type inventoryRepo struct {
	data *Data
	log  *slog.Logger
}

// NewInventoryRepo .
func NewInventoryRepo(data *Data, logger *slog.Logger) biz.InventoryRepo {
	return &inventoryRepo{
		data: data,
		log:  logger,
	}
}

func (i inventoryRepo) Create(ctx context.Context, inventory *domain.Inventory) (*domain.Inventory, error) {
	res, err := i.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO goods_inventories (sku_id, inventory, add_time, update_time) VALUES (?, ?, NOW(), NOW())",
		inventory.SkuID, inventory.Inventory,
	)
	if err != nil {
		return nil, errors.InternalServer("INVENTORY_SAVE_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	inventory.ID = id
	return inventory, nil
}
