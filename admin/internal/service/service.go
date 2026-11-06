package service

import (
	"log/slog"

	v1 "admin/api/admin/v1"
	"admin/internal/biz"
	"github.com/google/wire"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(NewAdminService)

// AdminService is a admin service.
type AdminService struct {
	v1.UnimplementedAdminServer

	uc  *biz.UserUsecase
	ua  *biz.AddressUsecase
	gu  *biz.GoodsUsecase
	ou  *biz.OrderUsecase
	pu  *biz.PermissionUsecase
	log *slog.Logger
}

// NewAdminService new a admin service.
func NewAdminService(uc *biz.UserUsecase, ua *biz.AddressUsecase, gu *biz.GoodsUsecase, ou *biz.OrderUsecase, pu *biz.PermissionUsecase, logger *slog.Logger) *AdminService {
	return &AdminService{
		uc:  uc,
		ua:  ua,
		gu:  gu,
		ou:  ou,
		pu:  pu,
		log: logger.With("module", "service/admin"),
	}
}
