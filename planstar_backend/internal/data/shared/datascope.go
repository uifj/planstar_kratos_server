package shared

import (
	bizshared "planstar/internal/biz/shared"
)

// ScopedWhere 返回数据权限过滤条件片段（ADR-0010 P0②）。
// 调用方把它拼进 SQL 的 WHERE（如 `SELECT ... FROM t WHERE 1=1` + 片段），
// 参数用 `args...` 透传给 QueryContext。
//
//   - ScopeAll：返回空片段、无参数（不过滤）。
//   - ScopeSelf：返回 `ownerColumn = ?` 并附带 ownerID（行级隔离）。
//
// 注意：列名由调用方以常量传入，禁止拼接不可信输入，避免 SQL 注入。
func ScopedWhere(scope bizshared.DataScope, ownerColumn string, ownerID int64) (string, []any) {
	if scope == bizshared.ScopeSelf {
		return ownerColumn + " = ?", []any{ownerID}
	}
	return "", nil
}
