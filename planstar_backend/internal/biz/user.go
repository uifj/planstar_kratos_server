package biz

import "context"

// AddressInput 新增地址入参（region 兼容单串 / 显式省市区）。
type AddressInput struct {
	Name      string
	Mobile    string
	Region    string
	Detail    string
	Province  string
	City      string
	District  string
	IsDefault bool
}

// UserRepo 是用户/地址仓储端口。
type UserRepo interface {
	Profile(ctx context.Context, userID int64) (*UserProfile, error)
	Addresses(ctx context.Context, userID int64) ([]*ShippingAddress, error)
	AddAddress(ctx context.Context, userID int64, in *AddressInput) (*ShippingAddress, error)
	DeleteAddress(ctx context.Context, userID int64, addressID int64) error
	SetDefaultAddress(ctx context.Context, userID int64, addressID int64) error
}

// UserUsecase 用户用例。
type UserUsecase struct {
	repo UserRepo
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

func (uc *UserUsecase) Profile(ctx context.Context, userID int64) (*UserProfile, error) {
	return uc.repo.Profile(ctx, userID)
}

func (uc *UserUsecase) Addresses(ctx context.Context, userID int64) ([]*ShippingAddress, error) {
	items, err := uc.repo.Addresses(ctx, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*ShippingAddress{}
	}
	return items, nil
}

func (uc *UserUsecase) AddAddress(ctx context.Context, userID int64, in *AddressInput) (*ShippingAddress, error) {
	if in.Name == "" || in.Mobile == "" {
		return nil, ErrBadRequest
	}
	return uc.repo.AddAddress(ctx, userID, in)
}

func (uc *UserUsecase) DeleteAddress(ctx context.Context, userID, addressID int64) error {
	if addressID <= 0 {
		return ErrBadRequest
	}
	return uc.repo.DeleteAddress(ctx, userID, addressID)
}

func (uc *UserUsecase) SetDefaultAddress(ctx context.Context, userID, addressID int64) error {
	if addressID <= 0 {
		return ErrBadRequest
	}
	return uc.repo.SetDefaultAddress(ctx, userID, addressID)
}
