package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent"
	"github.com/go-kratos/kratos-layout/internal/data/ent/permission"
	"github.com/go-kratos/kratos-layout/internal/data/ent/rolepermission"
)

func permissionToBiz(po *ent.Permission) *biz.Permission {
	if po == nil {
		return nil
	}
	return &biz.Permission{
		ID: idStr(po.ID), Code: po.Code, Name: po.Name,
		GroupName: po.GroupName, Sort: po.Sort, Status: po.Status,
	}
}

type permissionRepo struct{ data *Data }

func NewPermissionRepo(d *Data) biz.PermissionRepo { return &permissionRepo{data: d} }

func (r *permissionRepo) List(ctx context.Context) ([]*biz.Permission, error) {
	pos, err := r.data.db.Permission.Query().
		Where(permission.StatusEQ(biz.RowActive)).
		Order(permission.BySort(), permission.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.Permission, 0, len(pos))
	for _, po := range pos {
		out = append(out, permissionToBiz(po))
	}
	return out, nil
}

func (r *permissionRepo) ListCodesByRole(ctx context.Context, role int32) ([]string, error) {
	pos, err := r.data.db.RolePermission.Query().
		Where(rolepermission.RoleEQ(role), rolepermission.StatusEQ(biz.RowActive)).
		Order(rolepermission.ByPermissionCode()).All(ctx)
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(pos))
	for _, po := range pos {
		codes = append(codes, po.PermissionCode)
	}
	return codes, nil
}

// ReplaceRole rewrites a role's permission set in one transaction. The unique
// (role, permission_code) index makes concurrent writes safe; duplicates in
// the request are collapsed before insertion.
func (r *permissionRepo) ReplaceRole(ctx context.Context, role int32, codes []string) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.RolePermission.Delete().
		Where(rolepermission.RoleEQ(role)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		if _, err := tx.RolePermission.Create().
			SetRole(role).SetPermissionCode(code).SetStatus(biz.RowActive).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
