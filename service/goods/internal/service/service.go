package service

import (
	"context"

	"log/slog"
	"github.com/google/wire"
	inventoryV1 "goods/api/service/inventory/v1"
	v1 "goods/api/goods/v1"
	"goods/internal/biz"
	"goods/internal/conf"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewGoodsService)

// GoodsService is a goods service.
type GoodsService struct {
	v1.UnimplementedGoodsServer
	cac     *biz.CategoryUsecase
	bc      *biz.BrandUsecase
	gt      *biz.GoodsTypeUsecase
	s       *biz.SpecificationUsecase
	ga      *biz.GoodsAttrUsecase
	g       *biz.GoodsUsecase
	esGoods *biz.EsGoodsUsecase
	log     *slog.Logger
	inv     inventoryV1.InventoryClient

	mqAddr string
}

// NewGoodsService new a goods service.
func NewGoodsService(bc *biz.BrandUsecase, cac *biz.CategoryUsecase, gt *biz.GoodsTypeUsecase, s *biz.SpecificationUsecase,
	ga *biz.GoodsAttrUsecase, gc *biz.GoodsUsecase, esGoods *biz.EsGoodsUsecase, inv inventoryV1.InventoryClient, mqConf *conf.Mq, logger *slog.Logger) *GoodsService {
	svc := &GoodsService{
		bc:      bc,
		cac:     cac,
		gt:      gt,
		s:       s,
		ga:      ga,
		g:       gc,
		esGoods: esGoods,
		log:     logger,
		inv:     inv,
	}
	if mqConf != nil && mqConf.Addr != "" {
		svc.mqAddr = mqConf.Addr
		go svc.startPaidConsumer()
	}
	go func() {
		if err := svc.g.ReindexAll(context.Background()); err != nil {
			svc.log.Error("reindex all goods failed", slog.Any("err", err))
		}
	}()
	return svc
}
