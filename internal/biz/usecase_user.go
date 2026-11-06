package biz

import (
	"context"
	"errors"
	"strings"
)

// UserUsecase owns authentication and user administration. It depends only on
// pure-Go ports, so it can be unit tested with fakes.
type UserUsecase struct {
	repo    UserRepo
	perms   PermissionRepo
	hasher  PasswordHasher
	tokens  TokenManager
	captcha CaptchaStore
}

func NewUserUsecase(repo UserRepo, perms PermissionRepo, hasher PasswordHasher, tokens TokenManager, captcha CaptchaStore) *UserUsecase {
	return &UserUsecase{repo: repo, perms: perms, hasher: hasher, tokens: tokens, captcha: captcha}
}

// Captcha issues a new captcha challenge stored in Redis.
func (uc *UserUsecase) Captcha(ctx context.Context) (string, string, error) {
	return uc.captcha.Create(ctx)
}

// Register creates a user. role is AuthorityUser for the shop BFF and
// AuthorityAdmin for the admin BFF.
func (uc *UserUsecase) Register(ctx context.Context, mobile, username, password string, role int32) (*User, string, int64, error) {
	if !validMobile(mobile) || strings.TrimSpace(username) == "" || len(password) < 8 {
		return nil, "", 0, ErrUserInvalidArgument
	}
	if _, err := uc.repo.FindByMobile(ctx, mobile); err == nil {
		return nil, "", 0, ErrUserMobileExists
	} else if !errors.Is(err, ErrUserNotFound) {
		return nil, "", 0, err
	}
	if strings.TrimSpace(username) != "" {
		if _, err := uc.repo.FindByUsername(ctx, username); err == nil {
			return nil, "", 0, ErrUserInvalidArgument
		} else if !errors.Is(err, ErrUserNotFound) {
			return nil, "", 0, err
		}
	}
	hash, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, "", 0, err
	}
	u, err := uc.repo.Create(ctx, &User{
		ID: newID(), Mobile: mobile, Username: username,
		PasswordHash: hash, NickName: username, Role: role, Status: RowActive,
	})
	if err != nil {
		return nil, "", 0, err
	}
	token, exp, err := uc.tokens.Generate(ctx, u.ID, u.NickName, u.Role)
	if err != nil {
		return nil, "", 0, err
	}
	return u, token, exp, nil
}

// Login authenticates by mobile (C-end) or username (admin). When
// verifyCaptcha is true the captcha must also match; this is the C-end path.
func (uc *UserUsecase) Login(ctx context.Context, mobile, username, password, captcha, captchaID string, verifyCaptcha bool) (*User, string, int64, error) {
	if password == "" {
		return nil, "", 0, ErrUserInvalidArgument
	}
	var (
		u   *User
		err error
	)
	if mobile != "" {
		u, err = uc.repo.FindByMobile(ctx, mobile)
	} else if username != "" {
		u, err = uc.repo.FindByUsername(ctx, username)
	} else {
		return nil, "", 0, ErrUserInvalidArgument
	}
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, "", 0, ErrUserPasswordInvalid
		}
		return nil, "", 0, err
	}
	if !uc.hasher.Verify(u.PasswordHash, password) {
		return nil, "", 0, ErrUserPasswordInvalid
	}
	if verifyCaptcha {
		ok, err := uc.captcha.Verify(ctx, captchaID, captcha)
		if err != nil {
			return nil, "", 0, err
		}
		if !ok {
			return nil, "", 0, ErrCaptchaInvalid
		}
	}
	token, exp, err := uc.tokens.Generate(ctx, u.ID, u.NickName, u.Role)
	if err != nil {
		return nil, "", 0, err
	}
	return u, token, exp, nil
}

// RefreshToken validates the full claim set (including expiry) and issues a
// fresh token. The source project used WithoutClaimsValidation, which allowed
// expired tokens to be renewed forever; this version does not.
func (uc *UserUsecase) RefreshToken(ctx context.Context, token string) (*User, string, int64, error) {
	claims, err := uc.tokens.Parse(ctx, token)
	if err != nil || claims == nil || claims.ID == "" {
		return nil, "", 0, ErrTokenInvalid
	}
	u, err := uc.repo.FindByID(ctx, claims.ID)
	if err != nil {
		return nil, "", 0, err
	}
	newToken, exp, err := uc.tokens.Generate(ctx, u.ID, u.NickName, u.Role)
	if err != nil {
		return nil, "", 0, err
	}
	return u, newToken, exp, nil
}

// Detail returns the caller's profile plus the permission codes of its role.
func (uc *UserUsecase) Detail(ctx context.Context, userID string) (*User, []string, error) {
	u, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	codes, err := uc.perms.ListCodesByRole(ctx, u.Role)
	if err != nil {
		return nil, nil, err
	}
	return u, codes, nil
}

func (uc *UserUsecase) List(ctx context.Context, page, pageSize int) ([]*User, int32, error) {
	page, pageSize = pageBounds(page, pageSize)
	return uc.repo.List(ctx, page, pageSize)
}

// AddressUsecase owns the address book. userID always comes from JWT claims.
type AddressUsecase struct {
	repo AddressRepo
}

func NewAddressUsecase(repo AddressRepo) *AddressUsecase {
	return &AddressUsecase{repo: repo}
}

func (uc *AddressUsecase) Create(ctx context.Context, userID string, a *Address) (*Address, error) {
	if a == nil || strings.TrimSpace(a.Name) == "" || !validMobile(a.Mobile) {
		return nil, ErrUserInvalidArgument
	}
	a.ID = newID()
	a.UserID = userID
	a.Status = RowActive
	created, err := uc.repo.Create(ctx, a)
	if err != nil {
		return nil, err
	}
	if a.IsDefault == 1 {
		if err := uc.repo.SetDefault(ctx, userID, created.ID); err != nil {
			return nil, err
		}
		created.IsDefault = 1
	}
	return created, nil
}

func (uc *AddressUsecase) List(ctx context.Context, userID string) ([]*Address, error) {
	return uc.repo.ListByUser(ctx, userID)
}

func (uc *AddressUsecase) Update(ctx context.Context, userID string, a *Address) (*Address, error) {
	if a == nil || a.ID == "" {
		return nil, ErrUserInvalidArgument
	}
	a.UserID = userID
	updated, err := uc.repo.Update(ctx, a)
	if err != nil {
		return nil, err
	}
	if a.IsDefault == 1 {
		if err := uc.repo.SetDefault(ctx, userID, a.ID); err != nil {
			return nil, err
		}
		updated.IsDefault = 1
	}
	return updated, nil
}

func (uc *AddressUsecase) SetDefault(ctx context.Context, userID, id string) error {
	if id == "" {
		return ErrUserInvalidArgument
	}
	return uc.repo.SetDefault(ctx, userID, id)
}

func (uc *AddressUsecase) Delete(ctx context.Context, userID, id string) error {
	if id == "" {
		return ErrUserInvalidArgument
	}
	return uc.repo.Delete(ctx, userID, id)
}

func (uc *AddressUsecase) AdminList(ctx context.Context, uid string) ([]*Address, error) {
	return uc.repo.ListByUserForAdmin(ctx, uid)
}

func (uc *AddressUsecase) AdminDelete(ctx context.Context, id string) error {
	if id == "" {
		return ErrUserInvalidArgument
	}
	return uc.repo.DeleteByIDForAdmin(ctx, id)
}
