package data

import (
	"context"
	"database/sql"
	serrors "errors"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"log/slog"

	"goods/internal/biz"
	"goods/internal/domain"
)

// Goods 商品表
type Goods struct {
	ID                int64
	CategoryID        int32
	BrandsID          int32
	TypeID            int64
	Name              string
	NameAlias         string
	GoodsSn           string
	GoodsTags         string
	MarketPrice       int64
	GoodsBrief        string
	GoodsFrontImage   string
	GoodsImages       []string
	OnSale            bool
	ShipFree          bool
	ShipID            int32
	IsNew             bool
	IsHot             bool
	ClickNum          int64
	SoldNum           int64
	FavNum            int64
}

const goodsColumns = "id, category_id, brands_id, type_id, name, name_alias, goods_sn, goods_tags, market_price, goods_brief, goods_front_image, goods_images, on_sale, ship_free, ship_id, is_new, is_hot, click_num, sold_num, fav_num"

type goodsRepo struct {
	data *Data
	log  *slog.Logger
}

// NewGoodsRepo .
func NewGoodsRepo(data *Data, logger *slog.Logger) biz.GoodsRepo {
	return &goodsRepo{
		data: data,
		log:  logger,
	}
}

// scanGoods reads one goods row; the goods_images column is a JSON array text.
func scanGoods(row interface {
	Scan(dest ...any) error
}) (*domain.Goods, error) {
	var (
		g     domain.Goods
		images sql.NullString
	)
	if err := row.Scan(&g.ID, &g.CategoryID, &g.BrandsID, &g.TypeID, &g.Name, &g.NameAlias,
		&g.GoodsSn, &g.GoodsTags, &g.MarketPrice, &g.GoodsBrief, &g.GoodsFrontImage, &images,
		&g.OnSale, &g.ShipFree, &g.ShipID, &g.IsNew, &g.IsHot,
		&g.ClickNum, &g.SoldNum, &g.FavNum); err != nil {
		return nil, err
	}
	if images.Valid && images.String != "" {
		_ = json.Unmarshal([]byte(images.String), &g.GoodsImages)
	}
	return &g, nil
}

func (g goodsRepo) CreateGoods(c context.Context, goods *domain.Goods) (*domain.Goods, error) {
	images, _ := json.Marshal(goods.GoodsImages)
	res, err := g.data.DB(c).ExecContext(c,
		"INSERT INTO goods (category_id, brands_id, type_id, name, name_alias, goods_sn, goods_tags, market_price, goods_brief, goods_front_image, goods_images, on_sale, ship_free, ship_id, is_new, is_hot, add_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
		goods.CategoryID, goods.BrandsID, goods.TypeID, goods.Name, goods.NameAlias, goods.GoodsSn, goods.GoodsTags,
		goods.MarketPrice, goods.GoodsBrief, goods.GoodsFrontImage, string(images),
		goods.OnSale, goods.ShipFree, goods.ShipID, goods.IsNew, goods.IsHot,
	)
	if err != nil {
		return nil, errors.InternalServer("GOODS_CREATE_ERROR", "商品创建失败")
	}
	id, _ := res.LastInsertId()
	goods.ID = id
	return goods, nil
}

func (g goodsRepo) GoodsListByIDs(c context.Context, ids ...int64) ([]*domain.Goods, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	in, args := joinArgs(ids)
	rows, err := g.data.DB(c).QueryContext(c,
		"SELECT "+goodsColumns+" FROM goods WHERE id IN "+in+" AND deleted_at IS NULL", args...)
	if err != nil {
		return nil, errors.NotFound("GOODS_NOT_FOUND", "商品不存在")
	}
	defer rows.Close()
	var res []*domain.Goods
	for rows.Next() {
		item, err := scanGoods(rows)
		if err != nil {
			return nil, errors.NotFound("GOODS_NOT_FOUND", "商品不存在")
		}
		res = append(res, item)
	}
	return res, rows.Err()
}

func (g goodsRepo) Update(ctx context.Context, goods *domain.Goods) error {
	sets := []string{}
	args := []any{}
	if goods.Name != "" {
		sets = append(sets, "name = ?")
		args = append(args, goods.Name)
	}
	if goods.NameAlias != "" {
		sets = append(sets, "name_alias = ?")
		args = append(args, goods.NameAlias)
	}
	if goods.GoodsBrief != "" {
		sets = append(sets, "goods_brief = ?")
		args = append(args, goods.GoodsBrief)
	}
	if goods.GoodsFrontImage != "" {
		sets = append(sets, "goods_front_image = ?")
		args = append(args, goods.GoodsFrontImage)
	}
	if goods.MarketPrice > 0 {
		sets = append(sets, "market_price = ?")
		args = append(args, goods.MarketPrice)
	}
	sets = append(sets, "ship_free = ?", "is_new = ?", "is_hot = ?", "update_time = NOW()")
	args = append(args, goods.ShipFree, goods.IsNew, goods.IsHot, goods.ID)
	q := "UPDATE goods SET " + joinComma(sets) + " WHERE id = ?"
	_, err := g.data.DB(ctx).ExecContext(ctx, q, args...)
	return err
}

func (g goodsRepo) Delete(ctx context.Context, id int64) error {
	_, err := g.data.DB(ctx).ExecContext(ctx, "DELETE FROM goods WHERE id = ?", id)
	return err
}

func (g goodsRepo) UpdateStatus(ctx context.Context, id int64, onSale bool) error {
	_, err := g.data.DB(ctx).ExecContext(ctx,
		"UPDATE goods SET on_sale = ?, update_time = NOW() WHERE id = ?", onSale, id)
	return err
}

func (g goodsRepo) ListPage(ctx context.Context, keywords, skuCode string, categoryID, brandID int32, page, pageSize int) ([]*domain.Goods, int64, error) {
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if keywords != "" {
		like := "%" + keywords + "%"
		where = append(where, "(name LIKE ? OR name_alias LIKE ? OR goods_sn LIKE ?)")
		args = append(args, like, like, like)
	}
	if skuCode != "" {
		like := "%" + skuCode + "%"
		where = append(where, "id IN (SELECT goods_id FROM goods_skus WHERE sku_code LIKE ?)")
		args = append(args, like)
	}
	if categoryID > 0 {
		where = append(where, "category_id = ?")
		args = append(args, categoryID)
	}
	if brandID > 0 {
		where = append(where, "brands_id = ?")
		args = append(args, brandID)
	}
	cond := " WHERE " + joinComma(where)

	var total int64
	if err := g.data.DB(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM goods"+cond, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	rows, err := g.data.DB(ctx).QueryContext(ctx,
		"SELECT "+goodsColumns+" FROM goods"+cond+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	res := make([]*domain.Goods, 0, pageSize)
	for rows.Next() {
		item, err := scanGoods(rows)
		if err != nil {
			return nil, 0, err
		}
		res = append(res, item)
	}
	return res, total, rows.Err()
}

func (g goodsRepo) GetByID(ctx context.Context, id int64) (*domain.Goods, error) {
	row := g.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+goodsColumns+" FROM goods WHERE id = ? AND deleted_at IS NULL", id)
	goods, err := scanGoods(row)
	if serrors.Is(err, sql.ErrNoRows) {
		return nil, errors.NotFound("GOODS_NOT_FOUND", "商品不存在")
	}
	if err != nil {
		return nil, err
	}
	return goods, nil
}

func (g goodsRepo) ListAll(ctx context.Context) ([]*domain.Goods, error) {
	rows, err := g.data.DB(ctx).QueryContext(ctx,
		"SELECT "+goodsColumns+" FROM goods WHERE deleted_at IS NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.Goods
	for rows.Next() {
		item, err := scanGoods(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, rows.Err()
}

func (g goodsRepo) ListImagesByGoodsID(ctx context.Context, goodsID int64) ([]*domain.GoodsImage, error) {
	rows, err := g.data.DB(ctx).QueryContext(ctx,
		"SELECT id, goods_id, link, position, is_master FROM goods_images WHERE goods_id = ? AND deleted_at IS NULL ORDER BY position ASC", goodsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.GoodsImage
	for rows.Next() {
		var img domain.GoodsImage
		if err := rows.Scan(&img.ID, &img.GoodsID, &img.Link, &img.Position, &img.IsMaster); err != nil {
			return nil, err
		}
		res = append(res, &img)
	}
	return res, rows.Err()
}

func (g goodsRepo) IncrSoldNum(ctx context.Context, skuItems map[int64]int32) error {
	for skuID, num := range skuItems {
		var goodsID int64
		err := g.data.DB(ctx).QueryRowContext(ctx,
			"SELECT goods_id FROM goods_skus WHERE id = ? AND deleted_at IS NULL", skuID,
		).Scan(&goodsID)
		if serrors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := g.data.DB(ctx).ExecContext(ctx,
			"UPDATE goods SET sold_num = sold_num + ? WHERE id = ?", num, goodsID); err != nil {
			return err
		}
	}
	return nil
}

func (g goodsRepo) IsConsumed(ctx context.Context, eventID string) (bool, error) {
	var n int
	err := g.data.DB(ctx).QueryRowContext(ctx,
		"SELECT 1 FROM consumed_event WHERE event_id = ?", eventID,
	).Scan(&n)
	if serrors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g goodsRepo) MarkConsumed(ctx context.Context, eventID, orderSn string) error {
	_, err := g.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO consumed_event (event_id, order_sn, consumed_at) VALUES (?, ?, ?)",
		eventID, orderSn, time.Now())
	return err
}
