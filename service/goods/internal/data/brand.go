package data

import (
	"context"
	"database/sql"

	"github.com/go-kratos/kratos/v3/errors"

	"goods/internal/biz"
	"goods/internal/domain"
)

const brandColumns = "id, name, logo, `desc`, is_tab, sort"

type BrandRepo struct {
	data *Data
}

// NewBrandRepo builds the brands repository over the shared *Data.
func NewBrandRepo(data *Data) biz.BrandRepo {
	return &BrandRepo{data: data}
}

func (r *BrandRepo) Create(ctx context.Context, b *domain.Brand) (*domain.Brand, error) {
	res, err := r.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO brands (name, logo, `desc`, is_tab, sort, add_time, update_time) VALUES (?, ?, ?, ?, ?, NOW(), NOW())",
		b.Name, b.Logo, b.Desc, b.IsTab, b.Sort,
	)
	if err != nil {
		return nil, errors.InternalServer("SAVE_BRAND_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	b.ID = int32(id)
	return b, nil
}

func (r *BrandRepo) GetBradByName(ctx context.Context, name string) (*domain.Brand, error) {
	var (
		id   int64
		brand domain.Brand
	)
	err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+brandColumns+" FROM brands WHERE name = ? AND deleted_at IS NULL LIMIT 1", name,
	).Scan(&id, &brand.Name, &brand.Logo, &brand.Desc, &brand.IsTab, &brand.Sort)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound("BRAND_NOT_FOUND", "brand not found")
	}
	if err != nil {
		return nil, errors.InternalServer("BRAND_GET_ERROR", err.Error())
	}
	brand.ID = int32(id)
	return &brand, nil
}

func (r *BrandRepo) Update(ctx context.Context, b *domain.Brand) error {
	var exists int64
	if err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT id FROM brands WHERE id = ? AND deleted_at IS NULL", b.ID,
	).Scan(&exists); err == sql.ErrNoRows {
		return errors.NotFound("BRAND_NOT_FOUND", "brand not found")
	} else if err != nil {
		return errors.InternalServer("UPDATE_BRAND_ERROR", err.Error())
	}

	sets := []string{}
	args := []any{}
	if b.Name != "" {
		sets = append(sets, "name = ?")
		args = append(args, b.Name)
	}
	if b.Logo != "" {
		sets = append(sets, "logo = ?")
		args = append(args, b.Logo)
	}
	if b.IsTab {
		sets = append(sets, "is_tab = ?")
		args = append(args, b.IsTab)
	}
	if b.Sort != 0 {
		sets = append(sets, "sort = ?")
		args = append(args, b.Sort)
	}
	if b.Desc != "" {
		sets = append(sets, "`desc` = ?")
		args = append(args, b.Desc)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "update_time = NOW()")
	args = append(args, b.ID)
	q := "UPDATE brands SET " + joinComma(sets) + " WHERE id = ?"
	if _, err := r.data.DB(ctx).ExecContext(ctx, q, args...); err != nil {
		return errors.InternalServer("UPDATE_BRAND_ERROR", err.Error())
	}
	return nil
}

func (r *BrandRepo) List(ctx context.Context, p *biz.Pagination) ([]*domain.Brand, int64, error) {
	if p == nil || p.PageNum == 0 {
		if p == nil {
			p = &biz.Pagination{}
		}
		p.PageNum = 1
	}
	pageSize := p.PageSize
	switch {
	case pageSize > 100:
		pageSize = 100
	case pageSize <= 0:
		pageSize = 10
	}
	offset := (p.PageNum - 1) * pageSize

	var total int64
	if err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM brands WHERE deleted_at IS NULL",
	).Scan(&total); err != nil {
		return nil, 0, errors.InternalServer("BRAND_COUNT_ERROR", err.Error())
	}

	rows, err := r.data.DB(ctx).QueryContext(ctx,
		"SELECT "+brandColumns+" FROM brands WHERE deleted_at IS NULL ORDER BY id ASC LIMIT ? OFFSET ?",
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, errors.InternalServer("BRAND_LIST_ERROR", err.Error())
	}
	defer rows.Close()

	var rsp []*domain.Brand
	for rows.Next() {
		var (
			id    int64
			brand domain.Brand
		)
		if err := rows.Scan(&id, &brand.Name, &brand.Logo, &brand.Desc, &brand.IsTab, &brand.Sort); err != nil {
			return nil, 0, errors.InternalServer("BRAND_LIST_ERROR", err.Error())
		}
		brand.ID = int32(id)
		rsp = append(rsp, &brand)
	}
	return rsp, total, rows.Err()
}

func (r *BrandRepo) IsBrand(ctx context.Context, ids []int32) error {
	if len(ids) == 0 {
		return errors.InternalServer("BRAND_NOT_FOUND", "brand not found")
	}
	in, args := joinArgs(ids)
	var count int64
	if err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM brands WHERE id IN "+in+" AND deleted_at IS NULL", args...,
	).Scan(&count); err != nil {
		return errors.InternalServer("BRAND_NOT_FOUND", err.Error())
	}
	if int64(len(ids)) != count {
		return errors.InternalServer("BRAND_NOT_FOUND", "品牌不存在")
	}
	return nil
}

func (r *BrandRepo) ListByIds(ctx context.Context, ids ...int32) (domain.BrandList, error) {
	if len(ids) == 0 {
		return nil, errors.InternalServer("BRAND_NOT_FOUND", "请选择品牌")
	}
	in, args := joinArgs(ids)
	rows, err := r.data.DB(ctx).QueryContext(ctx,
		"SELECT "+brandColumns+" FROM brands WHERE id IN "+in+" AND deleted_at IS NULL", args...,
	)
	if err != nil {
		return nil, errors.InternalServer("BRAND_NOT_FOUND", err.Error())
	}
	defer rows.Close()

	var res domain.BrandList
	for rows.Next() {
		var (
			id    int64
			brand domain.Brand
		)
		if err := rows.Scan(&id, &brand.Name, &brand.Logo, &brand.Desc, &brand.IsTab, &brand.Sort); err != nil {
			return nil, errors.InternalServer("BRAND_NOT_FOUND", err.Error())
		}
		brand.ID = int32(id)
		res = append(res, &brand)
	}
	return res, rows.Err()
}

func (r *BrandRepo) IsBrandByID(ctx context.Context, id int32) (*domain.Brand, error) {
	var (
		vid   int64
		brand domain.Brand
	)
	err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+brandColumns+" FROM brands WHERE id = ? AND deleted_at IS NULL", id,
	).Scan(&vid, &brand.Name, &brand.Logo, &brand.Desc, &brand.IsTab, &brand.Sort)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound("BRAND_NOT_FOUND", "brand not found")
	}
	if err != nil {
		return nil, errors.InternalServer("BRAND_NOT_FOUND", err.Error())
	}
	brand.ID = int32(vid)
	return &brand, nil
}

// joinComma joins SET fragments with ", ".
func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
