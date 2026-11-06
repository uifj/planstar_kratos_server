// Code in this file is the translation boundary between the mall DTOs
// (api/mall/v1) and the domain objects (internal/biz). It is the only place
// where protobuf messages are read into DOs or DOs are projected into reply
// messages. Handlers in shop.go / admin.go call these helpers so they stay
// thin: validate a request, call one usecase, project the result.
//
// This file imports only v1 and biz. It never imports data, ent, redis,
// elastic, amqp or any transport package, which keeps the DTO -> DO arrow
// one-way as required by AGENTS.md.
package service

import (
	"time"

	v1 "github.com/go-kratos/kratos-layout/api/mall/v1"
	"github.com/go-kratos/kratos-layout/internal/biz"
)

// ---------------------------------------------------------------------------
// User / auth
// ---------------------------------------------------------------------------

func displayName(u *biz.User) string {
	if u == nil {
		return ""
	}
	if u.Username != "" {
		return u.Username
	}
	return u.NickName
}

func registerReply(u *biz.User, token string, exp int64) *v1.RegisterReply {
	if u == nil {
		return &v1.RegisterReply{}
	}
	return &v1.RegisterReply{
		Id:        u.ID,
		Mobile:    u.Mobile,
		Username:  displayName(u),
		Token:     token,
		ExpiredAt: exp,
	}
}

func captchaReply(id, image string) *v1.CaptchaReply {
	return &v1.CaptchaReply{CaptchaId: id, PicPath: image}
}

func userDetailReply(u *biz.User, codes []string) *v1.UserDetailResponse {
	if u == nil {
		return &v1.UserDetailResponse{}
	}
	if codes == nil {
		codes = []string{}
	}
	return &v1.UserDetailResponse{
		Id:          u.ID,
		Mobile:      u.Mobile,
		NickName:    u.NickName,
		Birthday:    u.Birthday,
		Gender:      u.Gender,
		Role:        u.Role,
		Permissions: codes,
	}
}

func userInfo(u *biz.User) *v1.UserInfo {
	if u == nil {
		return nil
	}
	return &v1.UserInfo{
		Id:       u.ID,
		Mobile:   u.Mobile,
		NickName: u.NickName,
		Birthday: u.Birthday,
		Gender:   u.Gender,
		Role:     u.Role,
	}
}

func userListReply(list []*biz.User, total int32) *v1.UserListReply {
	out := &v1.UserListReply{Total: total, List: make([]*v1.UserInfo, 0, len(list))}
	for _, u := range list {
		if info := userInfo(u); info != nil {
			out.List = append(out.List, info)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Address
// ---------------------------------------------------------------------------

// addressFromCreate maps a CreateAddressReq into a DO. The uid field is
// deliberately ignored: ownership is always derived from the JWT claims in
// the handler, never trusted from the client.
func addressFromCreate(r *v1.CreateAddressReq) *biz.Address {
	if r == nil {
		return nil
	}
	return &biz.Address{
		Name:      r.GetName(),
		Mobile:    r.GetMobile(),
		Province:  r.GetProvince(),
		City:      r.GetCity(),
		Districts: r.GetDistricts(),
		Address:   r.GetAddress(),
		PostCode:  r.GetPostCode(),
		IsDefault: r.GetIsDefault(),
	}
}

func addressFromUpdate(r *v1.UpdateAddressReq) *biz.Address {
	if r == nil {
		return nil
	}
	return &biz.Address{
		ID:        r.GetId(),
		Name:      r.GetName(),
		Mobile:    r.GetMobile(),
		Province:  r.GetProvince(),
		City:      r.GetCity(),
		Districts: r.GetDistricts(),
		Address:   r.GetAddress(),
		PostCode:  r.GetPostCode(),
		IsDefault: r.GetIsDefault(),
	}
}

func addressInfo(a *biz.Address) *v1.AddressInfo {
	if a == nil {
		return nil
	}
	return &v1.AddressInfo{
		Id:        a.ID,
		Name:      a.Name,
		Mobile:    a.Mobile,
		Province:  a.Province,
		City:      a.City,
		Districts: a.Districts,
		Address:   a.Address,
		PostCode:  a.PostCode,
		IsDefault: a.IsDefault,
	}
}

func addressListReply(list []*biz.Address) *v1.ListAddressReply {
	out := &v1.ListAddressReply{Results: make([]*v1.AddressInfo, 0, len(list))}
	for _, a := range list {
		if info := addressInfo(a); info != nil {
			out.Results = append(out.Results, info)
		}
	}
	return out
}

func userAddressListReply(list []*biz.Address) *v1.UserAddressListReply {
	out := &v1.UserAddressListReply{List: make([]*v1.AddressInfo, 0, len(list))}
	for _, a := range list {
		if info := addressInfo(a); info != nil {
			out.List = append(out.List, info)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Goods / catalogue
// ---------------------------------------------------------------------------

func goodsQuery(r *v1.GoodsListRequest) biz.GoodsQuery {
	if r == nil {
		return biz.GoodsQuery{}
	}
	return biz.GoodsQuery{
		Page:       int(r.GetPage()),
		PageSize:   int(r.GetPageSize()),
		Keywords:   r.GetKeywords(),
		CategoryID: r.GetCategoryId(),
		BrandID:    r.GetBrandId(),
		SkuCode:    r.GetSkuCode(),
	}
}

func goodsItem(g *biz.Goods) *v1.GoodsItem {
	if g == nil {
		return nil
	}
	return &v1.GoodsItem{
		Id:           g.ID,
		Name:         g.Name,
		GoodsSn:      g.GoodsSn,
		MarketPrice:  g.MarketPrice,
		OnSale:       g.OnSale,
		IsNew:        g.IsNew,
		IsHot:        g.IsHot,
		CategoryId:   g.CategoryID,
		BrandId:      g.BrandID,
		CategoryName: g.CategoryName,
		BrandName:    g.BrandName,
		SkuCount:     int32(len(g.Skus)),
		SoldNum:      g.SoldNum,
	}
}

func goodsListReply(list []*biz.Goods, total int32) *v1.GoodsListReply {
	out := &v1.GoodsListReply{Total: total, List: make([]*v1.GoodsItem, 0, len(list))}
	for _, g := range list {
		if item := goodsItem(g); item != nil {
			out.List = append(out.List, item)
		}
	}
	return out
}

func goodsSkuDetail(s *biz.GoodsSku) *v1.GoodsSkuDetail {
	if s == nil {
		return nil
	}
	return &v1.GoodsSkuDetail{
		Id:             s.ID,
		GoodsId:        s.GoodsID,
		SkuName:        s.SkuName,
		SkuCode:        s.SkuCode,
		BarCode:        s.BarCode,
		Price:          s.Price,
		PromotionPrice: s.PromotionPrice,
		Inventory:      s.Inventory,
		OnSale:         s.OnSale,
		Pic:            s.Pic,
	}
}

func goodsImageDetail(im *biz.GoodsImage) *v1.GoodsImageDetail {
	if im == nil {
		return nil
	}
	return &v1.GoodsImageDetail{
		Id:       im.ID,
		Url:      im.URL,
		Position: im.Position,
		IsMaster: im.IsMaster,
	}
}

func goodsDetailReply(g *biz.Goods) *v1.GoodsDetailReply {
	if g == nil {
		return nil
	}
	out := &v1.GoodsDetailReply{
		Id:              g.ID,
		CategoryId:      g.CategoryID,
		BrandId:         g.BrandID,
		TypeId:          g.TypeID,
		Name:            g.Name,
		GoodsSn:         g.GoodsSn,
		GoodsBrief:      g.GoodsBrief,
		GoodsFrontImage: g.GoodsFrontImage,
		GoodsImages:     make([]string, 0, len(g.Images)),
		MarketPrice:     g.MarketPrice,
		OnSale:          g.OnSale,
		ShipFree:        g.ShipFree,
		IsNew:           g.IsNew,
		IsHot:           g.IsHot,
		SoldNum:         g.SoldNum,
		BrandName:       g.BrandName,
		CategoryName:    g.CategoryName,
		Skus:            make([]*v1.GoodsSkuDetail, 0, len(g.Skus)),
		Images:          make([]*v1.GoodsImageDetail, 0, len(g.Images)),
	}
	for _, s := range g.Skus {
		if d := goodsSkuDetail(s); d != nil {
			out.Skus = append(out.Skus, d)
		}
	}
	for _, im := range g.Images {
		if d := goodsImageDetail(im); d != nil {
			out.Images = append(out.Images, d)
			out.GoodsImages = append(out.GoodsImages, im.URL)
		}
	}
	return out
}

// goodsFromCreate folds the flat admin create form into a Goods aggregate
// with a single SKU. The biz usecase assigns ids, normalises statuses and
// seeds one inventory row per SKU.
func goodsFromCreate(r *v1.CreateGoodsRequest) *biz.Goods {
	if r == nil {
		return nil
	}
	qty := r.GetSkuInventory()
	if qty == 0 {
		qty = r.GetInventory()
	}
	g := &biz.Goods{
		CategoryID:      r.GetCategoryId(),
		BrandID:         r.GetBrandId(),
		TypeID:          r.GetTypeId(),
		Name:            r.GetName(),
		GoodsSn:         r.GetGoodsSn(),
		MarketPrice:     r.GetMarketPrice(),
		GoodsBrief:      r.GetGoodsBrief(),
		GoodsFrontImage: r.GetGoodsFrontImage(),
		ShipFree:        r.GetShipFree(),
		IsNew:           r.GetIsNew(),
		IsHot:           r.GetIsHot(),
		OnSale:          r.GetOnSale(),
		Skus: []*biz.GoodsSku{{
			SkuName:        r.GetSkuName(),
			SkuCode:        r.GetSkuCode(),
			BarCode:        r.GetBarCode(),
			Price:          r.GetSkuPrice(),
			PromotionPrice: r.GetPromotionPrice(),
			Inventory:      qty,
			OnSale:         r.GetOnSale(),
			Pic:            r.GetSkuImage(),
		}},
	}
	if image := r.GetGoodsFrontImage(); image != "" {
		g.Images = []*biz.GoodsImage{{URL: image, Position: 1, IsMaster: true}}
	}
	return g
}

func goodsFromUpdate(r *v1.UpdateGoodsRequest) *biz.Goods {
	if r == nil {
		return nil
	}
	return &biz.Goods{
		ID:              r.GetId(),
		Name:            r.GetName(),
		NameAlias:       r.GetNameAlias(),
		GoodsBrief:      r.GetGoodsBrief(),
		GoodsFrontImage: r.GetGoodsFrontImage(),
		MarketPrice:     r.GetMarketPrice(),
		ShipFree:        r.GetShipFree(),
		IsNew:           r.GetIsNew(),
		IsHot:           r.GetIsHot(),
	}
}

func categoryItem(c *biz.Category) *v1.CategoryItem {
	if c == nil {
		return nil
	}
	return &v1.CategoryItem{
		Id:             c.ID,
		Name:           c.Name,
		ParentCategory: c.ParentCategory,
		Level:          c.Level,
	}
}

func categoryListReply(list []*biz.Category) *v1.CategoryListReply {
	out := &v1.CategoryListReply{List: make([]*v1.CategoryItem, 0, len(list))}
	for _, c := range list {
		if item := categoryItem(c); item != nil {
			out.List = append(out.List, item)
		}
	}
	return out
}

func categoryFromSave(r *v1.CategorySaveRequest) *biz.Category {
	if r == nil {
		return nil
	}
	return &biz.Category{
		ID:             r.GetId(),
		Name:           r.GetName(),
		ParentCategory: r.GetParentCategory(),
		Level:          r.GetLevel(),
		Sort:           r.GetSort(),
	}
}

func brandItem(b *biz.Brand) *v1.BrandItem {
	if b == nil {
		return nil
	}
	return &v1.BrandItem{Id: b.ID, Name: b.Name, Logo: b.Logo}
}

func brandListReply(list []*biz.Brand) *v1.BrandListReply {
	out := &v1.BrandListReply{List: make([]*v1.BrandItem, 0, len(list))}
	for _, b := range list {
		if item := brandItem(b); item != nil {
			out.List = append(out.List, item)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Cart
// ---------------------------------------------------------------------------

func cartItemFromAdd(r *v1.CartAddRequest) *biz.CartItem {
	if r == nil {
		return nil
	}
	return &biz.CartItem{
		GoodsID:   r.GetGoodsId(),
		GoodsSn:   r.GetGoodsSn(),
		GoodsName: r.GetGoodsName(),
		SkuID:     r.GetSkuId(),
		GoodsNum:  r.GetGoodsNum(),
		IsSelect:  r.GetIsSelect(),
	}
}

func cartItemReply(c *biz.CartItem) *v1.CartItemReply {
	if c == nil {
		return nil
	}
	return &v1.CartItemReply{
		Id:         c.ID,
		UserId:     c.UserID,
		GoodsId:    c.GoodsID,
		GoodsSn:    c.GoodsSn,
		GoodsName:  c.GoodsName,
		SkuId:      c.SkuID,
		GoodsPrice: c.GoodsPrice,
		GoodsNum:   c.GoodsNum,
		IsSelect:   c.IsSelect,
	}
}

func cartListReply(list []*biz.CartItem) *v1.CartListReply {
	out := &v1.CartListReply{List: make([]*v1.CartItemReply, 0, len(list))}
	for _, c := range list {
		if item := cartItemReply(c); item != nil {
			out.List = append(out.List, item)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Order
// ---------------------------------------------------------------------------

func orderInputs(items []*v1.OrderCartItem) []biz.OrderItemInput {
	out := make([]biz.OrderItemInput, 0, len(items))
	for _, it := range items {
		if it == nil {
			continue
		}
		out = append(out, biz.OrderItemInput{
			CartID: it.GetCartId(),
			SkuID:  it.GetSkuId(),
			Num:    it.GetSkuNum(),
		})
	}
	return out
}

func orderGoodsReply(g *biz.OrderGoods) *v1.OrderGoodsReply {
	if g == nil {
		return nil
	}
	return &v1.OrderGoodsReply{
		Id:         g.ID,
		SkuId:      g.SkuID,
		SkuName:    g.SkuName,
		SkuPrice:   g.SkuPrice,
		Num:        g.Num,
		TotalPrice: g.TotalPrice,
	}
}

// formatAddTime renders the unix-seconds order timestamp the same way the
// source project did (order.CreatedAt.Format("2006-01-02 15:04:05")).
func formatAddTime(unix int64) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).Local().Format("2006-01-02 15:04:05")
}

func orderInfoReply(o *biz.Order) *v1.OrderInfoReply {
	if o == nil {
		return nil
	}
	out := &v1.OrderInfoReply{
		Id:      o.ID,
		UserId:  o.UserID,
		OrderSn: o.OrderSn,
		Status:  o.Status,
		Post:    o.Post,
		Total:   o.Total,
		Address: o.Address,
		Name:    o.Name,
		Mobile:  o.Mobile,
		AddTime: formatAddTime(o.AddTime),
		Goods:   make([]*v1.OrderGoodsReply, 0, len(o.Goods)),
	}
	for _, g := range o.Goods {
		if r := orderGoodsReply(g); r != nil {
			out.Goods = append(out.Goods, r)
		}
	}
	return out
}

func orderListReply(list []*biz.Order, total int32) *v1.OrderListReply {
	out := &v1.OrderListReply{Total: total, List: make([]*v1.OrderInfoReply, 0, len(list))}
	for _, o := range list {
		if info := orderInfoReply(o); info != nil {
			out.List = append(out.List, info)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Payment
// ---------------------------------------------------------------------------

func paymentInfoReply(p *biz.Payment) *v1.PaymentInfoReply {
	if p == nil {
		return nil
	}
	return &v1.PaymentInfoReply{
		Id:        p.ID,
		PaymentNo: p.PaymentNo,
		OrderSn:   p.OrderSn,
		Amount:    p.Amount,
		Channel:   p.Channel,
		Status:    p.Status,
	}
}

// ---------------------------------------------------------------------------
// Dashboard / RBAC
// ---------------------------------------------------------------------------

func dashboardReply(s *biz.DashboardStats) *v1.DashboardStatsReply {
	if s == nil {
		return &v1.DashboardStatsReply{}
	}
	out := &v1.DashboardStatsReply{
		TotalOrders: s.TotalOrders,
		TotalSales:  s.TotalSales,
		TodayOrders: s.TodayOrders,
		TodaySales:  s.TodaySales,
		TotalUsers:  s.TotalUsers,
		StatusCounts: make([]*v1.StatusCount, 0, len(s.StatusCounts)),
		Last30Days:  make([]*v1.DailySales, 0, len(s.Last30Days)),
		TopGoods:    make([]*v1.TopGoods, 0, len(s.TopGoods)),
	}
	for _, c := range s.StatusCounts {
		out.StatusCounts = append(out.StatusCounts, &v1.StatusCount{Status: c.Status, Count: c.Count})
	}
	for _, d := range s.Last30Days {
		out.Last30Days = append(out.Last30Days, &v1.DailySales{Date: d.Date, OrderCount: d.OrderCount, Amount: d.Amount})
	}
	for _, g := range s.TopGoods {
		out.TopGoods = append(out.TopGoods, &v1.TopGoods{SkuId: g.SkuID, SkuName: g.SkuName, Num: g.Num, Amount: g.Amount})
	}
	return out
}

func permissionItem(p *biz.Permission) *v1.PermissionItem {
	if p == nil {
		return nil
	}
	return &v1.PermissionItem{
		Id:        p.ID,
		Code:      p.Code,
		Name:      p.Name,
		GroupName: p.GroupName,
		Sort:      p.Sort,
	}
}

func permissionListReply(list []*biz.Permission) *v1.PermissionListReply {
	out := &v1.PermissionListReply{List: make([]*v1.PermissionItem, 0, len(list))}
	for _, p := range list {
		if item := permissionItem(p); item != nil {
			out.List = append(out.List, item)
		}
	}
	return out
}
