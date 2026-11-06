package data

import (
	"context"
	"database/sql"

	"github.com/go-kratos/kratos/v3/errors"

	"cart/internal/biz"
	"cart/internal/domain"
)

const cartColumns = "id, user_id, goods_id, sku_id, goods_price, goods_num, goods_sn, goods_name, is_select"

type cartRepo struct {
	data *Data
}

// NewCartRepo builds the shop_carts repository over the shared *Data.
func NewCartRepo(data *Data) biz.CartRepo {
	return &cartRepo{
		data: data,
	}
}

func (r *cartRepo) Create(ctx context.Context, c *domain.ShopCart) (*domain.ShopCart, error) {
	var (
		id     int64
		gnum   int32
	)
	err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT id, goods_num FROM shop_carts WHERE user_id = ? AND sku_id = ? AND deleted_at IS NULL",
		c.UserId, c.SkuId,
	).Scan(&id, &gnum)

	if err == sql.ErrNoRows {
		res, err := r.data.DB(ctx).ExecContext(ctx,
			"INSERT INTO shop_carts (user_id, goods_id, sku_id, goods_price, goods_num, goods_sn, goods_name, is_select, add_time, update_time) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
			c.UserId, c.GoodsId, c.SkuId, c.GoodsPrice, c.GoodsNum, c.GoodsSn, c.GoodsName, c.IsSelect,
		)
		if err != nil {
			return nil, errors.InternalServer("CREATE_CART_ERROR", "创建购物车失败")
		}
		newID, _ := res.LastInsertId()
		c.ID = newID
		return c, nil
	}
	if err != nil {
		return nil, errors.InternalServer("CREATE_CART_ERROR", "创建购物车失败")
	}

	// Existing line: accumulate the quantity (mirrors the original gorm merge).
	if _, err := r.data.DB(ctx).ExecContext(ctx,
		"UPDATE shop_carts SET goods_price = ?, goods_num = ?, goods_sn = ?, goods_name = ?, is_select = ?, update_time = NOW() WHERE id = ?",
		c.GoodsPrice, gnum+c.GoodsNum, c.GoodsSn, c.GoodsName, c.IsSelect, id,
	); err != nil {
		return nil, errors.InternalServer("CREATE_CART_ERROR", "创建购物车失败")
	}
	return r.Get(ctx, id, c.UserId)
}

func (r *cartRepo) List(ctx context.Context, userId int64) (domain.ShopCartList, error) {
	rows, err := r.data.DB(ctx).QueryContext(ctx,
		"SELECT "+cartColumns+" FROM shop_carts WHERE user_id = ? AND deleted_at IS NULL",
		userId,
	)
	if err != nil {
		return nil, errors.InternalServer("SELECT_CART_ERROR", "用户购物车列表查询失败")
	}
	defer rows.Close()

	var rsp domain.ShopCartList
	for rows.Next() {
		var c domain.ShopCart
		if err := rows.Scan(&c.ID, &c.UserId, &c.GoodsId, &c.SkuId, &c.GoodsPrice, &c.GoodsNum, &c.GoodsSn, &c.GoodsName, &c.IsSelect); err != nil {
			return nil, errors.InternalServer("SELECT_CART_ERROR", "用户购物车列表查询失败")
		}
		rsp = append(rsp, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.InternalServer("SELECT_CART_ERROR", "用户购物车列表查询失败")
	}
	return rsp, nil
}

func (r *cartRepo) Update(ctx context.Context, c *domain.ShopCart) error {
	var exists int64
	err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT id FROM shop_carts WHERE id = ? AND user_id = ? AND deleted_at IS NULL",
		c.ID, c.UserId,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return errors.NotFound("CART_NOT_FOUND", "购物车记录不存在")
	}
	if err != nil {
		return errors.InternalServer("UPDATE_CART_ERROR", "购物车更新失败")
	}
	if c.GoodsNum <= 0 {
		return errors.New(400, "CART_NUM_INVALID", "商品数量必须大于 0")
	}
	if _, err := r.data.DB(ctx).ExecContext(ctx,
		"UPDATE shop_carts SET goods_num = ?, update_time = NOW() WHERE id = ?",
		c.GoodsNum, c.ID,
	); err != nil {
		return errors.InternalServer("UPDATE_CART_ERROR", "购物车更新失败")
	}
	return nil
}

func (r *cartRepo) Delete(ctx context.Context, id, userId int64) error {
	res, err := r.data.DB(ctx).ExecContext(ctx,
		"UPDATE shop_carts SET deleted_at = NOW(), update_time = NOW() WHERE id = ? AND user_id = ? AND deleted_at IS NULL",
		id, userId,
	)
	if err != nil {
		return errors.InternalServer("DELETE_CART_ERROR", "购物车删除失败")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.NotFound("CART_NOT_FOUND", "购物车记录不存在")
	}
	return nil
}

func (r *cartRepo) Get(ctx context.Context, id, userId int64) (*domain.ShopCart, error) {
	var c domain.ShopCart
	err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+cartColumns+" FROM shop_carts WHERE id = ? AND user_id = ? AND deleted_at IS NULL",
		id, userId,
	).Scan(&c.ID, &c.UserId, &c.GoodsId, &c.SkuId, &c.GoodsPrice, &c.GoodsNum, &c.GoodsSn, &c.GoodsName, &c.IsSelect)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound("CART_NOT_FOUND", "购物车记录不存在")
	}
	if err != nil {
		return nil, errors.InternalServer("SELECT_CART_ERROR", "购物车查询失败")
	}
	return &c, nil
}
