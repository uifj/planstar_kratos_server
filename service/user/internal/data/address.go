package data

import (
	"context"
	"log/slog"

	"github.com/go-kratos/kratos/v3/errors"

	"user/internal/biz"
	"user/internal/data/ent"
	"user/internal/data/ent/address"
	"user/internal/domain"
)

type adderessRepo struct {
	data *Data
	log  *slog.Logger
}

// NewAddressRepo .
func NewAddressRepo(data *Data, logger *slog.Logger) biz.AddressRepo {
	return &adderessRepo{data: data, log: logger}
}

func toDomain(a *ent.Address) *domain.Address {
	return &domain.Address{
		ID:        a.ID,
		UserID:    a.UserID,
		IsDefault: a.IsDefault,
		Mobile:    a.Mobile,
		Name:      a.Name,
		Province:  a.Province,
		City:      a.City,
		Districts: a.Districts,
		Address:   a.Address,
		PostCode:  a.PostCode,
	}
}

func (r *adderessRepo) CreateAddress(ctx context.Context, req *domain.Address) (*domain.Address, error) {
	created, err := r.data.DB(ctx).Address.Create().
		SetUserID(req.UserID).
		SetIsDefault(req.IsDefault).
		SetMobile(req.Mobile).
		SetName(req.Name).
		SetProvince(req.Province).
		SetCity(req.City).
		SetDistricts(req.Districts).
		SetAddress(req.Address).
		SetPostCode(req.PostCode).
		Save(ctx)
	if err != nil {
		return nil, errors.New(500, "CREATE_ADDRESS_NOT_FOUND", "创建用户地址失败")
	}
	return toDomain(created), nil
}

func (r *adderessRepo) AddressListByUid(ctx context.Context, uid int64) ([]*domain.Address, error) {
	rows, err := r.data.DB(ctx).Address.Query().Where(address.UserIDEQ(uid), address.Deleted(false)).All(ctx)
	if err != nil {
		return nil, errors.New(500, "USER_ADDRESS_LIST_ERROR", "查询用户地址列表失败")
	}
	var list []*domain.Address
	for _, a := range rows {
		list = append(list, toDomain(a))
	}
	return list, nil
}

func (r *adderessRepo) UpdateAddress(ctx context.Context, req *domain.Address) error {
	row, err := r.data.DB(ctx).Address.Query().Where(address.IDEQ(req.ID), address.Deleted(false)).Only(ctx)
	if ent.IsNotFound(err) {
		return errors.NotFound("USER_ADDRESS_NOT_FOUND", "user address not found")
	}
	if err != nil {
		return errors.New(500, "UPDATE_USER_ADDRESS_ERROR", "用户地址更新失败")
	}
	if row.UserID != req.UserID {
		return errors.New(500, "UPDATE_USER_ADDRESS_ERROR", "用户 ID 参数有误")
	}
	if err := r.data.DB(ctx).Address.UpdateOneID(row.ID).
		SetAddress(req.Address).
		SetCity(req.City).
		SetMobile(req.Mobile).
		SetName(req.Name).
		SetIsDefault(req.IsDefault).
		SetPostCode(req.PostCode).
		SetDistricts(req.Districts).
		SetProvince(req.Province).
		Exec(ctx); err != nil {
		return errors.New(500, "UPDATE_USER_ADDRESS_ERROR", "用户地址更新失败")
	}
	return nil
}

func (r *adderessRepo) DefaultAddress(ctx context.Context, req *domain.Address) error {
	return r.data.ExecTx(ctx, func(ctx context.Context) error {
		exists, err := r.data.DB(ctx).Address.Query().
			Where(address.IDEQ(req.ID), address.UserIDEQ(req.UserID), address.Deleted(false)).
			Exist(ctx)
		if err != nil || !exists {
			return errors.NotFound("USER_ADDRESS_NOT_FOUND", "user address not found")
		}
		if err := r.data.DB(ctx).Address.Update().
			Where(address.UserIDEQ(req.UserID), address.IsDefaultEQ(1)).
			SetIsDefault(0).Exec(ctx); err != nil {
			return errors.New(500, "SET_ADDRESS_ERROR", "用户地址修改失败")
		}
		if err := r.data.DB(ctx).Address.Update().
			Where(address.IDEQ(req.ID), address.UserIDEQ(req.UserID), address.Deleted(false)).
			SetIsDefault(1).Exec(ctx); err != nil {
			return errors.New(500, "SET_ADDRESS_ERROR", "用户地址修改失败")
		}
		return nil
	})
}

func (r *adderessRepo) DeleteAddress(ctx context.Context, req *domain.Address) error {
	row, err := r.data.DB(ctx).Address.Query().Where(address.IDEQ(req.ID), address.Deleted(false)).Only(ctx)
	if ent.IsNotFound(err) {
		return errors.NotFound("ADDRESS_NOT_FOUND", "address not found")
	}
	if err != nil {
		return err
	}
	if row.UserID != req.UserID {
		return errors.NotFound("NOT_USER_ADDRESS_FOR_FOUND", "Is not the address of this user")
	}
	if err := r.data.DB(ctx).Address.DeleteOneID(row.ID).Exec(ctx); err != nil {
		return errors.New(500, "DELETE_USER_ADDRESS_ERROR", "用户地址删除失败")
	}
	return nil
}

func (r *adderessRepo) GetAddress(ctx context.Context, req *domain.Address) (*domain.Address, error) {
	row, err := r.data.DB(ctx).Address.Query().Where(address.IDEQ(req.ID), address.UserIDEQ(req.UserID), address.Deleted(false)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, errors.NotFound("USER_ADDRESS_NOT_FOUND", "user address not found")
	}
	if err != nil {
		return nil, err
	}
	return toDomain(row), nil
}
