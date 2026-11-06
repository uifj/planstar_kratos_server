// Code in this file is the transport adapter for the mall API. Each handler
// converts a DTO into a DO, delegates to a biz usecase, and converts the DO
// back into the reply DTO. It never imports data, ent, redis, elastic or amqp.
//
// C-end BFF (22 RPCs). The owner of a request is always derived from the JWT
// claims via biz.UserIDFromContext, never from a client-supplied uid.
package service

import (
	"context"

	v1 "github.com/go-kratos/kratos-layout/api/mall/v1"
	"github.com/go-kratos/kratos-layout/internal/biz"

	"google.golang.org/protobuf/types/known/emptypb"
)

// ShopService implements v1.ShopServiceServer. It holds only usecases: no
// data, ent, redis, elastic or amqp imports are allowed in this layer.
type ShopService struct {
	v1.UnimplementedShopServiceServer

	users     *biz.UserUsecase
	addresses *biz.AddressUsecase
	goods     *biz.GoodsUsecase
	carts     *biz.CartUsecase
	orders    *biz.OrderUsecase
	payments  *biz.PaymentUsecase
}

// NewShopService creates the ShopService.
func NewShopService(
	users *biz.UserUsecase,
	addresses *biz.AddressUsecase,
	goods *biz.GoodsUsecase,
	carts *biz.CartUsecase,
	orders *biz.OrderUsecase,
	payments *biz.PaymentUsecase,
) *ShopService {
	return &ShopService{
		users:     users,
		addresses: addresses,
		goods:     goods,
		carts:     carts,
		orders:    orders,
		payments:  payments,
	}
}

// Register creates a normal (AuthorityUser) account and returns a token.
func (s *ShopService) Register(ctx context.Context, req *v1.RegisterReq) (*v1.RegisterReply, error) {
	u, token, exp, err := s.users.Register(ctx, req.GetMobile(), req.GetUsername(), req.GetPassword(), biz.AuthorityUser)
	if err != nil {
		return nil, err
	}
	return registerReply(u, token, exp), nil
}

// RefreshToken validates the full claim set and issues a fresh token.
func (s *ShopService) RefreshToken(ctx context.Context, req *v1.RefreshTokenRequest) (*v1.RegisterReply, error) {
	u, token, exp, err := s.users.RefreshToken(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return registerReply(u, token, exp), nil
}

// Login authenticates by mobile and requires a valid captcha (C-end path).
func (s *ShopService) Login(ctx context.Context, req *v1.LoginReq) (*v1.RegisterReply, error) {
	u, token, exp, err := s.users.Login(ctx, req.GetMobile(), req.GetUsername(), req.GetPassword(), req.GetCaptcha(), req.GetCaptchaId(), true)
	if err != nil {
		return nil, err
	}
	return registerReply(u, token, exp), nil
}

// Captcha issues a Redis-backed captcha challenge.
func (s *ShopService) Captcha(ctx context.Context, _ *emptypb.Empty) (*v1.CaptchaReply, error) {
	id, image, err := s.users.Captcha(ctx)
	if err != nil {
		return nil, err
	}
	return captchaReply(id, image), nil
}

// Detail returns the caller's profile plus its role's permission codes.
func (s *ShopService) Detail(ctx context.Context, _ *emptypb.Empty) (*v1.UserDetailResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	u, codes, err := s.users.Detail(ctx, uid)
	if err != nil {
		return nil, err
	}
	return userDetailReply(u, codes), nil
}

// CreateAddress adds an address owned by the authenticated caller.
func (s *ShopService) CreateAddress(ctx context.Context, req *v1.CreateAddressReq) (*v1.AddressInfo, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	created, err := s.addresses.Create(ctx, uid, addressFromCreate(req))
	if err != nil {
		return nil, err
	}
	return addressInfo(created), nil
}

// AddressListByUid lists the authenticated caller's addresses.
func (s *ShopService) AddressListByUid(ctx context.Context, _ *emptypb.Empty) (*v1.ListAddressReply, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.addresses.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	return addressListReply(list), nil
}

// UpdateAddress edits an address owned by the authenticated caller.
func (s *ShopService) UpdateAddress(ctx context.Context, req *v1.UpdateAddressReq) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.addresses.Update(ctx, uid, addressFromUpdate(req)); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// DefaultAddress marks one of the caller's addresses as default.
func (s *ShopService) DefaultAddress(ctx context.Context, req *v1.AddressReq) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.addresses.SetDefault(ctx, uid, req.GetId()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// DeleteAddress soft-deletes one of the caller's addresses.
func (s *ShopService) DeleteAddress(ctx context.Context, req *v1.AddressReq) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.addresses.Delete(ctx, uid, req.GetId()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// GoodsList searches on-sale goods (Elasticsearch with a SQL fallback).
func (s *ShopService) GoodsList(ctx context.Context, req *v1.GoodsListRequest) (*v1.GoodsListReply, error) {
	list, total, err := s.goods.List(ctx, goodsQuery(req))
	if err != nil {
		return nil, err
	}
	return goodsListReply(list, total), nil
}

// GoodsDetail returns the aggregate for one goods row.
func (s *ShopService) GoodsDetail(ctx context.Context, req *v1.GoodsDetailRequest) (*v1.GoodsDetailReply, error) {
	g, err := s.goods.Detail(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return goodsDetailReply(g), nil
}

// CartList lists the caller's cart lines.
func (s *ShopService) CartList(ctx context.Context, _ *v1.CartListRequest) (*v1.CartListReply, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.carts.List(ctx, uid)
	if err != nil {
		return nil, err
	}
	return cartListReply(list), nil
}

// CartAdd adds a line, re-reading the authoritative price from the SKU.
func (s *ShopService) CartAdd(ctx context.Context, req *v1.CartAddRequest) (*v1.CartItemReply, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.carts.Add(ctx, uid, cartItemFromAdd(req))
	if err != nil {
		return nil, err
	}
	return cartItemReply(item), nil
}

// CartUpdate changes the quantity of a cart line.
func (s *ShopService) CartUpdate(ctx context.Context, req *v1.CartUpdateRequest) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.carts.Update(ctx, uid, req.GetId(), req.GetGoodsNum(), nil); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// CartDelete removes a cart line.
func (s *ShopService) CartDelete(ctx context.Context, req *v1.CartDeleteRequest) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.carts.Delete(ctx, uid, req.GetId()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// OrderCreate prices the requested lines server-side and persists the order,
// its snapshot, its line items and the order.created outbox row in one tx.
func (s *ShopService) OrderCreate(ctx context.Context, req *v1.OrderCreateRequest) (*v1.OrderInfoReply, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.orders.Create(ctx, uid, req.GetAddress(), orderInputs(req.GetCartItem()))
	if err != nil {
		return nil, err
	}
	return orderInfoReply(o), nil
}

// OrderCancel cancels a pending/locked order and releases its locks.
func (s *ShopService) OrderCancel(ctx context.Context, req *v1.OrderCancelRequest) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.orders.Cancel(ctx, uid, req.GetOrderSn()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// OrderList lists the caller's orders, optionally filtered by status.
func (s *ShopService) OrderList(ctx context.Context, req *v1.OrderListRequest) (*v1.OrderListReply, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	list, total, err := s.orders.List(ctx, uid, int(req.GetStatus()), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return orderListReply(list, total), nil
}

// OrderDetail returns one of the caller's orders.
func (s *ShopService) OrderDetail(ctx context.Context, req *v1.OrderDetailRequest) (*v1.OrderInfoReply, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	o, err := s.orders.Detail(ctx, uid, req.GetOrderSn())
	if err != nil {
		return nil, err
	}
	return orderInfoReply(o), nil
}

// PaymentCreate opens a pending payment for a locked order.
func (s *ShopService) PaymentCreate(ctx context.Context, req *v1.PaymentCreateRequest) (*v1.PaymentInfoReply, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	p, err := s.payments.Create(ctx, uid, req.GetOrderSn(), req.GetAmount(), req.GetChannel())
	if err != nil {
		return nil, err
	}
	return paymentInfoReply(p), nil
}

// PaymentMockPay marks a payment successful and writes payment.success to the
// outbox in the same transaction.
func (s *ShopService) PaymentMockPay(ctx context.Context, req *v1.PaymentMockPayRequest) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.payments.MockPay(ctx, uid, req.GetPaymentNo()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}
