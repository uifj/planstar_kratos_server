package data

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"golang.org/x/crypto/pbkdf2"

	"user/internal/biz"
	"user/internal/data/ent"
	"user/internal/data/ent/user"
)

const (
	pwdSaltLen   = 16
	pwdIteration = 10000
	pwdKeyLen    = 32
)

type userRepo struct {
	data *Data
	log  *slog.Logger
}

// NewUserRepo .
func NewUserRepo(data *Data, logger *slog.Logger) biz.UserRepo {
	return &userRepo{data: data, log: logger}
}

// EncryptPassword uses PBKDF2-SHA512 and returns "$pbkdf2-sha512$<saltHex>$<keyHex>".
func EncryptPassword(psd string) string {
	salt := make([]byte, pwdSaltLen)
	if _, err := rand.Read(salt); err != nil {
		// Fall back to a deterministic salt only when the entropy source fails.
		for i := range salt {
			salt[i] = byte(len(psd) + i)
		}
	}
	dk := pbkdf2.Key([]byte(psd), salt, pwdIteration, pwdKeyLen, sha512.New)
	return "$pbkdf2-sha512$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(dk)
}

// VerifyPassword compares a plaintext password against the stored hash.
func VerifyPassword(psd, encryptedPassword string) (bool, error) {
	parts := strings.Split(encryptedPassword, "$")
	if len(parts) != 4 || parts[1] != "pbkdf2-sha512" || parts[2] == "" || parts[3] == "" {
		return false, errors.New(500, "INVALID_PASSWORD_HASH", "非法的密码哈希格式")
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false, errors.New(500, "INVALID_PASSWORD_HASH", "非法的密码哈希格式")
	}
	want, err := hex.DecodeString(parts[3])
	if err != nil {
		return false, errors.New(500, "INVALID_PASSWORD_HASH", "非法的密码哈希格式")
	}
	dk := pbkdf2.Key([]byte(psd), salt, pwdIteration, len(want), sha512.New)
	return subtle.ConstantTimeCompare(dk, want) == 1, nil
}

func modelToUser(u *ent.User) biz.User {
	return biz.User{
		ID:        u.ID,
		Mobile:    u.Mobile,
		Password:  u.Password,
		NickName:  u.NickName,
		Gender:    u.Gender,
		Role:      u.Role,
		Birthday:  u.Birthday,
		CreatedAt: u.CreatedAt,
	}
}

func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) (*biz.User, error) {
	exists, err := r.data.DB(ctx).User.Query().Where(user.MobileEQ(u.Mobile), user.Deleted(false)).Exist(ctx)
	if err != nil {
		return nil, errors.New(500, "FIND_USER_ERROR", err.Error())
	}
	if exists {
		return nil, errors.New(500, "USER_EXIST", "用户已存在"+u.Mobile)
	}
	created, err := r.data.DB(ctx).User.Create().
		SetMobile(u.Mobile).
		SetPassword(EncryptPassword(u.Password)).
		SetNickName(u.NickName).
		Save(ctx)
	if err != nil {
		return nil, errors.New(500, "CREAT_USER_ERROR", "用户创建失败")
	}
	out := modelToUser(created)
	return &out, nil
}

func (r *userRepo) ListUser(ctx context.Context, pageNum, pageSize int) ([]*biz.User, int, error) {
	q := r.data.DB(ctx).User.Query().Where(user.Deleted(false))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, errors.New(500, "FIND_USER_ERROR", "find user error")
	}
	if pageNum <= 0 {
		pageNum = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	rows, err := q.Offset((pageNum - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, errors.New(500, "FIND_USER_ERROR", "find user error")
	}
	rv := make([]*biz.User, 0, len(rows))
	for _, u := range rows {
		m := modelToUser(u)
		rv = append(rv, &m)
	}
	return rv, total, nil
}

func (r *userRepo) UserByMobile(ctx context.Context, mobile string) (*biz.User, error) {
	u, err := r.data.DB(ctx).User.Query().Where(user.MobileEQ(mobile), user.Deleted(false)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, errors.NotFound("USER_NOT_FOUND", "user not found")
	}
	if err != nil {
		return nil, errors.New(500, "FIND_USER_ERROR", "find user error")
	}
	out := modelToUser(u)
	return &out, nil
}

func (r *userRepo) UpdateUser(ctx context.Context, u *biz.User) (bool, error) {
	row, err := r.data.DB(ctx).User.Query().Where(user.IDEQ(u.ID), user.Deleted(false)).Only(ctx)
	if ent.IsNotFound(err) {
		return false, errors.NotFound("USER_NOT_FOUND", "用户不存在")
	}
	if err != nil {
		return false, errors.New(500, "FIND_USER_ERROR", err.Error())
	}
	if err := r.data.DB(ctx).User.UpdateOneID(row.ID).
		SetNickName(u.NickName).
		SetGender(u.Gender).
		SetNillableBirthday(u.Birthday).
		Exec(ctx); err != nil {
		return false, errors.New(500, "USER_NOT_FOUND", err.Error())
	}
	return true, nil
}

func (r *userRepo) GetUserById(ctx context.Context, id int64) (*biz.User, error) {
	u, err := r.data.DB(ctx).User.Get(ctx, id)
	if ent.IsNotFound(err) {
		return nil, errors.NotFound("USER_NOT_FOUND", "user not found")
	}
	if err != nil {
		return nil, errors.New(500, "USER_NOT_FOUND", err.Error())
	}
	out := modelToUser(u)
	return &out, nil
}

func (r *userRepo) CheckPassword(ctx context.Context, psd, encryptedPassword string) (bool, error) {
	return VerifyPassword(psd, encryptedPassword)
}
