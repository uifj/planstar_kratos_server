package shared

import "strings"

// MaskMobile 脱敏手机号，例如 138****1234。
// 领域日志需要脱敏，但不应为此 import 带 HTTP/SDK 依赖的 notify 包。
func MaskMobile(mobile string) string {
	r := []rune(strings.TrimSpace(mobile))
	switch {
	case len(r) == 0:
		return ""
	case len(r) <= 4:
		return strings.Repeat("*", len(r))
	case len(r) <= 7:
		return string(r[:2]) + strings.Repeat("*", len(r)-2)
	default:
		return string(r[:3]) + "****" + string(r[len(r)-4:])
	}
}
