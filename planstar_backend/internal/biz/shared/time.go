package shared

import "time"

// FmtDT 输出客户端可解析的 DATETIME 字符串（契约：yyyy-MM-dd HH:mm:ss）。
func FmtDT(t time.Time) string { return t.Format("2006-01-02 15:04:05") }

// FmtDTp 空时间返回空串。
func FmtDTp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return FmtDT(*t)
}

// BirthTs 生日 DATE -> 毫秒时间戳（0 = 未设置，契约要求）。
func BirthTs(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}
