package shared

import "testing"

func TestDataScopeValid(t *testing.T) {
	if !ScopeSelf.Valid() || !ScopeAll.Valid() {
		t.Fatal("self/all 必须为已知 scope")
	}
	if DataScope("department").Valid() {
		t.Fatal("最小闭环之外的 scope 不应判为已知值")
	}
}

func TestDataScopeCanAccess(t *testing.T) {
	cases := []struct {
		name       string
		scope      DataScope
		ownerID    int64
		operatorID int64
		want       bool
	}{
		{"self own resource", ScopeSelf, 7, 7, true},
		{"self other resource", ScopeSelf, 7, 8, false},
		{"self zero owner", ScopeSelf, 0, 8, false},
		{"all any resource", ScopeAll, 7, 8, true},
		{"all zero owner", ScopeAll, 0, 8, true},
		{"unknown defaults to self-like deny", DataScope(""), 7, 8, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.scope.CanAccess(tc.ownerID, tc.operatorID); got != tc.want {
				t.Fatalf("CanAccess(%d,%d) = %v, want %v", tc.ownerID, tc.operatorID, got, tc.want)
			}
		})
	}
}
