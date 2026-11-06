package data

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-kratos/kratos/v3/errors"

	"goods/internal/biz"
)

const categoryColumns = "id, name, parent_category_id, level, is_tab, sort"

type CategoryRepo struct {
	data *Data
}

// NewCategoryRepo builds the categories repository over the shared *Data.
func NewCategoryRepo(data *Data) biz.CategoryRepo {
	return &CategoryRepo{data: data}
}

type categoryPO struct {
	ID       int32
	Name     string
	ParentID int32
	Level    int32
	IsTab    bool
	Sort     int32
}

func (r *CategoryRepo) DeleteCategory(ctx context.Context, id int32) error {
	res, err := r.data.DB(ctx).ExecContext(ctx,
		"UPDATE categories SET deleted_at = NOW(), update_time = NOW() WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return errors.InternalServer("DELETE_CATGORY_ERROR", err.Error())
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.InternalServer("DELETE_CATGORY_ERROR", "商品分类不存在")
	}
	return nil
}

func (r *CategoryRepo) UpdateCategory(ctx context.Context, req *biz.CategoryInfo) error {
	var exists int64
	if err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT id FROM categories WHERE id = ? AND deleted_at IS NULL", req.ID,
	).Scan(&exists); err == sql.ErrNoRows {
		return errors.NotFound("CATEGORY_NOT_FOUND", "商品分类不存在")
	} else if err != nil {
		return errors.InternalServer("CATEGORY_UPDATE_ERROR", err.Error())
	}

	sets := []string{}
	args := []any{}
	if req.Name != "" {
		sets = append(sets, "name = ?")
		args = append(args, req.Name)
	}
	if req.ParentCategory != 0 {
		sets = append(sets, "parent_category_id = ?")
		args = append(args, req.ParentCategory)
	}
	if req.Level != 0 {
		sets = append(sets, "level = ?")
		args = append(args, req.Level)
	}
	if req.IsTab {
		sets = append(sets, "is_tab = ?")
		args = append(args, req.IsTab)
	}
	if len(sets) == 0 {
		return nil
	}
	sets = append(sets, "update_time = NOW()")
	args = append(args, req.ID)
	q := "UPDATE categories SET " + joinComma(sets) + " WHERE id = ?"
	if _, err := r.data.DB(ctx).ExecContext(ctx, q, args...); err != nil {
		return errors.InternalServer("CATEGORY_UPDATE_ERROR", "商品分类创建失败")
	}
	return nil
}

func (r *CategoryRepo) AddCategory(ctx context.Context, req *biz.CategoryInfo) (*biz.CategoryInfo, error) {
	parentID := req.ParentCategory
	if req.Level != 1 {
		var exists int64
		if err := r.data.DB(ctx).QueryRowContext(ctx,
			"SELECT id FROM categories WHERE id = ? AND deleted_at IS NULL", req.ParentCategory,
		).Scan(&exists); err == sql.ErrNoRows {
			return nil, errors.NotFound("CATEGORY_NOT_FOUND", "商品分类不存在")
		} else if err != nil {
			return nil, errors.InternalServer("CATEGORY_CREATE_ERROR", err.Error())
		}
	}

	res, err := r.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO categories (name, parent_category_id, level, is_tab, sort, add_time, update_time) VALUES (?, ?, ?, ?, ?, NOW(), NOW())",
		req.Name, parentID, req.Level, req.IsTab, req.Sort,
	)
	if err != nil {
		return nil, errors.InternalServer("CATEGORY_CREATE_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	return &biz.CategoryInfo{
		ID:             int32(id),
		Name:           req.Name,
		ParentCategory: parentID,
		Level:          req.Level,
		IsTab:          req.IsTab,
		Sort:           req.Sort,
	}, nil
}

// loadAllCategories reads every live category row once.
func (r *CategoryRepo) loadAllCategories(ctx context.Context) ([]*categoryPO, error) {
	rows, err := r.data.DB(ctx).QueryContext(ctx,
		"SELECT "+categoryColumns+" FROM categories WHERE deleted_at IS NULL ORDER BY sort ASC, id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*categoryPO
	for rows.Next() {
		var c categoryPO
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.Level, &c.IsTab, &c.Sort); err != nil {
			return nil, err
		}
		list = append(list, &c)
	}
	return list, rows.Err()
}

func (r *CategoryRepo) Category(ctx context.Context) ([]*biz.Category, error) {
	list, err := r.loadAllCategories(ctx)
	if err != nil {
		return nil, errors.InternalServer("CATEGORY_NOT_FOUND", err.Error())
	}
	if len(list) == 0 {
		return nil, errors.NotFound("CATEGORY_NOT_FOUND", "分类不存在")
	}

	// Group children by parent id.
	byParent := make(map[int32][]*categoryPO, len(list))
	for _, c := range list {
		byParent[c.ParentID] = append(byParent[c.ParentID], c)
	}

	// attach recursively builds a category node and its descendants.
	var attach func(c *categoryPO) *biz.Category
	attach = func(c *categoryPO) *biz.Category {
		node := &biz.Category{
			ID:               c.ID,
			Name:             c.Name,
			ParentCategoryID: c.ParentID,
			Level:            c.Level,
			IsTab:            c.IsTab,
			Sort:             c.Sort,
		}
		for _, child := range byParent[c.ID] {
			node.SubCategory = append(node.SubCategory, attach(child))
		}
		return node
	}

	var res []*biz.Category
	for _, c := range list {
		if c.Level == 1 {
			res = append(res, attach(c))
		}
	}
	return res, nil
}

func (r *CategoryRepo) GetCategoryByID(ctx context.Context, id int32) (*biz.CategoryInfo, error) {
	var c categoryPO
	err := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+categoryColumns+" FROM categories WHERE id = ? AND deleted_at IS NULL", id,
	).Scan(&c.ID, &c.Name, &c.ParentID, &c.Level, &c.IsTab, &c.Sort)
	if err == sql.ErrNoRows {
		return nil, errors.NotFound("CATEGORY_NOT_FOUND", "分类不存在")
	}
	if err != nil {
		return nil, errors.InternalServer("CATEGORY_ERROR", err.Error())
	}
	return &biz.CategoryInfo{
		ID:             c.ID,
		Name:           c.Name,
		ParentCategory: c.ParentID,
		Level:          c.Level,
		IsTab:          c.IsTab,
		Sort:           c.Sort,
	}, nil
}

func (r *CategoryRepo) SubCategory(ctx context.Context, req biz.CategoryInfo) ([]*biz.CategoryInfo, error) {
	rows, err := r.data.DB(ctx).QueryContext(ctx,
		"SELECT "+categoryColumns+" FROM categories WHERE parent_category_id = ? AND deleted_at IS NULL ORDER BY sort ASC, id ASC",
		req.ID,
	)
	if err != nil {
		return nil, errors.NotFound("CATEGORY_NOT_FOUND", "分类不存在")
	}
	defer rows.Close()

	var out []*biz.CategoryInfo
	for rows.Next() {
		var c categoryPO
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID, &c.Level, &c.IsTab, &c.Sort); err != nil {
			return nil, errors.NotFound("CATEGORY_NOT_FOUND", "分类不存在")
		}
		out = append(out, &biz.CategoryInfo{
			ID:             c.ID,
			Name:           c.Name,
			ParentCategory: c.ParentID,
			Level:          c.Level,
			IsTab:          c.IsTab,
			Sort:           c.Sort,
		})
	}
	return out, rows.Err()
}

func (r *CategoryRepo) GetCategoryAll(ctx context.Context, level, id int32) ([]interface{}, error) {
	var subQuery string
	// 把一级级分类下的所有三级分类都拿到
	if level == 1 {
		subQuery = fmt.Sprintf("SELECT id FROM categories WHERE parent_category_id IN (SELECT id FROM categories WHERE parent_category_id=%d)", id)
	} else if level == 2 {
		subQuery = fmt.Sprintf("SELECT id FROM categories WHERE parent_category_id=%d", id)
	} else if level == 3 {
		subQuery = fmt.Sprintf("SELECT id FROM categories WHERE id=%d", id)
	}

	rows, err := r.data.DB(ctx).QueryContext(ctx, subQuery)
	if err != nil {
		return nil, errors.InternalServer("CATEGORY_ERROR", err.Error())
	}
	defer rows.Close()

	categoryIds := make([]interface{}, 0)
	for rows.Next() {
		var cid int32
		if err := rows.Scan(&cid); err != nil {
			return nil, errors.InternalServer("CATEGORY_ERROR", err.Error())
		}
		categoryIds = append(categoryIds, cid)
	}
	return categoryIds, rows.Err()
}
