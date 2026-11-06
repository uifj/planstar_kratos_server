package biz

import (
	"context"
	"log/slog"
	"goods/internal/domain"
)

type Sku struct {
	ID             int64
	GoodsID        int64
	GoodsSn        string
	GoodsName      string
	SkuName        string
	SkuCode        string
	BarCode        string
	Price          int64
	PromotionPrice int64
	Points         int64
	RemarksInfo    string
	Pic            string
	Inventory      int64
	OnSale         bool
	AttrInfo       string
}

type GoodsSkuRepo interface {
	Create(context.Context, *domain.GoodsSku) (*domain.GoodsSku, error)
	CreateSkuRelation(context.Context, []*domain.GoodsSpecificationSku) error
	ListByIds(context.Context, []int64) ([]*domain.GoodsSku, error)
	ListByGoodsID(context.Context, int64) ([]*domain.GoodsSku, error)
}

type GoodsSkuUsecase struct {
	repo GoodsSkuRepo
	log  *slog.Logger
}

func NewGoodsSkuUsecase(repo GoodsSkuRepo, logger *slog.Logger) *GoodsSkuUsecase {
	return &GoodsSkuUsecase{repo: repo, log: logger}
}
