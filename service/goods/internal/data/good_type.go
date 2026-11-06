package data

import (
	"context"
	"database/sql"
	serrors "errors"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"log/slog"

	"goods/internal/biz"
	"goods/internal/domain"
)

const goodsTypeColumns = "id, name, type_code, name_alias, is_virtual, `desc`, sort"

type goodsTypeRepo struct {
	data *Data
	log  *slog.Logger
}

// NewGoodsTypeRepo .
func NewGoodsTypeRepo(data *Data, logger *slog.Logger) biz.GoodsTypeRepo {
	return &goodsTypeRepo{
		data: data,
		log:  logger,
	}
}

// scanGoodsType reads one goods_types row into a domain GoodsType.
func scanGoodsType(row interface {
	Scan(dest ...any) error
}) (*domain.GoodsType, error) {
	var t domain.GoodsType
	if err := row.Scan(&t.ID, &t.Name, &t.TypeCode, &t.NameAlias, &t.IsVirtual, &t.Desc, &t.Sort); err != nil {
		return nil, err
	}
	return &t, nil
}

// CreateGoodsType 创建基本的商品类型
func (g *goodsTypeRepo) CreateGoodsType(ctx context.Context, req *domain.GoodsType) (int64, error) {
	res, err := g.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO goods_types (name, type_code, name_alias, is_virtual, `desc`, sort, add_time, update_time) "+
			"VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())",
		req.Name, req.TypeCode, req.NameAlias, req.IsVirtual, req.Desc, req.Sort,
	)
	if err != nil {
		return 0, errors.InternalServer("GOODS_TYPE_SAVE_ERROR", err.Error())
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (g *goodsTypeRepo) CreateGoodsBrandType(ctx context.Context, typeID int64, brandIds string) error {
	ids := strings.Split(brandIds, ",")
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids)*2)
	for _, id := range ids {
		j, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			return errors.InternalServer("GOODS_TYPE_CREATE_ERROR", err.Error())
		}
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, j, typeID)
	}
	if len(placeholders) == 0 {
		return nil
	}
	q := "INSERT INTO goods_type_brands (brand_id, type_id) VALUES " + strings.Join(placeholders, ", ")
	if _, err := g.data.DB(ctx).ExecContext(ctx, q, args...); err != nil {
		return errors.InternalServer("GOODS_TYPE_CREATE_ERROR", err.Error())
	}
	return nil
}

func (g *goodsTypeRepo) GetGoodsTypeByID(ctx context.Context, typeID int64) (*domain.GoodsType, error) {
	goodsType, err := scanGoodsType(g.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+goodsTypeColumns+" FROM goods_types WHERE id = ? AND deleted_at IS NULL", typeID))
	if serrors.Is(err, sql.ErrNoRows) {
		return nil, errors.NotFound("GOODS_TYPE_NOT_FOUND", "商品类型不存在")
	}
	if err != nil {
		return nil, errors.InternalServer("GOODS_TYPE_GET_ERROR", err.Error())
	}
	return goodsType, nil
}

func (g *goodsTypeRepo) IsExistsByID(ctx context.Context, typeID int64) (*domain.GoodsType, error) {
	return g.GetGoodsTypeByID(ctx, typeID)
}
