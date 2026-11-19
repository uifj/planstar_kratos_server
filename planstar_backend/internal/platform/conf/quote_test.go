package conf

import (
	"testing"
	"time"
)

// TestQuoteTTL 校验报价 TTL 的配置读取集中在 conf 包（ADR-0001 P1）。
func TestQuoteTTL(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want time.Duration
	}{
		{"未配置使用默认 15 分钟", "", 15 * time.Minute},
		{"合法秒数", "42", 42 * time.Second},
		{"非正数回落默认", "-1", 15 * time.Minute},
		{"零回落默认", "0", 15 * time.Minute},
		{"非法值回落默认", "not-a-number", 15 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("PLANSTAR_QUOTE_TTL_SECONDS", tc.env)
			if got := QuoteTTL(); got != tc.want {
				t.Errorf("QuoteTTL() = %v, want %v", got, tc.want)
			}
		})
	}
}
