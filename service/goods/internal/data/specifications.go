package data

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"log/slog"

	"goods/internal/biz"
	"goods/internal/domain"
)

const specificationColumns = "id, type_id, name, sort, status, is_sku, is_select"

type specificationRepo struct {
	data *Data
	log  *slog.Logger
}

// NewSpecificationRepo .
func NewSpecificationRepo(data *Data, logger *slog.Logger) biz.SpecificationRepo {
	return &specificationRepo{
		data: data,
		log:  logger,
	}
}

func (g *specificationRepo) CreateSpecification(ctx context.Context, req *domain.Specification) (int64, error) {
	res, err := g.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO specifications_attrs (type_id, name, sort, status, is_sku, is_select, add_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())",
		req.TypeID, req.Name, req.Sort, req.Status, req.IsSKU, req.IsSelect,
	)
	if err != nil {
		return 0, errors.InternalServer("SPECIFICATION_SAVED_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (g *specificationRepo) CreateSpecificationValue(ctx context.Context, AttrId int64, req []*domain.SpecificationValue) error {
	placeholders := make([]string, 0, len(req))
	args := make([]any, 0, len(req)*3)
	for _, v := range req {
		placeholders = append(placeholders, "(?, ?, ?)")
		args = append(args, AttrId, v.Value, v.Sort)
	}
	if len(placeholders) == 0 {
		return nil
	}
	q := "INSERT INTO specifications_attr_values (attr_id, value, sort) VALUES " + strings.Join(placeholders, ", ")
	if _, err := g.data.DB(ctx).ExecContext(ctx, q, args...); err != nil {
		return errors.InternalServer("SPECIFICATION_VALUE_SAVED_ERROR", err.Error())
	}
	return nil
}

func (g *specificationRepo) ListByIds(ctx context.Context, id ...*int64) (domain.SpecificationList, error) {
	if len(id) == 0 {
		return nil, nil
	}
	raw := make([]int64, 0, len(id))
	for _, p := range id {
		raw = append(raw, *p)
	}
	in, args := joinArgs(raw)
	rows, err := g.data.DB(ctx).QueryContext(ctx,
		"SELECT "+specificationColumns+" FROM specifications_attrs WHERE id IN "+in+" AND deleted_at IS NULL", args...)
	if err != nil {
		return nil, errors.NotFound("SPECIFICATION_NOT_FOUND", "规格不存在")
	}
	defer rows.Close()

	var res domain.SpecificationList
	for rows.Next() {
		var s domain.Specification
		if err := rows.Scan(&s.ID, &s.TypeID, &s.Name, &s.Sort, &s.Status, &s.IsSKU, &s.IsSelect); err != nil {
			return nil, errors.NotFound("SPECIFICATION_NOT_FOUND", "规格不存在")
		}
		res = append(res, &s)
	}
	return res, rows.Err()
}
