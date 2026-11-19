package authz

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"planstar/internal/platform/authn"
)

func newTestEnforcer(t *testing.T) *Enforcer {
	t.Helper()
	e, err := New(NewMemoryPolicyStore())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

func authorize(t *testing.T, e *Enforcer, subjects []string, obj, act string) bool {
	t.Helper()
	ok, err := e.Authorize(context.Background(), subjects, obj, act)
	if err != nil {
		t.Fatalf("Authorize(%v,%s,%s): %v", subjects, obj, act, err)
	}
	return ok
}

func TestDefaultDeny(t *testing.T) {
	e := newTestEnforcer(t)
	// 没有任何角色的身份（Enforcer 会附 *authenticated 伪角色）只能访问两个基础端点。
	if authorize(t, e, []string{"stranger"}, "/admin/reports", "GET") {
		t.Fatal("陌生用户不应能读举报列表（默认拒绝）")
	}
	if !authorize(t, e, []string{"stranger"}, "/admin/rbac/me", "GET") {
		t.Fatal("已认证身份应能查看自身权限")
	}
	if authorize(t, e, nil, "/admin/reports/1/handle", "POST") {
		t.Fatal("无 subject 不应放行写操作")
	}
}

func TestSuperAdminIsExplicitPolicy(t *testing.T) {
	e := newTestEnforcer(t)
	if !authorize(t, e, []string{RuleSubjectSuperAdmin}, "/metrics", "GET") {
		t.Fatal("共享运维令牌映射的 superadmin 应被显式策略放行")
	}
	if !authorize(t, e, []string{RuleSubjectSuperAdmin}, "/admin/anything/deep/path", "DELETE") {
		t.Fatal("superadmin 的 * 策略应匹配多层路径")
	}
	// 策略可撤销：删掉 superadmin 基线是不允许的，但新增运行时策略可单独管理。
	if err := e.RemoveRule(context.Background(), BuiltinPolicies[0]); err == nil {
		t.Fatal("内置基线策略必须不可删除，避免锁死运维入口")
	}
}

func TestKeyMatch2PathPolicies(t *testing.T) {
	e := newTestEnforcer(t)
	if !authorize(t, e, []string{RoleOperator}, "/admin/reports/123", "GET") {
		t.Error("keyMatch2 应匹配 /admin/reports/:id")
	}
	if !authorize(t, e, []string{RoleOperator}, "/admin/refunds/9/review", "POST") {
		t.Error("operator 应有退款审核权限")
	}
	if authorize(t, e, []string{RoleOperator}, "/admin/rbac/policies", "GET") {
		t.Error("operator 不应能查看 RBAC 策略（最小权限）")
	}
	if authorize(t, e, []string{RoleAuditor}, "/admin/reports/123/handle", "POST") {
		t.Error("审计员不应有处置写权限")
	}
	if !authorize(t, e, []string{RoleRBACAdmin}, "/admin/rbac/policies", "POST") {
		t.Error("rbac_admin 应能管理策略")
	}
}

func TestRuntimePolicyAddAndRemove(t *testing.T) {
	e := newTestEnforcer(t)
	rule := Rule{Sub: "custom_role", Obj: "/admin/console", Act: "GET"}
	if authorize(t, e, []string{"custom_role"}, "/admin/console", "GET") {
		t.Fatal("新增前不应放行")
	}
	if err := e.AddRule(context.Background(), rule); err != nil {
		t.Fatalf("AddRule: %v", err)
	}
	if !authorize(t, e, []string{"custom_role"}, "/admin/console", "GET") {
		t.Fatal("新增策略后应放行")
	}
	if err := e.RemoveRule(context.Background(), rule); err != nil {
		t.Fatalf("RemoveRule: %v", err)
	}
	if authorize(t, e, []string{"custom_role"}, "/admin/console", "GET") {
		t.Fatal("删除策略后应恢复默认拒绝")
	}
	if err := e.AddRule(context.Background(), Rule{Sub: "x", Obj: "relative-path", Act: "GET"}); err == nil {
		t.Fatal("obj 必须以 / 开头或为 *")
	}
}

func TestGroupingInheritsRole(t *testing.T) {
	e := newTestEnforcer(t)
	if authorize(t, e, []string{"bob"}, "/admin/reports", "GET") {
		t.Fatal("分配角色前不应放行")
	}
	if err := e.AddGrouping(context.Background(), Grouping{User: "bob", Role: RoleAuditor}); err != nil {
		t.Fatalf("AddGrouping: %v", err)
	}
	if !authorize(t, e, []string{"bob"}, "/admin/reports", "GET") {
		t.Fatal("g 策略应让用户继承角色权限")
	}
	if authorize(t, e, []string{"bob"}, "/admin/reports/1/handle", "POST") {
		t.Fatal("继承角色不应越权")
	}
}

func TestPolicyStorePersistenceAndReload(t *testing.T) {
	store := NewMemoryPolicyStore()
	e, err := New(store)
	if err != nil {
		t.Fatal(err)
	}
	rule := Rule{Sub: "persisted", Obj: "/admin/console", Act: "GET"}
	if err := e.AddRule(context.Background(), rule); err != nil {
		t.Fatal(err)
	}
	e2, err := New(store)
	if err != nil {
		t.Fatalf("用同一 store 重建失败: %v", err)
	}
	if !authorize(t, e2, []string{"persisted"}, "/admin/console", "GET") {
		t.Fatal("增量策略应从 store 恢复")
	}
	if err := e2.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if len(e2.Rules()) != len(BuiltinPolicies)+1 {
		t.Errorf("Reload 后策略数 = %d", len(e2.Rules()))
	}
}

func TestConcurrentEnforceAndPolicyChanges(t *testing.T) {
	e := newTestEnforcer(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			role := fmt.Sprintf("role_%d", n%3)
			_ = e.AddRule(context.Background(), Rule{Sub: role, Obj: "/admin/console", Act: "GET"})
			_, _ = e.Authorize(context.Background(), []string{role}, "/admin/console", "GET")
			_ = e.Reload(context.Background())
		}(i)
	}
	wg.Wait()
	if !authorize(t, e, []string{RuleSubjectSuperAdmin}, "/admin/console", "GET") {
		t.Fatal("并发策略变更后内置基线仍应生效")
	}
}

func TestDecisionFailClosedWithoutAuthorizer(t *testing.T) {
	m := NewMiddleware(nil, nil)
	allowed, _, _ := m.Decision(context.Background(), &authn.Identity{Subject: "alice"}, "/admin/reports", "GET")
	if allowed {
		t.Fatal("未装配授权器必须拒绝，绝不退化成全放行")
	}
}

// 以下两项是 PRODUCTION_AUDIT S1/S2 的攻击性回归测试。

func TestRuntimeRuleCannotGrantWildcardPrivilege(t *testing.T) {
	e := newTestEnforcer(t)
	attacks := []Rule{
		{Sub: "attacker", Obj: "*", Act: "*"},
		{Sub: "attacker", Obj: "/admin/*", Act: "*"},
		{Sub: RuleSubjectSuperAdmin, Obj: "/admin/reports", Act: "GET"},
	}
	for _, r := range attacks {
		if err := e.AddRule(context.Background(), r); err == nil {
			t.Errorf("危险策略 %+v 应被拒绝", r)
		}
	}
	if authorize(t, e, []string{"attacker"}, "/admin/reports", "GET") {
		t.Error("攻击主体不应获得任何权限")
	}
}

func TestGroupingCannotGrantSuperAdmin(t *testing.T) {
	e := newTestEnforcer(t)
	if err := e.AddGrouping(context.Background(), Grouping{User: "bob", Role: RuleSubjectSuperAdmin}); err == nil {
		t.Fatal("通过 g 策略授予 superadmin 必须被拒绝")
	}
	if err := e.AddGrouping(context.Background(), Grouping{User: "bob", Role: RoleAuditor}); err != nil {
		t.Fatalf("白名单角色应可授予: %v", err)
	}
	if !authorize(t, e, []string{"bob"}, "/admin/reports", "GET") {
		t.Error("白名单角色应正常继承权限")
	}
}
