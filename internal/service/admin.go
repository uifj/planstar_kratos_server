// Code in this file is the transport adapter for the mall API. Each handler
// converts a DTO into a DO, delegates to a biz usecase, and converts the DO
// back into the reply DTO. It never imports data, ent, redis, elastic or amqp.
//
// Admin BFF (32 RPCs). adminOnly middleware already enforces
// Claims.AuthorityID == biz.AuthorityAdmin for every non-public operation,
// so handlers do not re-check the role.
package service

import (
	"context"

	v1 "github.com/go-kratos/kratos-layout/api/mall/v1"
	"github.com/go-kratos/kratos-layout/internal/biz"

	"google.golang.org/protobuf/types/known/emptypb"
)

// AdminService implements v1.AdminServiceServer.
type AdminService struct {
	v1.UnimplementedAdminServiceServer

	users       *biz.UserUsecase
	addresses   *biz.AddressUsecase
	goods       *biz.GoodsUsecase
	orders      *biz.OrderUsecase
	permissions *biz.PermissionUsecase
}

// NewAdminService creates the AdminService.
func NewAdminService(
	users *biz.UserUsecase,
	addresses *biz.AddressUsecase,
	goods *biz.GoodsUsecase,
	orders *biz.OrderUsecase,
	permissions *biz.PermissionUsecase,
) *AdminService {
	return &AdminService{
		users:       users,
		addresses:   addresses,
		goods:       goods,
		orders:      orders,
		permissions: permissions,
	}
}

// Register creates an AuthorityAdmin account.
func (s *AdminService) Register(ctx context.Context, req *v1.RegisterReq) (*v1.RegisterReply, error) {
	u, token, exp, err := s.users.Register(ctx, req.GetMobile(), req.GetUsername(), req.GetPassword(), biz.AuthorityAdmin)
	if err != nil {
		return nil, err
	}
	return registerReply(u, token, exp), nil
}

// RefreshToken validates the full claim set and issues a fresh token.
func (s *AdminService) RefreshToken(ctx context.Context, req *v1.RefreshTokenRequest) (*v1.RegisterReply, error) {
	u, token, exp, err := s.users.RefreshToken(ctx, req.GetToken())
	if err != nil {
		return nil, err
	}
	return registerReply(u, token, exp), nil
}

// Login authenticates by username. The source admin BFF had captcha checking
// commented out, so verifyCaptcha is false here.
func (s *AdminService) Login(ctx context.Context, req *v1.LoginReq) (*v1.RegisterReply, error) {
	u, token, exp, err := s.users.Login(ctx, req.GetMobile(), req.GetUsername(), req.GetPassword(), req.GetCaptcha(), req.GetCaptchaId(), false)
	if err != nil {
		return nil, err
	}
	return registerReply(u, token, exp), nil
}

// Captcha issues a Redis-backed captcha challenge.
func (s *AdminService) Captcha(ctx context.Context, _ *emptypb.Empty) (*v1.CaptchaReply, error) {
	id, image, err := s.users.Captcha(ctx)
	if err != nil {
		return nil, err
	}
	return captchaReply(id, image), nil
}

// Detail returns the caller's profile plus its role's permission codes.
func (s *AdminService) Detail(ctx context.Context, _ *emptypb.Empty) (*v1.UserDetailResponse, error) {
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

// CreateAddress adds an address owned by the authenticated admin caller.
func (s *AdminService) CreateAddress(ctx context.Context, req *v1.CreateAddressReq) (*v1.AddressInfo, error) {
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

// AddressListByUid lists the authenticated admin caller's addresses.
func (s *AdminService) AddressListByUid(ctx context.Context, _ *emptypb.Empty) (*v1.ListAddressReply, error) {
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

// UpdateAddress edits an address owned by the authenticated admin caller.
func (s *AdminService) UpdateAddress(ctx context.Context, req *v1.UpdateAddressReq) (*v1.CheckResponse, error) {
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
func (s *AdminService) DefaultAddress(ctx context.Context, req *v1.AddressReq) (*v1.CheckResponse, error) {
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
func (s *AdminService) DeleteAddress(ctx context.Context, req *v1.AddressReq) (*v1.CheckResponse, error) {
	uid, err := biz.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.addresses.Delete(ctx, uid, req.GetId()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// GoodsList lists all goods from SQL, including off-sale rows.
func (s *AdminService) GoodsList(ctx context.Context, req *v1.GoodsListRequest) (*v1.GoodsListReply, error) {
	list, total, err := s.goods.AdminList(ctx, goodsQuery(req))
	if err != nil {
		return nil, err
	}
	return goodsListReply(list, total), nil
}

// UpdateGoods edits the mutable fields of a goods row.
func (s *AdminService) UpdateGoods(ctx context.Context, req *v1.UpdateGoodsRequest) (*v1.CheckResponse, error) {
	if err := s.goods.Update(ctx, goodsFromUpdate(req)); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// DeleteGoods soft-deletes a goods row.
func (s *AdminService) DeleteGoods(ctx context.Context, req *v1.DeleteGoodsRequest) (*v1.CheckResponse, error) {
	if err := s.goods.Delete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// UpdateGoodsStatus toggles the on_sale flag.
func (s *AdminService) UpdateGoodsStatus(ctx context.Context, req *v1.UpdateGoodsStatusRequest) (*v1.CheckResponse, error) {
	if err := s.goods.SetOnSale(ctx, req.GetId(), req.GetOnSale()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// OrderList lists all users' orders, optionally filtered by status.
func (s *AdminService) OrderList(ctx context.Context, req *v1.OrderListRequest) (*v1.OrderListReply, error) {
	list, total, err := s.orders.AdminList(ctx, int(req.GetStatus()), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return orderListReply(list, total), nil
}

// OrderDetail returns any user's order by order_sn.
func (s *AdminService) OrderDetail(ctx context.Context, req *v1.OrderDetailRequest) (*v1.OrderInfoReply, error) {
	o, err := s.orders.AdminDetail(ctx, req.GetOrderSn())
	if err != nil {
		return nil, err
	}
	return orderInfoReply(o), nil
}

// ShipOrder advances a paid order to shipped and records the tracking number.
func (s *AdminService) ShipOrder(ctx context.Context, req *v1.ShipOrderRequest) (*v1.CheckResponse, error) {
	if err := s.orders.Ship(ctx, req.GetOrderSn(), req.GetPost()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// RefundOrder advances a paid/shipped/completed order to refunding.
func (s *AdminService) RefundOrder(ctx context.Context, req *v1.RefundOrderRequest) (*v1.CheckResponse, error) {
	if err := s.orders.Refund(ctx, req.GetOrderSn()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// UserList lists registered accounts.
func (s *AdminService) UserList(ctx context.Context, req *v1.UserListRequest) (*v1.UserListReply, error) {
	list, total, err := s.users.List(ctx, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return userListReply(list, total), nil
}

// UserAddressList lists the addresses of a specific user (admin override).
func (s *AdminService) UserAddressList(ctx context.Context, req *v1.UserAddressListRequest) (*v1.UserAddressListReply, error) {
	list, err := s.addresses.AdminList(ctx, req.GetUid())
	if err != nil {
		return nil, err
	}
	return userAddressListReply(list), nil
}

// UserAddressDelete deletes any user's address by id (admin override).
func (s *AdminService) UserAddressDelete(ctx context.Context, req *v1.UserAddressDeleteRequest) (*v1.CheckResponse, error) {
	if err := s.addresses.AdminDelete(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// CreateGoods creates a goods aggregate with one SKU and seeds inventory.
func (s *AdminService) CreateGoods(ctx context.Context, req *v1.CreateGoodsRequest) (*v1.CheckResponse, error) {
	if err := s.goods.Create(ctx, goodsFromCreate(req)); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// GoodsDetail returns the aggregate for one goods row.
func (s *AdminService) GoodsDetail(ctx context.Context, req *v1.GoodsDetailRequest) (*v1.GoodsDetailReply, error) {
	g, err := s.goods.Detail(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return goodsDetailReply(g), nil
}

// CategoryList lists active categories.
func (s *AdminService) CategoryList(ctx context.Context, _ *emptypb.Empty) (*v1.CategoryListReply, error) {
	list, err := s.goods.Categories(ctx)
	if err != nil {
		return nil, err
	}
	return categoryListReply(list), nil
}

// BrandList lists active brands.
func (s *AdminService) BrandList(ctx context.Context, req *v1.BrandListRequest) (*v1.BrandListReply, error) {
	list, err := s.goods.Brands(ctx, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return brandListReply(list), nil
}

// CreateCategory creates a category.
func (s *AdminService) CreateCategory(ctx context.Context, req *v1.CategorySaveRequest) (*v1.CategoryItem, error) {
	c, err := s.goods.CreateCategory(ctx, categoryFromSave(req))
	if err != nil {
		return nil, err
	}
	return categoryItem(c), nil
}

// UpdateCategory edits a category.
func (s *AdminService) UpdateCategory(ctx context.Context, req *v1.CategorySaveRequest) (*v1.CheckResponse, error) {
	if err := s.goods.UpdateCategory(ctx, categoryFromSave(req)); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// DeleteCategory soft-deletes a category.
func (s *AdminService) DeleteCategory(ctx context.Context, req *v1.CategoryDeleteRequest) (*v1.CheckResponse, error) {
	if err := s.goods.DeleteCategory(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}

// DashboardStats returns the aggregated order/user statistics.
func (s *AdminService) DashboardStats(ctx context.Context, _ *emptypb.Empty) (*v1.DashboardStatsReply, error) {
	stats, err := s.orders.Dashboard(ctx)
	if err != nil {
		return nil, err
	}
	return dashboardReply(stats), nil
}

// PermissionList lists all permission definitions.
func (s *AdminService) PermissionList(ctx context.Context, _ *emptypb.Empty) (*v1.PermissionListReply, error) {
	list, err := s.permissions.List(ctx)
	if err != nil {
		return nil, err
	}
	return permissionListReply(list), nil
}

// RolePermissionList lists the permission codes granted to a role.
func (s *AdminService) RolePermissionList(ctx context.Context, req *v1.RolePermissionListRequest) (*v1.RolePermissionListReply, error) {
	codes, err := s.permissions.RoleList(ctx, req.GetRole())
	if err != nil {
		return nil, err
	}
	if codes == nil {
		codes = []string{}
	}
	return &v1.RolePermissionListReply{Codes: codes}, nil
}

// RolePermissionUpdate replaces the permission codes granted to a role.
func (s *AdminService) RolePermissionUpdate(ctx context.Context, req *v1.RolePermissionUpdateRequest) (*v1.CheckResponse, error) {
	if err := s.permissions.RoleUpdate(ctx, req.GetRole(), req.GetCodes()); err != nil {
		return nil, err
	}
	return okCheck(), nil
}
