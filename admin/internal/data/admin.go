package data

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-kratos/aip-go/ents"
	"github.com/google/uuid"
	"planstar/admin/internal/biz"
	"planstar/admin/internal/data/ent"
	"planstar/admin/internal/data/ent/admin"
	"planstar/admin/internal/data/ent/predicate"
)

func convertAdmin(po *ent.Admin) *biz.Admin {
	return &biz.Admin{
		ID:        po.ID,
		Name:      po.Name,
		Email:     po.Email,
		Avatar:    po.Avatar,
		Access:    po.Access,
		Password:  po.Password,
		Status:    biz.AdminStatus(po.Status),
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

// notDeleted excludes soft-deleted rows. Every read goes through it, so a
// deleted admin is invisible above `data` without the caller opting in.
func notDeleted() predicate.Admin {
	return admin.StatusNEQ(int32(biz.AdminStatusDeleted))
}

type adminRepo struct {
	data *Data
}

// NewAdminRepo creates a new AdminRepo instance.
func NewAdminRepo(data *Data) biz.AdminRepo {
	return &adminRepo{
		data: data,
	}
}

func (r *adminRepo) FindByID(ctx context.Context, id uuid.UUID) (*biz.Admin, error) {
	po, err := r.data.db.Admin.Query().
		Where(admin.IDEQ(id), notDeleted()).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminNotFound
		}
		return nil, err
	}
	return convertAdmin(po), nil
}

func (r *adminRepo) FindByName(ctx context.Context, name string) (*biz.Admin, error) {
	po, err := r.data.db.Admin.Query().Where(admin.NameEQ(name), notDeleted()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminNotFound
		}
		return nil, err
	}
	return convertAdmin(po), nil
}

func (r *adminRepo) FindByEmail(ctx context.Context, email string) (*biz.Admin, error) {
	po, err := r.data.db.Admin.Query().Where(admin.EmailEQ(email), notDeleted()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminNotFound
		}
		return nil, err
	}
	return convertAdmin(po), nil
}

func (r *adminRepo) ListAdmins(ctx context.Context, opts ...biz.ListOption) ([]*biz.Admin, error) {
	o := biz.ListOptions{Limit: 20}
	for _, opt := range opts {
		opt(&o)
	}
	pos, err := r.data.db.Admin.Query().
		Where(notDeleted(), ents.ApplyFilter(o.Filter)).
		Order(ents.ApplyOrderBy(o.OrderBy)).
		Offset(o.Offset).
		Limit(o.Limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	var admins []*biz.Admin
	for _, po := range pos {
		admins = append(admins, convertAdmin(po))
	}
	return admins, nil
}

func (r *adminRepo) CreateAdmin(ctx context.Context, record *biz.Admin) (*biz.Admin, error) {
	create := r.data.db.Admin.Create().
		SetName(record.Name).
		SetEmail(record.Email).
		SetAvatar(record.Avatar).
		SetAccess(record.Access).
		SetPassword(record.Password)
	// An unspecified status falls back to the schema default (ACTIVE).
	if record.Status != biz.AdminStatusUnspecified {
		create.SetStatus(int32(record.Status))
	}
	po, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return convertAdmin(po), nil
}

func (r *adminRepo) UpdateAdmin(ctx context.Context, record *biz.Admin) (*biz.Admin, error) {
	update := r.data.db.Admin.UpdateOneID(record.ID).
		Where(notDeleted()).
		SetName(record.Name).
		SetEmail(record.Email).
		SetAccess(record.Access).
		SetAvatar(record.Avatar)
	// Only update the password if it's not empty
	if record.Password != "" {
		update.SetPassword(record.Password)
	}
	// An unspecified status leaves the stored one unchanged.
	if record.Status != biz.AdminStatusUnspecified {
		update.SetStatus(int32(record.Status))
	}
	po, err := update.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrAdminNotFound
		}
		return nil, err
	}
	return convertAdmin(po), nil
}

// DeleteAdmin soft-deletes the row: the record stays in storage with a
// DELETED status, which every read in this repo filters out.
func (r *adminRepo) DeleteAdmin(ctx context.Context, id uuid.UUID) error {
	err := r.data.db.Admin.UpdateOneID(id).
		Where(notDeleted()).
		SetStatus(int32(biz.AdminStatusDeleted)).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return biz.ErrAdminNotFound
		}
		return err
	}
	return nil
}

// MustChangePassword reads the first-login flag. The column is not part of the
// ent schema yet, so the query goes through database/sql instead of ent.
func (r *adminRepo) MustChangePassword(ctx context.Context, id uuid.UUID) (bool, error) {
	var mustChange bool
	err := r.data.readDB.QueryRowContext(ctx,
		`SELECT must_change_password FROM admins WHERE id = ? AND status <> ?`,
		id.String(), int32(biz.AdminStatusDeleted),
	).Scan(&mustChange)
	if errors.Is(err, sql.ErrNoRows) {
		return false, biz.ErrAdminNotFound
	}
	if err != nil {
		return false, err
	}
	return mustChange, nil
}

// ClearMustChangePassword is called after a successful password change.
func (r *adminRepo) ClearMustChangePassword(ctx context.Context, id uuid.UUID) error {
	res, err := r.data.readDB.ExecContext(ctx,
		`UPDATE admins SET must_change_password = 0, updated_at = NOW(6) WHERE id = ? AND status <> ?`,
		id.String(), int32(biz.AdminStatusDeleted),
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return biz.ErrAdminNotFound
	}
	return nil
}
