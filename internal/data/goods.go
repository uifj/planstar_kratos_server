package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent"
	"github.com/go-kratos/kratos-layout/internal/data/ent/brand"
	"github.com/go-kratos/kratos-layout/internal/data/ent/category"
	"github.com/go-kratos/kratos-layout/internal/data/ent/goods"
	"github.com/go-kratos/kratos-layout/internal/data/ent/goodsimage"
	"github.com/go-kratos/kratos-layout/internal/data/ent/goodssku"
	"github.com/go-kratos/kratos-layout/internal/data/ent/predicate"
)

func goodsToBiz(po *ent.Goods) *biz.Goods {
	if po == nil {
		return nil
	}
	return &biz.Goods{
		ID: idStr(po.ID), CategoryID: po.CategoryID, BrandID: po.BrandID,
		TypeID: po.TypeID, Name: po.Name, NameAlias: po.NameAlias,
		GoodsSn: po.GoodsSn, GoodsBrief: po.GoodsBrief,
		GoodsFrontImage: po.GoodsFrontImage, MarketPrice: po.MarketPrice,
		OnSale: po.OnSale, ShipFree: po.ShipFree, IsNew: po.IsNew,
		IsHot: po.IsHot, SoldNum: po.SoldNum, Status: po.Status,
	}
}

func skuToBiz(po *ent.GoodsSku) *biz.GoodsSku {
	if po == nil {
		return nil
	}
	return &biz.GoodsSku{
		ID: idStr(po.ID), GoodsID: po.GoodsID, SkuName: po.SkuName,
		SkuCode: po.SkuCode, BarCode: po.BarCode, Price: po.Price,
		PromotionPrice: po.PromotionPrice, Inventory: po.Inventory,
		OnSale: po.OnSale, Pic: po.Pic, Specs: po.Specs, Status: po.Status,
	}
}

func imageToBiz(po *ent.GoodsImage) *biz.GoodsImage {
	if po == nil {
		return nil
	}
	return &biz.GoodsImage{
		ID: idStr(po.ID), GoodsID: po.GoodsID, URL: po.URL,
		Position: po.Position, IsMaster: po.IsMaster, Status: po.Status,
	}
}

type goodsRepo struct{ data *Data }

func NewGoodsRepo(d *Data) biz.GoodsRepo { return &goodsRepo{data: d} }

func (r *goodsRepo) hydrate(ctx context.Context, po *ent.Goods) (*biz.Goods, error) {
	g := goodsToBiz(po)
	skus, err := r.data.db.GoodsSku.Query().Where(goodssku.GoodsIDEQ(po.ID.String()), goodssku.StatusEQ(biz.RowActive)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range skus {
		g.Skus = append(g.Skus, skuToBiz(s))
	}
	imgs, err := r.data.db.GoodsImage.Query().Where(goodsimage.GoodsIDEQ(po.ID.String()), goodsimage.StatusEQ(biz.RowActive)).Order(goodsimage.ByPosition()).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, i := range imgs {
		g.Images = append(g.Images, imageToBiz(i))
	}
	g.CategoryName = r.CategoryName(ctx, po.CategoryID)
	g.BrandName = r.BrandName(ctx, po.BrandID)
	return g, nil
}

func (r *goodsRepo) Create(ctx context.Context, g *biz.Goods) (*biz.Goods, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	po, err := tx.Goods.Create().
		SetID(parseID(g.ID)).SetCategoryID(g.CategoryID).SetBrandID(g.BrandID).
		SetTypeID(g.TypeID).SetName(g.Name).SetNameAlias(g.NameAlias).
		SetGoodsSn(g.GoodsSn).SetGoodsBrief(g.GoodsBrief).
		SetGoodsFrontImage(g.GoodsFrontImage).SetMarketPrice(g.MarketPrice).
		SetOnSale(g.OnSale).SetShipFree(g.ShipFree).SetIsNew(g.IsNew).
		SetIsHot(g.IsHot).SetSoldNum(g.SoldNum).SetStatus(g.Status).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	for _, s := range g.Skus {
		c := tx.GoodsSku.Create().
			SetID(parseID(s.ID)).SetGoodsID(po.ID.String()).
			SetSkuName(s.SkuName).SetSkuCode(s.SkuCode).SetBarCode(s.BarCode).
			SetPrice(s.Price).SetPromotionPrice(s.PromotionPrice).
			SetInventory(s.Inventory).SetOnSale(s.OnSale).SetPic(s.Pic).
			SetStatus(s.Status)
		if s.Specs != nil {
			c.SetSpecs(s.Specs)
		}
		if _, err := c.Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	for _, im := range g.Images {
		if _, err := tx.GoodsImage.Create().
			SetID(parseID(im.ID)).SetGoodsID(po.ID.String()).
			SetURL(im.URL).SetPosition(im.Position).SetIsMaster(im.IsMaster).
			SetStatus(im.Status).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, idStr(po.ID))
}

func (r *goodsRepo) FindByID(ctx context.Context, id string) (*biz.Goods, error) {
	po, err := r.data.db.Goods.Query().
		Where(goods.IDEQ(parseID(id)), goods.StatusEQ(biz.RowActive)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrGoodsNotFound
		}
		return nil, err
	}
	return r.hydrate(ctx, po)
}

func (r *goodsRepo) FindSkuByID(ctx context.Context, id string) (*biz.GoodsSku, error) {
	po, err := r.data.db.GoodsSku.Query().
		Where(goodssku.IDEQ(parseID(id)), goodssku.StatusEQ(biz.RowActive)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrGoodsSkuNotFound
		}
		return nil, err
	}
	return skuToBiz(po), nil
}

func (r *goodsRepo) List(ctx context.Context, q biz.GoodsQuery) ([]*biz.Goods, int32, error) {
	var ps []predicate.Goods
	ps = append(ps, goods.StatusEQ(biz.RowActive))
	if len(q.IDs) > 0 {
		ids := make([]string, 0, len(q.IDs))
		for _, s := range q.IDs {
			if s != "" {
				ids = append(ids, s)
			}
		}
		if len(ids) == 0 {
			return []*biz.Goods{}, 0, nil
		}
		// Preserve ES relevance order in the caller by hydrating one by one.
		out := make([]*biz.Goods, 0, len(ids))
		for _, id := range ids {
			g, err := r.FindByID(ctx, id)
			if err == nil {
				out = append(out, g)
			}
		}
		return out, int32(len(out)), nil
	}
	if q.OnlyOnSale {
		ps = append(ps, goods.OnSaleEQ(true))
	}
	if q.CategoryID != "" {
		ps = append(ps, goods.CategoryIDEQ(q.CategoryID))
	}
	if q.BrandID != "" {
		ps = append(ps, goods.BrandIDEQ(q.BrandID))
	}
	query := r.data.db.Goods.Query().Where(ps...)
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := r.data.db.Goods.Query().Where(ps...).
		Order(goods.ByID()).
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.Goods, 0, len(pos))
	for _, po := range pos {
		g, err := r.hydrate(ctx, po)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, g)
	}
	return out, int32(total), nil
}

func (r *goodsRepo) AllForIndex(ctx context.Context) ([]*biz.Goods, error) {
	pos, err := r.data.db.Goods.Query().Where(goods.StatusEQ(biz.RowActive)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.Goods, 0, len(pos))
	for _, po := range pos {
		g, err := r.hydrate(ctx, po)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, nil
}

func (r *goodsRepo) Update(ctx context.Context, g *biz.Goods) error {
	_, err := r.data.db.Goods.UpdateOneID(parseID(g.ID)).
		Where(goods.StatusEQ(biz.RowActive)).
		SetName(g.Name).SetNameAlias(g.NameAlias).SetGoodsBrief(g.GoodsBrief).
		SetGoodsFrontImage(g.GoodsFrontImage).SetMarketPrice(g.MarketPrice).
		SetShipFree(g.ShipFree).SetIsNew(g.IsNew).SetIsHot(g.IsHot).Save(ctx)
	return err
}

func (r *goodsRepo) SetOnSale(ctx context.Context, id string, onSale bool) error {
	_, err := r.data.db.Goods.UpdateOneID(parseID(id)).
		Where(goods.StatusEQ(biz.RowActive)).SetOnSale(onSale).Save(ctx)
	return err
}

func (r *goodsRepo) Delete(ctx context.Context, id string) error {
	_, err := r.data.db.Goods.UpdateOneID(parseID(id)).
		Where(goods.StatusEQ(biz.RowActive)).SetStatus(biz.RowDeleted).Save(ctx)
	return err
}

func (r *goodsRepo) IncrSoldNum(ctx context.Context, skuID string, delta int64) error {
	sku, err := r.data.db.GoodsSku.Query().Where(goodssku.IDEQ(parseID(skuID))).Only(ctx)
	if err != nil {
		return err
	}
	_, err = r.data.db.Goods.Update().
		Where(goods.IDEQ(parseID(sku.GoodsID))).AddSoldNum(delta).Save(ctx)
	return err
}

func (r *goodsRepo) Categories(ctx context.Context) ([]*biz.Category, error) {
	pos, err := r.data.db.Category.Query().Where(category.StatusEQ(biz.RowActive)).Order(category.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.Category, 0, len(pos))
	for _, po := range pos {
		out = append(out, &biz.Category{
			ID: idStr(po.ID), Name: po.Name, ParentCategory: po.ParentCategory,
			Level: po.Level, Sort: po.Sort, Status: po.Status,
		})
	}
	return out, nil
}

func (r *goodsRepo) Brands(ctx context.Context, page, pageSize int) ([]*biz.Brand, error) {
	pos, err := r.data.db.Brand.Query().Where(brand.StatusEQ(biz.RowActive)).
		Order(brand.ByID()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.Brand, 0, len(pos))
	for _, po := range pos {
		out = append(out, &biz.Brand{ID: idStr(po.ID), Name: po.Name, Logo: po.Logo, Status: po.Status})
	}
	return out, nil
}

func (r *goodsRepo) CreateCategory(ctx context.Context, c *biz.Category) (*biz.Category, error) {
	po, err := r.data.db.Category.Create().
		SetID(parseID(c.ID)).SetName(c.Name).SetParentCategory(c.ParentCategory).
		SetLevel(c.Level).SetSort(c.Sort).SetStatus(c.Status).Save(ctx)
	if err != nil {
		return nil, err
	}
	c.ID = idStr(po.ID)
	return c, nil
}

func (r *goodsRepo) UpdateCategory(ctx context.Context, c *biz.Category) error {
	_, err := r.data.db.Category.UpdateOneID(parseID(c.ID)).
		Where(category.StatusEQ(biz.RowActive)).
		SetName(c.Name).SetParentCategory(c.ParentCategory).
		SetLevel(c.Level).SetSort(c.Sort).Save(ctx)
	return err
}

func (r *goodsRepo) DeleteCategory(ctx context.Context, id string) error {
	_, err := r.data.db.Category.UpdateOneID(parseID(id)).
		Where(category.StatusEQ(biz.RowActive)).SetStatus(biz.RowDeleted).Save(ctx)
	return err
}

func (r *goodsRepo) CategoryName(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	po, err := r.data.db.Category.Query().Where(category.IDEQ(parseID(id))).Only(ctx)
	if err != nil {
		return ""
	}
	return po.Name
}

func (r *goodsRepo) BrandName(ctx context.Context, id string) string {
	if id == "" {
		return ""
	}
	po, err := r.data.db.Brand.Query().Where(brand.IDEQ(parseID(id))).Only(ctx)
	if err != nil {
		return ""
	}
	return po.Name
}
