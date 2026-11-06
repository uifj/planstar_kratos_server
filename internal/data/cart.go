package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent"
	"github.com/go-kratos/kratos-layout/internal/data/ent/shopcart"
)

func cartToBiz(po *ent.ShopCart) *biz.CartItem {
	if po == nil {
		return nil
	}
	return &biz.CartItem{
		ID: idStr(po.ID), UserID: po.UserID, GoodsID: po.GoodsID,
		GoodsSn: po.GoodsSn, GoodsName: po.GoodsName, SkuID: po.SkuID,
		GoodsPrice: po.GoodsPrice, GoodsNum: po.GoodsNum,
		IsSelect: po.IsSelect, Status: po.Status,
	}
}

type cartRepo struct{ data *Data }

func NewCartRepo(d *Data) biz.CartRepo { return &cartRepo{data: d} }

func (r *cartRepo) Add(ctx context.Context, c *biz.CartItem) (*biz.CartItem, error) {
	po, err := r.data.db.ShopCart.Create().
		SetID(parseID(c.ID)).SetUserID(c.UserID).SetGoodsID(c.GoodsID).
		SetGoodsSn(c.GoodsSn).SetGoodsName(c.GoodsName).SetSkuID(c.SkuID).
		SetGoodsPrice(c.GoodsPrice).SetGoodsNum(c.GoodsNum).
		SetIsSelect(c.IsSelect).SetStatus(c.Status).Save(ctx)
	if err != nil {
		return nil, err
	}
	return cartToBiz(po), nil
}

func (r *cartRepo) ListByUser(ctx context.Context, userID string) ([]*biz.CartItem, error) {
	pos, err := r.data.db.ShopCart.Query().
		Where(shopcart.UserIDEQ(userID), shopcart.StatusEQ(biz.RowActive)).
		Order(shopcart.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.CartItem, 0, len(pos))
	for _, po := range pos {
		out = append(out, cartToBiz(po))
	}
	return out, nil
}

func (r *cartRepo) FindByID(ctx context.Context, id string) (*biz.CartItem, error) {
	po, err := r.data.db.ShopCart.Query().
		Where(shopcart.IDEQ(parseID(id)), shopcart.StatusEQ(biz.RowActive)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrCartNotFound
		}
		return nil, err
	}
	return cartToBiz(po), nil
}

func (r *cartRepo) UpdateNum(ctx context.Context, userID, id string, num int32, selected *bool) error {
	u := r.data.db.ShopCart.UpdateOneID(parseID(id)).
		Where(shopcart.UserIDEQ(userID), shopcart.StatusEQ(biz.RowActive)).
		SetGoodsNum(num)
	if selected != nil {
		u.SetIsSelect(*selected)
	}
	_, err := u.Save(ctx)
	if err != nil && isNotFound(err) {
		return biz.ErrCartNotFound
	}
	return err
}

func (r *cartRepo) Delete(ctx context.Context, userID, id string) error {
	_, err := r.data.db.ShopCart.UpdateOneID(parseID(id)).
		Where(shopcart.UserIDEQ(userID), shopcart.StatusEQ(biz.RowActive)).
		SetStatus(biz.RowDeleted).Save(ctx)
	return err
}

func (r *cartRepo) DeleteMany(ctx context.Context, userID string, ids []string) error {
	for _, id := range ids {
		if err := r.Delete(ctx, userID, id); err != nil {
			return err
		}
	}
	return nil
}
