package shared

import (
	"reflect"
	"testing"

	bizshared "planstar/internal/biz/shared"
)

func TestScopedWhere(t *testing.T) {
	cases := []struct {
		name      string
		scope     bizshared.DataScope
		ownerCol  string
		ownerID   int64
		wantWhere string
		wantArgs  []any
	}{
		{"self filters by owner", bizshared.ScopeSelf, "user_id", 7, "user_id = ?", []any{int64(7)}},
		{"all no filter", bizshared.ScopeAll, "user_id", 7, "", nil},
		{"unknown no filter (fail-closed at caller)", bizshared.DataScope(""), "user_id", 7, "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			where, args := ScopedWhere(tc.scope, tc.ownerCol, tc.ownerID)
			if where != tc.wantWhere {
				t.Fatalf("where = %q, want %q", where, tc.wantWhere)
			}
			if !reflect.DeepEqual(args, tc.wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, tc.wantArgs)
			}
		})
	}
}
