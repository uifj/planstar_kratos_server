package authn

import (
	"context"
	"crypto/subtle"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// User 是运营账号。PasswordHash 必须是 bcrypt hash，禁止明文或弱哈希。
type User struct {
	Username     string   `yaml:"username" json:"username"`
	PasswordHash string   `yaml:"password_hash" json:"password_hash"`
	UID          string   `yaml:"uid" json:"uid"`
	Roles        []string `yaml:"roles" json:"roles"`
}

// UserStore 抽象账号源，便于未来替换为 MySQL / LDAP / OIDC 而不动中间件。
type UserStore interface {
	Authenticate(ctx context.Context, username, password string) (*User, error)
	All() []User
}

type StaticUserStore struct {
	byName map[string]User
	order  []string
}

// NewStaticUserStore 在启动期校验账号目录：用户名唯一、hash 合法、角色非空。
// 配置错误直接失败关闭，避免运行时静默放行或全部账号不可用。
func NewStaticUserStore(users []User) (*StaticUserStore, error) {
	s := &StaticUserStore{byName: make(map[string]User, len(users))}
	for _, u := range users {
		name := strings.TrimSpace(u.Username)
		if name == "" {
			return nil, fmt.Errorf("运营账号用户名不能为空")
		}
		if _, dup := s.byName[name]; dup {
			return nil, fmt.Errorf("运营账号重复: %q", name)
		}
		if _, err := bcrypt.Cost([]byte(u.PasswordHash)); err != nil {
			return nil, fmt.Errorf("运营账号 %q 的 password_hash 不是合法 bcrypt hash（禁止明文）: %w", name, err)
		}
		if len(u.Roles) == 0 {
			return nil, fmt.Errorf("运营账号 %q 未分配角色（默认拒绝）", name)
		}
		u.Username = name
		s.byName[name] = u
		s.order = append(s.order, name)
	}
	return s, nil
}

// Authenticate 用常量时间方式遍历全部账号，避免用户名枚举时序差异。
func (s *StaticUserStore) Authenticate(_ context.Context, username, password string) (*User, error) {
	var matched *User
	for _, name := range s.order {
		u := s.byName[name]
		if subtle.ConstantTimeCompare([]byte(name), []byte(username)) == 1 {
			copy := u
			matched = &copy
		}
	}
	if matched == nil {
		// 与错误口令走同一条 bcrypt 路径，避免时序区分「用户不存在」与「口令错误」。
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
		return nil, ErrBadCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(matched.PasswordHash), []byte(password)); err != nil {
		return nil, ErrBadCredentials
	}
	return matched, nil
}

func (s *StaticUserStore) All() []User {
	out := make([]User, 0, len(s.order))
	for _, name := range s.order {
		out = append(out, s.byName[name])
	}
	return out
}

// dummyBcryptHash 是固定盐的无效 hash，仅用于常量时间兜底，永不匹配任何口令。
const dummyBcryptHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO5nF7nJ0dX5c8V9kK3uQwYtZrXvLmNbHgS"

// ---------------------------------------------------------------------------
// Identity context：认证阶段写入，授权与审计阶段读取。

type identityContextKey struct{}

func WithIdentity(ctx context.Context, id *Identity) context.Context {
	return context.WithValue(ctx, identityContextKey{}, id)
}

func FromContext(ctx context.Context) *Identity {
	id, _ := ctx.Value(identityContextKey{}).(*Identity)
	return id
}
