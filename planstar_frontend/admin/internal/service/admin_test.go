package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/transport"
	v1 "planstar/admin/api/kratos/admin/v1"
	"planstar/admin/internal/biz"
	"planstar/admin/pkg/auth"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// fakeAdminRepo is a small in-memory AdminRepo; service tests exercise the
// transport adapter against real usecase wiring.
type fakeAdminRepo struct {
	byID       map[uuid.UUID]*biz.Admin
	byName     map[string]*biz.Admin
	byEmail    map[string]*biz.Admin
	mustChange map[uuid.UUID]bool
}

func newFakeAdminRepo() *fakeAdminRepo {
	return &fakeAdminRepo{
		byID:       make(map[uuid.UUID]*biz.Admin),
		byName:     make(map[string]*biz.Admin),
		byEmail:    make(map[string]*biz.Admin),
		mustChange: make(map[uuid.UUID]bool),
	}
}

func (r *fakeAdminRepo) FindByID(_ context.Context, id uuid.UUID) (*biz.Admin, error) {
	if a, ok := r.byID[id]; ok {
		return a, nil
	}
	return nil, biz.ErrAdminNotFound
}

func (r *fakeAdminRepo) FindByName(_ context.Context, name string) (*biz.Admin, error) {
	if a, ok := r.byName[name]; ok {
		return a, nil
	}
	return nil, biz.ErrAdminNotFound
}

func (r *fakeAdminRepo) FindByEmail(_ context.Context, email string) (*biz.Admin, error) {
	if a, ok := r.byEmail[email]; ok {
		return a, nil
	}
	return nil, biz.ErrAdminNotFound
}

func (r *fakeAdminRepo) ListAdmins(context.Context, ...biz.ListOption) ([]*biz.Admin, error) {
	return nil, nil
}

func (r *fakeAdminRepo) CreateAdmin(_ context.Context, a *biz.Admin) (*biz.Admin, error) {
	r.byID[a.ID] = a
	return a, nil
}

func (r *fakeAdminRepo) UpdateAdmin(_ context.Context, a *biz.Admin) (*biz.Admin, error) {
	r.byID[a.ID] = a
	return a, nil
}

func (r *fakeAdminRepo) DeleteAdmin(context.Context, uuid.UUID) error { return nil }

func (r *fakeAdminRepo) MustChangePassword(_ context.Context, id uuid.UUID) (bool, error) {
	if _, ok := r.byID[id]; !ok {
		return false, biz.ErrAdminNotFound
	}
	return r.mustChange[id], nil
}

func (r *fakeAdminRepo) ClearMustChangePassword(_ context.Context, id uuid.UUID) error {
	r.mustChange[id] = false
	return nil
}

// fakeLimiter records calls so tests can assert the service wires the
// AttemptLimiter contract correctly.
type fakeLimiter struct {
	allowed    bool
	allowedErr error
	failCalls  int
	resetCalls int
	lastKey    string
}

func (f *fakeLimiter) Allowed(_ context.Context, key string) (bool, error) {
	f.lastKey = key
	return f.allowed, f.allowedErr
}

func (f *fakeLimiter) Fail(context.Context, string) error {
	f.failCalls++
	return nil
}

func (f *fakeLimiter) Reset(context.Context, string) error {
	f.resetCalls++
	return nil
}

type fakeRevocation struct {
	revoked    []string
	lastTTL    time.Duration
	revokeErr  error
	revokedSet map[string]bool
}

func (f *fakeRevocation) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}
	f.revoked = append(f.revoked, jti)
	f.lastTTL = ttl
	return nil
}

func (f *fakeRevocation) Revoked(_ context.Context, jti string) (bool, error) {
	return f.revokedSet[jti], nil
}

// fakeTransport satisfies both transport.Transporter and http.Transporter so
// SetCookie / RequestFromServerContext work without a live server.
type fakeTransport struct {
	header  http.Header
	request *http.Request
}

func (t *fakeTransport) Kind() transport.Kind            { return transport.KindHTTP }
func (t *fakeTransport) Endpoint() string                { return "" }
func (t *fakeTransport) Operation() string               { return "" }
func (t *fakeTransport) RequestHeader() transport.Header { return fakeHeader{t.header} }
func (t *fakeTransport) ReplyHeader() transport.Header   { return fakeHeader{t.header} }
func (t *fakeTransport) Request() *http.Request          { return t.request }
func (t *fakeTransport) PathTemplate() string            { return "" }

type fakeHeader struct{ http.Header }

func (h fakeHeader) Keys() []string {
	keys := make([]string, 0, len(h.Header))
	for k := range h.Header {
		keys = append(keys, k)
	}
	return keys
}

func serviceConfig() auth.Config {
	return auth.Config{
		Secret:     "service-test-secret",
		CookieName: "psAdminToken",
		TokenTTL:   time.Hour,
	}
}

func newService(t *testing.T, limiter auth.AttemptLimiter, revoked auth.RevocationStore) (*AdminService, *fakeAdminRepo) {
	t.Helper()
	repo := newFakeAdminRepo()
	hashed, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	admin := &biz.Admin{
		ID:       uuid.New(),
		Name:     "admin",
		Email:    "admin@example.com",
		Password: string(hashed),
		Access:   auth.RoleAdmin,
		Status:   biz.AdminStatusActive,
	}
	repo.byID[admin.ID] = admin
	repo.byName[admin.Name] = admin
	repo.byEmail[admin.Email] = admin
	return NewAdminService(biz.NewAdminUsecase(repo), serviceConfig(), revoked, limiter), repo
}

func requestContext(remoteAddr string) context.Context {
	req, _ := http.NewRequest(http.MethodPost, "/v1/admins/login", nil)
	req.RemoteAddr = remoteAddr
	tr := &fakeTransport{header: make(http.Header), request: req}
	return transport.NewServerContext(context.Background(), tr)
}

func TestLogin_LockedReturns429(t *testing.T) {
	limiter := &fakeLimiter{allowed: false}
	svc, _ := newService(t, limiter, nil)

	_, err := svc.Login(requestContext("203.0.113.7:5555"), &v1.LoginRequest{
		Username: "admin",
		Password: "correct-password",
	})
	if err == nil {
		t.Fatal("expected a lock error")
	}
	if got := kerrors.Code(err); got != http.StatusTooManyRequests {
		t.Fatalf("code = %d, want 429 (%v)", got, err)
	}
	if limiter.lastKey != "u:admin|203.0.113.7" {
		t.Errorf("limit key = %q, want identity + IP", limiter.lastKey)
	}
}

func TestLogin_FailureCountsAttempt(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}
	svc, _ := newService(t, limiter, nil)

	_, err := svc.Login(requestContext("203.0.113.7:5555"), &v1.LoginRequest{
		Username: "admin",
		Password: "wrong-password",
	})
	if err == nil {
		t.Fatal("expected invalid credentials")
	}
	if limiter.failCalls != 1 {
		t.Fatalf("Fail calls = %d, want 1", limiter.failCalls)
	}
	if limiter.resetCalls != 0 {
		t.Fatalf("Reset calls = %d, want 0", limiter.resetCalls)
	}
}

func TestLogin_SuccessResetsAttempts(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}
	svc, _ := newService(t, limiter, nil)

	admin, err := svc.Login(requestContext("203.0.113.7:5555"), &v1.LoginRequest{
		Username: "admin",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if admin == nil || admin.Name != "admin" {
		t.Fatalf("unexpected reply: %#v", admin)
	}
	if limiter.resetCalls != 1 {
		t.Fatalf("Reset calls = %d, want 1", limiter.resetCalls)
	}
}

func TestLogin_ByEmail(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}
	svc, _ := newService(t, limiter, nil)

	admin, err := svc.Login(requestContext("203.0.113.7:5555"), &v1.LoginRequest{
		Email:    "admin@example.com",
		Password: "correct-password",
	})
	if err != nil {
		t.Fatalf("Login by email: %v", err)
	}
	if admin == nil || admin.Email != "admin@example.com" {
		t.Fatalf("unexpected reply: %#v", admin)
	}
	if limiter.lastKey != "e:admin@example.com|203.0.113.7" {
		t.Errorf("limit key = %q, want email identity + IP", limiter.lastKey)
	}
}

func TestLogin_EmptyIdentityReturns400(t *testing.T) {
	limiter := &fakeLimiter{allowed: true}
	svc, _ := newService(t, limiter, nil)

	_, err := svc.Login(requestContext("203.0.113.7:5555"), &v1.LoginRequest{
		Username: "  ",
		Password: "correct-password",
	})
	if err == nil {
		t.Fatal("expected an invalid identity error")
	}
	if got := kerrors.Code(err); got != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400 (%v)", got, err)
	}
	if limiter.lastKey != "" {
		t.Fatalf("limiter consulted for an invalid identity: %q", limiter.lastKey)
	}
}

func TestLogout_RevokesCurrentJTI(t *testing.T) {
	store := &fakeRevocation{}
	svc, repo := newService(t, nil, store)
	var adminID uuid.UUID
	for id := range repo.byID {
		adminID = id
	}
	claims := &auth.Auth{UserID: adminID, Access: auth.RoleAdmin}
	claims.ID = "jti-under-test"
	ctx := auth.NewContext(requestContext("203.0.113.7:5555"), claims)

	if _, err := svc.Logout(ctx, nil); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if len(store.revoked) != 1 || store.revoked[0] != "jti-under-test" {
		t.Fatalf("revoked = %v, want the current jti", store.revoked)
	}
	if store.lastTTL != serviceConfig().TokenTTL {
		t.Errorf("ttl = %v, want %v", store.lastTTL, serviceConfig().TokenTTL)
	}
}

func TestLogout_RevocationFailureIsReported(t *testing.T) {
	store := &fakeRevocation{revokeErr: errors.New("redis down")}
	svc, repo := newService(t, nil, store)
	var adminID uuid.UUID
	for id := range repo.byID {
		adminID = id
	}
	claims := &auth.Auth{UserID: adminID, Access: auth.RoleAdmin}
	claims.ID = "jti-under-test"
	ctx := auth.NewContext(requestContext("203.0.113.7:5555"), claims)

	if _, err := svc.Logout(ctx, nil); err == nil {
		t.Fatal("expected the revocation error to surface")
	}
}

func TestChangePassword_ReissuesCookieAndClearsFlag(t *testing.T) {
	svc, repo := newService(t, nil, nil)
	var adminID uuid.UUID
	for id := range repo.byID {
		adminID = id
	}
	repo.mustChange[adminID] = true
	claims := &auth.Auth{UserID: adminID, Access: auth.RoleAdmin}
	ctx := auth.NewContext(requestContext("203.0.113.7:5555"), claims)

	admin, err := svc.ChangePassword(ctx, &v1.ChangePasswordRequest{
		OldPassword: "correct-password",
		NewPassword: "brand-new-password",
	})
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if admin.MustChangePassword {
		t.Fatal("must_change_password must be false after the change")
	}
	if repo.mustChange[adminID] {
		t.Fatal("repo flag was not cleared")
	}
}
