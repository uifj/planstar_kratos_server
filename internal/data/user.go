package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent"
	"github.com/go-kratos/kratos-layout/internal/data/ent/user"
	"github.com/go-kratos/kratos-layout/internal/data/ent/useraddress"
)

func userToBiz(po *ent.User) *biz.User {
	if po == nil {
		return nil
	}
	return &biz.User{
		ID: idStr(po.ID), Mobile: po.Mobile, Username: po.Username,
		PasswordHash: po.PasswordHash, NickName: po.NickName,
		Birthday: po.Birthday, Gender: po.Gender, Role: po.Role,
		Status: po.Status, CreatedAt: po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
}

func addressToBiz(po *ent.UserAddress) *biz.Address {
	if po == nil {
		return nil
	}
	return &biz.Address{
		ID: idStr(po.ID), UserID: po.UserID, Name: po.Name, Mobile: po.Mobile,
		Province: po.Province, City: po.City, Districts: po.Districts,
		Address: po.Address, PostCode: po.PostCode, IsDefault: po.IsDefault,
		Status: po.Status,
	}
}

type userRepo struct{ data *Data }

func NewUserRepo(d *Data) biz.UserRepo { return &userRepo{data: d} }

func (r *userRepo) Create(ctx context.Context, u *biz.User) (*biz.User, error) {
	po, err := r.data.db.User.Create().
		SetID(parseID(u.ID)).
		SetMobile(u.Mobile).SetUsername(u.Username).
		SetPasswordHash(u.PasswordHash).SetNickName(u.NickName).
		SetBirthday(u.Birthday).SetGender(u.Gender).
		SetRole(u.Role).SetStatus(u.Status).
		Save(ctx)
	if err != nil {
		if isConstraint(err) {
			return nil, biz.ErrUserMobileExists
		}
		return nil, err
	}
	return userToBiz(po), nil
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*biz.User, error) {
	po, err := r.data.db.User.Query().Where(user.IDEQ(parseID(id)), user.StatusEQ(biz.RowActive)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return userToBiz(po), nil
}

func (r *userRepo) FindByMobile(ctx context.Context, mobile string) (*biz.User, error) {
	po, err := r.data.db.User.Query().Where(user.MobileEQ(mobile), user.StatusEQ(biz.RowActive)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return userToBiz(po), nil
}

func (r *userRepo) FindByUsername(ctx context.Context, username string) (*biz.User, error) {
	po, err := r.data.db.User.Query().Where(user.UsernameEQ(username), user.StatusEQ(biz.RowActive)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrUserNotFound
		}
		return nil, err
	}
	return userToBiz(po), nil
}

func (r *userRepo) List(ctx context.Context, page, pageSize int) ([]*biz.User, int32, error) {
	total, err := r.data.db.User.Query().Where(user.StatusEQ(biz.RowActive)).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := r.data.db.User.Query().
		Where(user.StatusEQ(biz.RowActive)).
		Order(user.ByID()).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.User, 0, len(pos))
	for _, po := range pos {
		out = append(out, userToBiz(po))
	}
	return out, int32(total), nil
}

func (r *userRepo) Count(ctx context.Context) (int64, error) {
	return int64(r.data.db.User.Query().Where(user.StatusEQ(biz.RowActive)).CountX(ctx)), nil
}

type addressRepo struct{ data *Data }

func NewAddressRepo(d *Data) biz.AddressRepo { return &addressRepo{data: d} }

func (r *addressRepo) Create(ctx context.Context, a *biz.Address) (*biz.Address, error) {
	po, err := r.data.db.UserAddress.Create().
		SetID(parseID(a.ID)).SetUserID(a.UserID).
		SetName(a.Name).SetMobile(a.Mobile).
		SetProvince(a.Province).SetCity(a.City).SetDistricts(a.Districts).
		SetAddress(a.Address).SetPostCode(a.PostCode).
		SetIsDefault(a.IsDefault).SetStatus(a.Status).Save(ctx)
	if err != nil {
		return nil, err
	}
	return addressToBiz(po), nil
}

func (r *addressRepo) FindByID(ctx context.Context, id string) (*biz.Address, error) {
	po, err := r.data.db.UserAddress.Query().
		Where(useraddress.IDEQ(parseID(id)), useraddress.StatusEQ(biz.RowActive)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrAddressNotFound
		}
		return nil, err
	}
	return addressToBiz(po), nil
}

func (r *addressRepo) ListByUser(ctx context.Context, userID string) ([]*biz.Address, error) {
	pos, err := r.data.db.UserAddress.Query().
		Where(useraddress.UserIDEQ(userID), useraddress.StatusEQ(biz.RowActive)).
		Order(useraddress.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	return addressesToBiz(pos), nil
}

func (r *addressRepo) ListByUserForAdmin(ctx context.Context, userID string) ([]*biz.Address, error) {
	return r.ListByUser(ctx, userID)
}

func addressesToBiz(pos []*ent.UserAddress) []*biz.Address {
	out := make([]*biz.Address, 0, len(pos))
	for _, po := range pos {
		out = append(out, addressToBiz(po))
	}
	return out
}

func (r *addressRepo) Update(ctx context.Context, a *biz.Address) (*biz.Address, error) {
	po, err := r.data.db.UserAddress.UpdateOneID(parseID(a.ID)).
		Where(useraddress.UserIDEQ(a.UserID), useraddress.StatusEQ(biz.RowActive)).
		SetName(a.Name).SetMobile(a.Mobile).
		SetProvince(a.Province).SetCity(a.City).SetDistricts(a.Districts).
		SetAddress(a.Address).SetPostCode(a.PostCode).
		SetIsDefault(a.IsDefault).Save(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrAddressNotFound
		}
		return nil, err
	}
	return addressToBiz(po), nil
}

func (r *addressRepo) SetDefault(ctx context.Context, userID, id string) error {
	_, err := r.data.db.UserAddress.Update().
		Where(useraddress.UserIDEQ(userID), useraddress.StatusEQ(biz.RowActive)).
		SetIsDefault(0).Save(ctx)
	if err != nil {
		return err
	}
	_, err = r.data.db.UserAddress.UpdateOneID(parseID(id)).
		Where(useraddress.UserIDEQ(userID), useraddress.StatusEQ(biz.RowActive)).
		SetIsDefault(1).Save(ctx)
	if err != nil && isNotFound(err) {
		return biz.ErrAddressNotFound
	}
	return err
}

func (r *addressRepo) Delete(ctx context.Context, userID, id string) error {
	_, err := r.data.db.UserAddress.Update().
		Where(useraddress.IDEQ(parseID(id)), useraddress.UserIDEQ(userID), useraddress.StatusEQ(biz.RowActive)).
		SetStatus(biz.RowDeleted).Save(ctx)
	return err
}

func (r *addressRepo) DeleteByIDForAdmin(ctx context.Context, id string) error {
	_, err := r.data.db.UserAddress.Update().
		Where(useraddress.IDEQ(parseID(id)), useraddress.StatusEQ(biz.RowActive)).
		SetStatus(biz.RowDeleted).Save(ctx)
	return err
}
