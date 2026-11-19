package authz

// BuiltinModel 是仓内可审计的 RBAC 模型（ADR-0007 D3）。
// 与 go-wind-admin 不同：模型不以隐藏形式放在外部 kratos-authz 引擎里，
// 而是以字符串常量随代码提交，可被单测与 code review 直接检查。
const BuiltinModel = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p_eft == allow))

[matchers]
m = (r.sub == p.sub || g(r.sub, p.sub)) && (p.obj == "*" || keyMatch2(r.obj, p.obj)) && (p.act == "*" || r.act == p.act)
`

// Rule 是一条 p 策略：subject（角色码或用户名）、对象（method path 中的 path）、动作（HTTP method）。
type Rule struct {
	Sub string `json:"sub"`
	Obj string `json:"obj"`
	Act string `json:"act"`
}

// BuiltinPolicies 是编译期基线，运行时不可删除（防止误删 superadmin 把自己锁死）。
// 没有任何命中 allow 即拒绝 —— Casbin 默认 deny，绝不提供 noop 引擎。
var BuiltinPolicies = []Rule{
	// break-glass：共享运维令牌映射为显式 superadmin 策略，可审计、可撤销。
	{RuleSubjectSuperAdmin, "*", "*"},

	// 所有已认证账号都能登出并查看自己的权限。
	{"*authenticated", "/admin/auth/logout", "POST"},
	{"*authenticated", "/admin/rbac/me", "GET"},

	// 只读审计员。
	{RoleAuditor, "/admin/console", "GET"},
	{RoleAuditor, "/admin", "GET"},
	{RoleAuditor, "/admin/", "GET"},
	{RoleAuditor, "/admin/reports", "GET"},
	{RoleAuditor, "/admin/reports/:id", "GET"},
	{RoleAuditor, "/admin/projects/:id", "GET"},

	// 运营操作员：审计员权限 + 处置/上下架/退款审核写权限。
	{RoleOperator, "/admin/console", "GET"},
	{RoleOperator, "/admin", "GET"},
	{RoleOperator, "/admin/", "GET"},
	{RoleOperator, "/admin/reports", "GET"},
	{RoleOperator, "/admin/reports/:id", "GET"},
	{RoleOperator, "/admin/reports/:id/handle", "POST"},
	{RoleOperator, "/admin/projects/:id", "GET"},
	{RoleOperator, "/admin/projects/:id/review", "POST"},
	{RoleOperator, "/admin/feeds/:id/status", "POST"},
	{RoleOperator, "/admin/goods/:id/status", "POST"},
	{RoleOperator, "/admin/goods-skus/:id/status", "POST"},
	{RoleOperator, "/admin/refunds/:id/review", "POST"},

	// RBAC 管理员：只能管理策略/角色，不能碰业务处置端点（最小权限）。
	{RoleRBACAdmin, "/admin/rbac", "GET"},
	{RoleRBACAdmin, "/admin/rbac/*", "*"},
	{RoleRBACAdmin, "/admin/rbac/users/:name/roles", "*"},
}

const (
	RuleSubjectSuperAdmin = "superadmin"
	RoleOperator          = "operator"
	RoleAuditor           = "auditor"
	RoleRBACAdmin         = "rbac_admin"
	// SubjectAuthenticated 是内置伪角色，由 Enforcer 在每次判定时附给所有已认证身份。
	SubjectAuthenticated = "*authenticated"
)

// GrantableRoles 是允许通过管理 API 授予的角色白名单。
// superadmin 只能通过 break-glass 共享运维令牌获得（基线策略 {superadmin,*,*}），
// 绝不开放给 RBAC 管理接口，避免自我提权（PRODUCTION_AUDIT S2）。
var GrantableRoles = []string{RoleOperator, RoleAuditor, RoleRBACAdmin}

func IsGrantableRole(role string) bool {
	for _, r := range GrantableRoles {
		if r == role {
			return true
		}
	}
	return false
}
