package biz

import (
	"context"
	"log/slog"
	"goods/internal/domain"
)

type GoodsImages struct {
	Hello string
}

type GoodsImagesRepo interface {
	CreateGreeter(context.Context, *domain.Goods) error
	UpdateGreeter(context.Context, *domain.Goods) error
}

type GoodsImageUsecase struct {
	repo GoodsImagesRepo
	log  *slog.Logger
}

func NewGoodsImagesUsecase(repo GoodsImagesRepo, logger *slog.Logger) *GoodsImageUsecase {
	return &GoodsImageUsecase{repo: repo, log: logger}
}
