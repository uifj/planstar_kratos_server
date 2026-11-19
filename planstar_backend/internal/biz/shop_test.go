package biz

import (
	"context"
	"errors"
	"testing"
)

// stubShopRepo 是商城仓储桩，用于验证错误透传与切片非 nil 约定。
type stubShopRepo struct {
	goods []*Goods
	err   error
}

func (s *stubShopRepo) Goods(_ context.Context) ([]*Goods, error) { return s.goods, nil }

func (s *stubShopRepo) GoodsDetail(_ context.Context, _ int64) (*Goods, error) {
	return nil, s.err
}

func (s *stubShopRepo) Redeem(_ context.Context, _, _, _ int64) (*RedeemResult, error) {
	return nil, s.err
}

func TestShopRedeemErrors(t *testing.T) {
	for _, want := range []error{ErrInsufficientCoins, ErrSoldOut} {
		uc := NewShopUsecase(&stubShopRepo{err: want})
		if _, err := uc.Redeem(context.Background(), 1, 1, 1); !errors.Is(err, want) {
			t.Fatalf("redeem err = %v, want %v", err, want)
		}
	}
}

func TestShopRedeemValidatesIDs(t *testing.T) {
	uc := NewShopUsecase(&stubShopRepo{})
	if _, err := uc.Redeem(context.Background(), 0, 1, 1); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("anonymous redeem err = %v, want ErrUnauthorized", err)
	}
	if _, err := uc.Redeem(context.Background(), 1, 0, 1); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("bad goods id err = %v, want ErrBadRequest", err)
	}
	if _, err := uc.Redeem(context.Background(), 1, 1, 0); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("bad sku id err = %v, want ErrBadRequest", err)
	}
}

func TestShopGoodsListNonNil(t *testing.T) {
	uc := NewShopUsecase(&stubShopRepo{})
	items, err := uc.GoodsList(context.Background())
	if err != nil {
		t.Fatalf("goods list err = %v", err)
	}
	if items == nil {
		t.Fatalf("goods list is nil, want empty slice")
	}
}
