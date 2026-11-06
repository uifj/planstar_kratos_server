package data

import (
	"context"
	"database/sql"
	serrors "errors"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"log/slog"

	"goods/internal/biz"
	"goods/internal/domain"
)

const (
	attrGroupColumns = "id, goods_type_id, title, `desc`, status, sort"
	attrColumns      = "id, goods_type_id, group_id, title, `desc`, status, sort"
)

type goodsAttrRepo struct {
	data *Data
	log  *slog.Logger
}

// NewGoodsAttrRepo .
func NewGoodsAttrRepo(data *Data, logger *slog.Logger) biz.GoodsAttrRepo {
	return &goodsAttrRepo{
		data: data,
		log:  logger,
	}
}

func (g *goodsAttrRepo) CreateGoodsGroupAttr(ctx context.Context, a *domain.AttrGroup) (*domain.AttrGroup, error) {
	res, err := g.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO goods_attr_groups (goods_type_id, title, `desc`, status, sort, add_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, NOW(), NOW())",
		a.TypeID, a.Title, a.Desc, a.Status, a.Sort,
	)
	if err != nil {
		return nil, errors.InternalServer("ATTR_GROUP_SAVE_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	a.ID = id
	return a, nil
}

func (g *goodsAttrRepo) IsExistsGroupByID(ctx context.Context, groupId int64) (*domain.AttrGroup, error) {
	var (
		group domain.AttrGroup
	)
	err := g.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+attrGroupColumns+" FROM goods_attr_groups WHERE id = ? AND deleted_at IS NULL", groupId,
	).Scan(&group.ID, &group.TypeID, &group.Title, &group.Desc, &group.Status, &group.Sort)
	if serrors.Is(err, sql.ErrNoRows) {
		return nil, errors.NotFound("ATTR_GROUP_NOT_FOUND", "商品属性组不存在")
	}
	if err != nil {
		return nil, errors.InternalServer("ATTR_GROUP_GET_ERROR", err.Error())
	}
	return &group, nil
}

func (g *goodsAttrRepo) CreateGoodsAttr(ctx context.Context, a *domain.GoodsAttr) (*domain.GoodsAttr, error) {
	res, err := g.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO goods_attrs (goods_type_id, group_id, title, `desc`, status, sort, add_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())",
		a.TypeID, a.GroupID, a.Title, a.Desc, a.Status, a.Sort,
	)
	if err != nil {
		return nil, errors.InternalServer("ATTR_SAVE_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	a.ID = id
	return a, nil
}

func (g *goodsAttrRepo) CreateGoodsAttrValue(ctx context.Context, r []*domain.GoodsAttrValue) ([]*domain.GoodsAttrValue, error) {
	placeholders := make([]string, 0, len(r))
	args := make([]any, 0, len(r)*3)
	for _, v := range r {
		placeholders = append(placeholders, "(?, ?, ?)")
		args = append(args, v.AttrId, v.GroupID, v.Value)
	}
	if len(placeholders) == 0 {
		return nil, nil
	}
	q := "INSERT INTO goods_attr_values (attr_id, group_id, value) VALUES " + strings.Join(placeholders, ", ")
	if _, err := g.data.DB(ctx).ExecContext(ctx, q, args...); err != nil {
		return nil, errors.InternalServer("ATTR_CREATE_ERROR", err.Error())
	}
	var out []*domain.GoodsAttrValue
	for _, v := range r {
		out = append(out, v)
	}
	return out, nil
}

func (g *goodsAttrRepo) GetAttrByIDs(ctx context.Context, ids []*int64) error {
	if len(ids) == 0 {
		return errors.NotFound("ATTR_NOT_FOUND", "商品属性不存在")
	}
	raw := make([]int64, 0, len(ids))
	for _, id := range ids {
		raw = append(raw, *id)
	}
	in, args := joinArgs(raw)
	var count int64
	if err := g.data.DB(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM goods_attrs WHERE id IN "+in+" AND deleted_at IS NULL", args...,
	).Scan(&count); err != nil {
		return errors.InternalServer("ATTR_GET_ERROR", err.Error())
	}
	if int64(len(raw)) != count {
		return errors.NotFound("ATTR_NOT_FOUND", "部分属性不存在")
	}
	return nil
}

func (g *goodsAttrRepo) ListByIds(ctx context.Context, ids ...int64) (domain.GoodsAttrList, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	in, args := joinArgs(ids)
	rows, err := g.data.DB(ctx).QueryContext(ctx,
		"SELECT "+attrColumns+" FROM goods_attrs WHERE id IN "+in+" AND deleted_at IS NULL", args...)
	if err != nil {
		return nil, errors.NotFound("ATTR_NOT_FOUND", "商品属性不存在")
	}
	defer rows.Close()

	var res domain.GoodsAttrList
	for rows.Next() {
		var attr domain.GoodsAttr
		if err := rows.Scan(&attr.ID, &attr.TypeID, &attr.GroupID, &attr.Title, &attr.Desc, &attr.Status, &attr.Sort); err != nil {
			return nil, errors.NotFound("ATTR_NOT_FOUND", "商品属性不存在")
		}
		res = append(res, &attr)
	}
	return res, rows.Err()
}
