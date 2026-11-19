package service

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	httpm "github.com/go-kratos/kratos/v3/transport/http"
	v1 "planstar/admin/api/kratos/admin/v1"
	"planstar/admin/internal/biz"
	"planstar/admin/pkg/auth"

	"github.com/google/uuid"
	"go.einride.tech/aip/fieldmask"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.einride.tech/aip/pagination"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func convertAdmin(m *biz.Admin) *v1.Admin {
	return &v1.Admin{
		Id:                 m.ID.String(),
		Name:               m.Name,
		Email:              m.Email,
		Avatar:             m.Avatar,
		Access:             m.Access,
		Status:             v1.Admin_Status(m.Status),
		CreatedAt:          timestamppb.New(m.CreatedAt),
		UpdatedAt:          timestamppb.New(m.UpdatedAt),
		MustChangePassword: m.MustChangePassword,
	}
}

// parseAdminID turns a wire ID into a UUID, reporting a bad one as a typed
// error so callers see a 400 instead of a not-found.
func parseAdminID(id string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, biz.ErrInvalidAdminID
	}
	return parsed, nil
}

// AdminService is a greeter service.
type AdminService struct {
	v1.UnimplementedAdminServiceServer

	uc      *biz.AdminUsecase
	cfg     auth.Config
	revoked auth.RevocationStore
	limiter auth.AttemptLimiter
}

// NewAdminService new a greeter service.
func NewAdminService(uc *biz.AdminUsecase, cfg auth.Config, revoked auth.RevocationStore, limiter auth.AttemptLimiter) *AdminService {
	return &AdminService{uc: uc, cfg: cfg, revoked: revoked, limiter: limiter}
}

// Current implements auth current admin retrieval.
func (s *AdminService) Current(ctx context.Context, req *emptypb.Empty) (*v1.Admin, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	admin, err := s.uc.GetAdmin(ctx, a.UserID)
	if err != nil {
		return nil, err
	}
	return convertAdmin(admin), nil
}

// Login implements auth login.
func (s *AdminService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.Admin, error) {
	var (
		identity     string
		authenticate func() (*biz.Admin, error)
	)
	switch username, email := strings.TrimSpace(req.Username), strings.TrimSpace(req.Email); {
	case username != "":
		identity = "u:" + username
		authenticate = func() (*biz.Admin, error) { return s.uc.LoginByUsername(ctx, username, req.Password) }
	case email != "":
		identity = "e:" + email
		authenticate = func() (*biz.Admin, error) { return s.uc.LoginByEmail(ctx, email, req.Password) }
	default:
		return nil, errors.BadRequest(v1.ErrorReason_INVALID_IDENTITY_TYPE.String(), "unsupported identity type")
	}
	// Check the lock before hashing the password so a locked account does not
	// burn bcrypt CPU on every rejected attempt.
	limitKey := loginLimitKey(ctx, identity)
	if !s.loginAllowed(ctx, limitKey) {
		return nil, errors.TooManyRequests(v1.ErrorReason_TOO_MANY_ATTEMPTS.String(), "too many login attempts, try again later")
	}
	admin, err := authenticate()
	if err != nil {
		if errors.Is(err, biz.ErrInvalidCredentials) {
			s.loginFailed(ctx, limitKey)
		}
		return nil, err
	}
	s.loginSucceeded(ctx, limitKey)
	if err := auth.SetCookie(ctx, s.cfg, admin.ID, admin.Access, admin.MustChangePassword, time.Now().Add(s.cfg.TokenLifetime())); err != nil {
		return nil, err
	}
	return convertAdmin(admin), nil
}

// loginAllowed consults the limiter. A limiter outage is logged and treated as
// "allowed": login availability matters more than throttling during a Redis
// incident, and the password check still runs.
func (s *AdminService) loginAllowed(ctx context.Context, key string) bool {
	if s.limiter == nil {
		return true
	}
	allowed, err := s.limiter.Allowed(ctx, key)
	if err != nil {
		log.WarnContext(ctx, "login limiter unavailable", "error", err)
		return true
	}
	return allowed
}

func (s *AdminService) loginFailed(ctx context.Context, key string) {
	if s.limiter == nil {
		return
	}
	if err := s.limiter.Fail(ctx, key); err != nil {
		log.WarnContext(ctx, "record login failure", "error", err)
	}
}

func (s *AdminService) loginSucceeded(ctx context.Context, key string) {
	if s.limiter == nil {
		return
	}
	if err := s.limiter.Reset(ctx, key); err != nil {
		log.WarnContext(ctx, "reset login failures", "error", err)
	}
}

// loginLimitKey scopes failed attempts to identity + client IP so one noisy
// client cannot lock out an account for everybody.
func loginLimitKey(ctx context.Context, identity string) string {
	req, ok := httpm.RequestFromServerContext(ctx)
	if !ok {
		return identity
	}
	ip := req.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	if ip == "" {
		return identity
	}
	return identity + "|" + ip
}

// Logout implements auth logout.
func (s *AdminService) Logout(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if err := s.uc.Logout(ctx, a.UserID); err != nil {
		return nil, err
	}
	if s.revoked != nil && a.ID != "" {
		ttl := s.cfg.TokenLifetime()
		if a.ExpiresAt != nil {
			if remaining := time.Until(a.ExpiresAt.Time); remaining > 0 {
				ttl = remaining
			}
		}
		if err := s.revoked.Revoke(ctx, a.ID, ttl); err != nil {
			return nil, err
		}
	}
	if err := auth.DeleteCookie(ctx, s.cfg); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ChangePassword replaces the current admin's password and re-issues the
// cookie without the PwdChange marker.
func (s *AdminService) ChangePassword(ctx context.Context, req *v1.ChangePasswordRequest) (*v1.Admin, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	admin, err := s.uc.ChangePassword(ctx, a.UserID, req.OldPassword, req.NewPassword)
	if err != nil {
		return nil, err
	}
	if err := auth.SetCookie(ctx, s.cfg, admin.ID, admin.Access, false, time.Now().Add(s.cfg.TokenLifetime())); err != nil {
		return nil, err
	}
	return convertAdmin(admin), nil
}

// CreateAdmin implements admin creation.
func (s *AdminService) CreateAdmin(ctx context.Context, req *v1.CreateAdminRequest) (*v1.Admin, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !a.HasAdminAccess() {
		return nil, auth.ErrPermissionDenied
	}
	admin, err := s.uc.CreateAdmin(ctx, &biz.Admin{
		Name:     req.Admin.Name,
		Email:    req.Admin.Email,
		Password: req.Admin.Password,
		Avatar:   req.Admin.Avatar,
		Access:   req.Admin.Access,
		Status:   biz.AdminStatus(req.Admin.Status),
	})
	if err != nil {
		return nil, err
	}
	return convertAdmin(admin), nil
}

// UpdateAdmin implements admin update.
func (s *AdminService) UpdateAdmin(ctx context.Context, req *v1.UpdateAdminRequest) (*v1.Admin, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !a.HasAdminAccess() {
		return nil, auth.ErrPermissionDenied
	}
	admin, err := s.GetAdmin(ctx, &v1.GetAdminRequest{Id: req.Admin.Id})
	if err != nil {
		return nil, err
	}
	fieldmask.Update(req.UpdateMask, admin, req.Admin)
	id, err := parseAdminID(admin.Id)
	if err != nil {
		return nil, err
	}
	updated, err := s.uc.UpdateAdmin(ctx, &biz.Admin{
		ID:       id,
		Name:     admin.Name,
		Email:    admin.Email,
		Password: admin.Password,
		Avatar:   admin.Avatar,
		Access:   admin.Access,
		Status:   biz.AdminStatus(admin.Status),
	})
	if err != nil {
		return nil, err
	}
	return convertAdmin(updated), nil
}

// DeleteAdmin implements admin deletion.
func (s *AdminService) DeleteAdmin(ctx context.Context, req *v1.DeleteAdminRequest) (*emptypb.Empty, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !a.HasAdminAccess() {
		return nil, auth.ErrPermissionDenied
	}
	id, err := parseAdminID(req.Id)
	if err != nil {
		return nil, err
	}
	if err := s.uc.DeleteAdmin(ctx, id); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// GetAdmin implements admin retrieval.
func (s *AdminService) GetAdmin(ctx context.Context, req *v1.GetAdminRequest) (*v1.Admin, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !a.HasAdminAccess() {
		return nil, auth.ErrPermissionDenied
	}
	id, err := parseAdminID(req.Id)
	if err != nil {
		return nil, err
	}
	admin, err := s.uc.GetAdmin(ctx, id)
	if err != nil {
		return nil, err
	}
	return convertAdmin(admin), nil
}

// ListAdmins implements admin listing with filtering, ordering, and pagination.
func (s *AdminService) ListAdmins(ctx context.Context, req *v1.ListAdminsRequest) (*v1.AdminSet, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !a.HasAdminAccess() {
		return nil, auth.ErrPermissionDenied
	}
	declarations, err := filtering.NewDeclarations(
		filtering.DeclareStandardFunctions(),
		filtering.DeclareIdent("name", filtering.TypeString),
		filtering.DeclareIdent("email", filtering.TypeString),
		filtering.DeclareIdent("created_at", filtering.TypeTimestamp),
	)
	if err != nil {
		return nil, err
	}
	filter, err := filtering.ParseFilter(req, declarations)
	if err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	orderBy, err := ordering.ParseOrderBy(req)
	if err != nil {
		return nil, err
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	admins, err := s.uc.ListAdmins(ctx,
		biz.ListFilter(filter),
		biz.ListOrderBy(orderBy),
		biz.ListLimit(int(req.PageSize)),
		biz.ListOffset(int(pageToken.Offset)),
	)
	if err != nil {
		return nil, err
	}
	adminSet := &v1.AdminSet{
		Admins: make([]*v1.Admin, 0, len(admins)),
	}
	if len(admins) >= int(req.PageSize) {
		adminSet.NextPageToken = pageToken.Next(req).String()
	}
	for _, admin := range admins {
		adminSet.Admins = append(adminSet.Admins, convertAdmin(admin))
	}
	return adminSet, nil
}
