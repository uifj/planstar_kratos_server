package shared

import "math"

// Fen 把元（DECIMAL(12,2)）转成客户端期望的分（×100，四舍五入为整数）。
// 契约：MySQL 存元，HTTP 出入参一律分；1 星币 = 100 分。
func Fen(v float64) int64 {
	if v == 0 {
		return 0
	}
	return int64(math.Round(v * 100))
}
