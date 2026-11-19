package biz

import "context"

// Goods 是星币商城商品（价格单位为星币）。
type Goods struct {
	GoodsID     int64      `json:"goodsId"`
	Name        string     `json:"name"`
	Brief       string     `json:"brief"`
	Description string     `json:"description"`
	Cover       string     `json:"cover"`
	SKUs        []GoodsSKU `json:"skus"`
}

// GoodsSKU 是商品规格（价格单位为星币）。
type GoodsSKU struct {
	SKUId      int64  `json:"skuId"`
	Name       string `json:"name"`
	PriceCoins int64  `json:"priceCoins"`
	Stock      int64  `json:"stock"`
	Sold       int64  `json:"sold"`
}

// RedeemResult 是兑换结果（balance 单位为分）。
type RedeemResult struct {
	OrderNo string `json:"orderNo"`
	Balance int64  `json:"balance"`
	Stock   int64  `json:"stock"`
	Sold    int64  `json:"sold"`
}

// ShopRepo 是星币商城仓储端口。
type ShopRepo interface {
	Goods(ctx context.Context) ([]*Goods, error)
	GoodsDetail(ctx context.Context, goodsID int64) (*Goods, error)
	Redeem(ctx context.Context, userID, goodsID, skuID int64) (*RedeemResult, error)
}

// ShopUsecase 星币商城用例。
type ShopUsecase struct {
	repo ShopRepo
}

func NewShopUsecase(repo ShopRepo) *ShopUsecase {
	return &ShopUsecase{repo: repo}
}

func (uc *ShopUsecase) GoodsList(ctx context.Context) ([]*Goods, error) {
	items, err := uc.repo.Goods(ctx)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*Goods{}
	}
	return items, nil
}

func (uc *ShopUsecase) GoodsDetail(ctx context.Context, goodsID int64) (*Goods, error) {
	if goodsID <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.GoodsDetail(ctx, goodsID)
}

func (uc *ShopUsecase) Redeem(ctx context.Context, userID, goodsID, skuID int64) (*RedeemResult, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if goodsID <= 0 || skuID <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.Redeem(ctx, userID, goodsID, skuID)
}
