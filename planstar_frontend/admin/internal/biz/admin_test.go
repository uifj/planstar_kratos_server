package biz

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// fakeAdminRepo is an in-memory AdminRepo for testing the usecase in
// isolation from the data layer.
type fakeAdminRepo struct {
	byID    map[uuid.UUID]*Admin
	byName  map[string]*Admin
	byEmail map[string]*Admin
	// mustChange mirrors the out-of-schema column.
	mustChange map[uuid.UUID]bool
	created    []*Admin
}

func newFakeAdminRepo() *fakeAdminRepo {
	return &fakeAdminRepo{
		byID:       make(map[uuid.UUID]*Admin),
		byName:     make(map[string]*Admin),
		byEmail:    make(map[string]*Admin),
		mustChange: make(map[uuid.UUID]bool),
	}
}

func (r *fakeAdminRepo) FindByID(_ context.Context, id uuid.UUID) (*Admin, error) {
	a, ok := r.byID[id]
	if !ok {
		return nil, ErrAdminNotFound
	}
	return a, nil
}

func (r *fakeAdminRepo) FindByName(_ context.Context, name string) (*Admin, error) {
	a, ok := r.byName[name]
	if !ok {
		return nil, ErrAdminNotFound
	}
	return a, nil
}

func (r *fakeAdminRepo) FindByEmail(_ context.Context, email string) (*Admin, error) {
	a, ok := r.byEmail[email]
	if !ok {
		return nil, ErrAdminNotFound
	}
	return a, nil
}

func (r *fakeAdminRepo) ListAdmins(context.Context, ...ListOption) ([]*Admin, error) {
	return nil, nil
}

func (r *fakeAdminRepo) CreateAdmin(_ context.Context, a *Admin) (*Admin, error) {
	r.created = append(r.created, a)
	return a, nil
}

func (r *fakeAdminRepo) UpdateAdmin(_ context.Context, a *Admin) (*Admin, error) {
	// Mirror the data repo: an empty password leaves the stored hash intact.
	if a.Password == "" {
		if old, ok := r.byID[a.ID]; ok {
			a.Password = old.Password
		}
	}
	r.byID[a.ID] = a
	return a, nil
}

func (r *fakeAdminRepo) DeleteAdmin(context.Context, uuid.UUID) error {
	return nil
}

func (r *fakeAdminRepo) MustChangePassword(_ context.Context, id uuid.UUID) (bool, error) {
	if _, ok := r.byID[id]; !ok {
		return false, ErrAdminNotFound
	}
	return r.mustChange[id], nil
}

func (r *fakeAdminRepo) ClearMustChangePassword(_ context.Context, id uuid.UUID) error {
	if _, ok := r.byID[id]; !ok {
		return ErrAdminNotFound
	}
	r.mustChange[id] = false
	return nil
}

// seedAdmin inserts an admin with a bcrypt-hashed password.
func seedAdmin(t *testing.T, repo *fakeAdminRepo, name, email, password string) *Admin {
	t.Helper()
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	a := &Admin{ID: uuid.New(), Name: name, Email: email, Password: string(hashed)}
	repo.byID[a.ID] = a
	repo.byName[name] = a
	repo.byEmail[email] = a
	return a
}

func TestLoginByUsername_Success(t *testing.T) {
	repo := newFakeAdminRepo()
	seedAdmin(t, repo, "admin", "admin@example.com", "admin")
	uc := NewAdminUsecase(repo)

	got, err := uc.LoginByUsername(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("expected login success, got error: %v", err)
	}
	if got.Name != "admin" {
		t.Fatalf("expected admin user, got %q", got.Name)
	}
}

func TestLoginByEmail_Success(t *testing.T) {
	repo := newFakeAdminRepo()
	seedAdmin(t, repo, "admin", "admin@example.com", "admin")
	uc := NewAdminUsecase(repo)

	got, err := uc.LoginByEmail(context.Background(), "admin@example.com", "admin")
	if err != nil {
		t.Fatalf("expected login success, got error: %v", err)
	}
	if got.Email != "admin@example.com" {
		t.Fatalf("expected admin user, got %q", got.Email)
	}
}

func TestLoginByUsername_WrongPassword(t *testing.T) {
	repo := newFakeAdminRepo()
	seedAdmin(t, repo, "admin", "admin@example.com", "admin")
	uc := NewAdminUsecase(repo)

	_, err := uc.LoginByUsername(context.Background(), "admin", "wrong")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
	if errors.Code(err) != 401 {
		t.Fatalf("expected 401 unauthorized, got code %d (%v)", errors.Code(err), err)
	}
}

func TestLoginByUsername_UserNotFound(t *testing.T) {
	repo := newFakeAdminRepo()
	uc := NewAdminUsecase(repo)

	_, err := uc.LoginByUsername(context.Background(), "ghost", "admin")
	if err == nil {
		t.Fatal("expected error for unknown user, got nil")
	}
	// Must be the same opaque 401 as a wrong password, to avoid user enumeration.
	if errors.Code(err) != 401 {
		t.Fatalf("expected 401 unauthorized, got code %d (%v)", errors.Code(err), err)
	}
	if errors.Is(err, ErrAdminNotFound) {
		t.Fatal("login must not leak ErrAdminNotFound for unknown users")
	}
}

func TestCreateAdmin_HashesPassword(t *testing.T) {
	repo := newFakeAdminRepo()
	uc := NewAdminUsecase(repo)

	_, err := uc.CreateAdmin(context.Background(), &Admin{Name: "bob", Password: "secret"})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected 1 created admin, got %d", len(repo.created))
	}
	stored := repo.created[0].Password
	if stored == "secret" {
		t.Fatal("password was stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte("secret")); err != nil {
		t.Fatalf("stored password is not a valid bcrypt hash of the input: %v", err)
	}
}

func TestUpdateAdmin_EmptyPasswordLeftUnchanged(t *testing.T) {
	repo := newFakeAdminRepo()
	existing := seedAdmin(t, repo, "bob", "bob@example.com", "secret")
	uc := NewAdminUsecase(repo)

	got, err := uc.UpdateAdmin(context.Background(), &Admin{ID: existing.ID, Name: "bob", Password: ""})
	if err != nil {
		t.Fatalf("update admin: %v", err)
	}
	if got.Password == "" {
		t.Fatal("empty password must leave the stored hash unchanged")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(got.Password), []byte("secret")); err != nil {
		t.Fatalf("stored hash no longer matches the original password: %v", err)
	}
}

func TestCreateAdmin_UsesBcryptCost12(t *testing.T) {
	repo := newFakeAdminRepo()
	uc := NewAdminUsecase(repo)

	created, err := uc.CreateAdmin(context.Background(), &Admin{Name: "bob", Password: "secret"})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(created.Password))
	if err != nil {
		t.Fatalf("bcrypt.Cost: %v", err)
	}
	if cost != 12 {
		t.Fatalf("bcrypt cost = %d, want 12", cost)
	}
}

func TestChangePassword_Success(t *testing.T) {
	repo := newFakeAdminRepo()
	existing := seedAdmin(t, repo, "admin", "admin@example.com", "old-password")
	repo.mustChange[existing.ID] = true
	uc := NewAdminUsecase(repo)

	got, err := uc.ChangePassword(context.Background(), existing.ID, "old-password", "new-password")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if got.MustChangePassword {
		t.Fatal("must_change_password must be cleared after a successful change")
	}
	if repo.mustChange[existing.ID] {
		t.Fatal("repo flag was not cleared")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(got.Password), []byte("new-password")); err != nil {
		t.Fatalf("new password was not stored: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(got.Password))
	if err != nil {
		t.Fatalf("bcrypt.Cost: %v", err)
	}
	if cost != 12 {
		t.Fatalf("bcrypt cost = %d, want 12", cost)
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	repo := newFakeAdminRepo()
	existing := seedAdmin(t, repo, "admin", "admin@example.com", "old-password")
	uc := NewAdminUsecase(repo)

	_, err := uc.ChangePassword(context.Background(), existing.ID, "wrong", "new-password")
	if err == nil {
		t.Fatal("expected an error for a wrong current password")
	}
	if errors.Code(err) != 400 {
		t.Fatalf("expected 400, got code %d (%v)", errors.Code(err), err)
	}
}

func TestChangePassword_RejectsShortPassword(t *testing.T) {
	repo := newFakeAdminRepo()
	existing := seedAdmin(t, repo, "admin", "admin@example.com", "old-password")
	uc := NewAdminUsecase(repo)

	_, err := uc.ChangePassword(context.Background(), existing.ID, "old-password", "short")
	if err == nil {
		t.Fatal("expected an error for a short new password")
	}
	if errors.Code(err) != 400 {
		t.Fatalf("expected 400, got code %d (%v)", errors.Code(err), err)
	}
}
