package service

import (
	"net/http"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

// shopGoods 返回在售商品列表（匿名可访问）。
func (s *Service) shopGoods(w http.ResponseWriter, r *http.Request) {
	items, err := s.shop.GoodsList(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

// shopGoodsDetail 返回商品详情（匿名可访问）。
func (s *Service) shopGoodsDetail(w http.ResponseWriter, r *http.Request) {
	goodsID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	item, err := s.shop.GoodsDetail(r.Context(), goodsID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// redeemGoods 使用星币兑换商品规格（必须登录）。
func (s *Service) redeemGoods(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	goodsID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	skuID, ok := pathID(r, "skuId")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	result, err := s.shop.Redeem(r.Context(), userID, goodsID, skuID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
