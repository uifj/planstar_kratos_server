package shared

// DataScope 表示运营账号的数据权限范围（ADR-0010 P0② 最小闭环：本人/全部）。
// 它是 Casbin 接口级 RBAC 之上的「行级」约束：接口放行后，data 层再按 scope
// 决定能看到/操作哪些行。当前只落地枚举与判定语义，尚未绑定到任何「操作者归属」
// 列；待运营后台出现可归属资源（如带 assignee 的审核单）时再接线。
type DataScope string

const (
	// ScopeSelf 仅能访问「归属者 == 操作者」的数据（最小权限默认）。
	ScopeSelf DataScope = "self"
	// ScopeAll 可访问全部数据（superadmin）。
	ScopeAll DataScope = "all"
)

// Valid 判定 scope 是否为已知值；未知值按 self 处理（fail-closed）。
func (s DataScope) Valid() bool {
	return s == ScopeSelf || s == ScopeAll
}

// CanAccess 是 IDOR 防护的核心判定：self 范围要求资源归属者 == 操作者，
// 否则拒绝（越权访问由调用方映射为 403/404）；all 范围放行。
func (s DataScope) CanAccess(ownerID, operatorID int64) bool {
	if s == ScopeAll {
		return true
	}
	return ownerID == operatorID
}
