package data

import (
	"github.com/go-kratos/kratos-layout/internal/data/ent"

	"github.com/google/uuid"
)

func parseID(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func idStr(id uuid.UUID) string { return id.String() }

func isNotFound(err error) bool { return ent.IsNotFound(err) }

func isConstraint(err error) bool { return ent.IsConstraintError(err) }
