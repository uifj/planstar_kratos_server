package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// AdminStatus is the lifecycle status of an admin. The values match the
// kratos.admin.v1.Admin_Status enum numbers, so `service` and `data` convert
// with a cast instead of a lookup table.
type AdminStatus int32

const (
	// AdminStatusUnspecified means the caller did not state a status: create
	// stores AdminStatusActive, update leaves the stored status unchanged.
	AdminStatusUnspecified AdminStatus = 0
	// AdminStatusActive means the admin can log in.
	AdminStatusActive AdminStatus = 1
	// AdminStatusInactive means the admin exists but cannot log in.
	AdminStatusInactive AdminStatus = 2
	// AdminStatusDeleted marks a soft-deleted admin; reads exclude it.
	AdminStatusDeleted AdminStatus = 3
)

// Admin is a Admin model.
type Admin struct {
	ID       uuid.UUID
	Name     string
	Email    string
	Password string
	Access   string
	Avatar   string
	Status   AdminStatus
	// MustChangePassword is the first-login gate; it is stored in a column the
	// ent schema does not model yet, so the usecase loads it explicitly.
	MustChangePassword bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// AdminRepo is a Greater repo.
type AdminRepo interface {
	FindByID(context.Context, uuid.UUID) (*Admin, error)
	FindByName(context.Context, string) (*Admin, error)
	FindByEmail(context.Context, string) (*Admin, error)
	ListAdmins(context.Context, ...ListOption) ([]*Admin, error)
	CreateAdmin(context.Context, *Admin) (*Admin, error)
	UpdateAdmin(context.Context, *Admin) (*Admin, error)
	// DeleteAdmin soft-deletes an admin by setting its status to
	// AdminStatusDeleted.
	DeleteAdmin(context.Context, uuid.UUID) error
	// MustChangePassword reports the first-login flag stored outside the ent
	// schema.
	MustChangePassword(context.Context, uuid.UUID) (bool, error)
	// ClearMustChangePassword clears the first-login flag after a successful
	// password change.
	ClearMustChangePassword(context.Context, uuid.UUID) error
}

// AdminUsecase is a Admin usecase.
type AdminUsecase struct {
	admin AdminRepo
}

// NewAdminUsecase new a Admin usecase.
func NewAdminUsecase(repo AdminRepo) *AdminUsecase {
	return &AdminUsecase{admin: repo}
}

// LoginByUsername logs in a user by username and password.
func (uc *AdminUsecase) LoginByUsername(ctx context.Context, username, password string) (*Admin, error) {
	user, err := uc.admin.FindByName(ctx, username)
	if err != nil {
		if errors.Is(err, ErrAdminNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := uc.loadPasswordState(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// LoginByEmail logs in a user by email and password.
func (uc *AdminUsecase) LoginByEmail(ctx context.Context, email, password string) (*Admin, error) {
	user, err := uc.admin.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrAdminNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	if err := uc.loadPasswordState(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Logout logs out the current user.
func (uc *AdminUsecase) Logout(ctx context.Context, adminID uuid.UUID) error {
	admin, err := uc.admin.FindByID(ctx, adminID)
	if err != nil {
		return err
	}
	log.InfoContext(ctx, "admin logged out", "name", admin.Name)
	return nil
}

// GetAdmin returns the current logged in user.
func (uc *AdminUsecase) GetAdmin(ctx context.Context, id uuid.UUID) (*Admin, error) {
	admin, err := uc.admin.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := uc.loadPasswordState(ctx, admin); err != nil {
		return nil, err
	}
	return admin, nil
}

// ListAdmins lists admin users with pagination.
func (uc *AdminUsecase) ListAdmins(ctx context.Context, opts ...ListOption) ([]*Admin, error) {
	admins, err := uc.admin.ListAdmins(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return admins, nil
}

// CreateAdmin creates a new admin user.
func (uc *AdminUsecase) CreateAdmin(ctx context.Context, admin *Admin) (*Admin, error) {
	if admin.Password != "" {
		hashed, err := hashPassword(admin.Password)
		if err != nil {
			return nil, err
		}
		admin.Password = hashed
	}
	return uc.admin.CreateAdmin(ctx, admin)
}

// UpdateAdmin updates an existing admin user.
func (uc *AdminUsecase) UpdateAdmin(ctx context.Context, admin *Admin) (*Admin, error) {
	// Empty password means "leave unchanged"; only hash when a new one is set.
	if admin.Password != "" {
		hashed, err := hashPassword(admin.Password)
		if err != nil {
			return nil, err
		}
		admin.Password = hashed
	}
	updated, err := uc.admin.UpdateAdmin(ctx, admin)
	if err != nil {
		return nil, err
	}
	if err := uc.loadPasswordState(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

// ChangePassword verifies the current password, enforces the minimum length,
// stores a bcrypt cost-12 hash and clears the first-login flag.
func (uc *AdminUsecase) ChangePassword(ctx context.Context, id uuid.UUID, oldPassword, newPassword string) (*Admin, error) {
	if len([]rune(newPassword)) < minPasswordLength {
		return nil, ErrWeakPassword
	}
	admin, err := uc.admin.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(oldPassword)); err != nil {
		return nil, ErrInvalidOldPassword
	}
	hashed, err := hashPassword(newPassword)
	if err != nil {
		return nil, err
	}
	admin.Password = hashed
	updated, err := uc.admin.UpdateAdmin(ctx, admin)
	if err != nil {
		return nil, err
	}
	if err := uc.admin.ClearMustChangePassword(ctx, id); err != nil {
		return nil, err
	}
	updated.MustChangePassword = false
	return updated, nil
}

const (
	// bcryptCost is deliberately above the library default (10): admin
	// passwords guard the whole console, so hashing cost matters more than the
	// few extra milliseconds at login.
	bcryptCost        = 12
	minPasswordLength = 8
)

// hashPassword hashes a plaintext password using bcrypt.
func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// loadPasswordState fills the first-login flag from the repo. The flag lives in
// a column the ent schema does not model, so it cannot ride along with
// convertAdmin.
func (uc *AdminUsecase) loadPasswordState(ctx context.Context, admin *Admin) error {
	mustChange, err := uc.admin.MustChangePassword(ctx, admin.ID)
	if err != nil {
		return err
	}
	admin.MustChangePassword = mustChange
	return nil
}

// DeleteAdmin soft-deletes an admin user by ID.
func (uc *AdminUsecase) DeleteAdmin(ctx context.Context, id uuid.UUID) error {
	return uc.admin.DeleteAdmin(ctx, id)
}
