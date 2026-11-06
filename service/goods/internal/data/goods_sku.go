package data

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"log/slog"

	"goods/internal/biz"
	"goods/internal/domain"
)

const skuColumns = "id, goods_id, goods_sn, goods_name, sku_name, sku_code, bar_code, price, promotion_price, points, remarks_info, pic, on_sale, attr_info, inventory"

type goodsSkuRepo struct {
	data *Data
	log  *slog.Logger
}

// NewGoodsSkuRepoRepo .
func NewGoodsSkuRepoRepo(data *Data, logger *slog.Logger) biz.GoodsSkuRepo {
	return &goodsSkuRepo{
		data: data,
		log:  logger,
	}
}

// scanSku reads one goods_skus row into a domain SKU.
func scanSku(row interface {
	Scan(dest ...any) error
}) (*domain.GoodsSku, error) {
	var s domain.GoodsSku
	if err := row.Scan(&s.ID, &s.GoodsID, &s.GoodsSn, &s.GoodsName, &s.SkuName, &s.SkuCode,
		&s.BarCode, &s.Price, &s.PromotionPrice, &s.Points, &s.RemarksInfo, &s.Pic,
		&s.OnSale, &s.AttrInfo, &s.Inventory); err != nil {
		return nil, err
	}
	return &s, nil
}

func (g *goodsSkuRepo) Create(ctx context.Context, req *domain.GoodsSku) (*domain.GoodsSku, error) {
	res, err := g.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO goods_skus (goods_id, goods_sn, goods_name, sku_name, sku_code, bar_code, price, promotion_price, points, remarks_info, pic, on_sale, attr_info, inventory, add_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
		req.GoodsID, req.GoodsSn, req.GoodsName, req.SkuName, req.SkuCode, req.BarCode,
		req.Price, req.PromotionPrice, req.Points, req.RemarksInfo, req.Pic,
		req.OnSale, req.AttrInfo, req.Inventory,
	)
	if err != nil {
		return nil, errors.InternalServer("SKU_SAVE_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	req.ID = id
	return req, nil
}

func (g *goodsSkuRepo) CreateSkuRelation(ctx context.Context, req []*domain.GoodsSpecificationSku) error {
	if len(req) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(req))
	args := make([]any, 0, len(req)*4)
	for _, sku := range req {
		placeholders = append(placeholders, "(?, ?, ?, ?)")
		args = append(args, sku.SkuID, sku.SkuCode, sku.SpecificationId, sku.ValueId)
	}
	q := "INSERT INTO goods_specification_skus (sku_id, sku_code, specification_id, value_id) VALUES " +
		strings.Join(placeholders, ", ")
	if _, err := g.data.DB(ctx).ExecContext(ctx, q, args...); err != nil {
		return errors.InternalServer("SKU_RELATION_SAVE_ERROR", err.Error())
	}
	return nil
}

func (g *goodsSkuRepo) ListByIds(ctx context.Context, ids []int64) ([]*domain.GoodsSku, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	in, args := joinArgs(ids)
	rows, err := g.data.DB(ctx).QueryContext(ctx,
		"SELECT "+skuColumns+" FROM goods_skus WHERE id IN "+in+" AND deleted_at IS NULL", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := make([]*domain.GoodsSku, 0, len(ids))
	for rows.Next() {
		item, err := scanSku(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, rows.Err()
}

func (g *goodsSkuRepo) ListByGoodsID(ctx context.Context, goodsID int64) ([]*domain.GoodsSku, error) {
	rows, err := g.data.DB(ctx).QueryContext(ctx,
		"SELECT "+skuColumns+" FROM goods_skus WHERE goods_id = ? AND deleted_at IS NULL", goodsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.GoodsSku
	for rows.Next() {
		item, err := scanSku(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, rows.Err()
}
