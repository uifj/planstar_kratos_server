package data

import (
	"context"
	"log/slog"

	"goods/internal/biz"
	"goods/internal/domain"
)

type goodsImagesRepo struct {
	data *Data
	log  *slog.Logger
}

// NewGoodsImagesRepo .
func NewGoodsImagesRepo(data *Data, logger *slog.Logger) biz.GoodsImagesRepo {
	return &goodsImagesRepo{
		data: data,
		log:  logger,
	}
}

func (r *goodsImagesRepo) CreateGreeter(ctx context.Context, g *domain.Goods) error {
	return nil
}

func (r *goodsImagesRepo) UpdateGreeter(ctx context.Context, g *domain.Goods) error {
	return nil
}
