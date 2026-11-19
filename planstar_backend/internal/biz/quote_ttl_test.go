package biz

import (
	"testing"
	"time"
)

// TestOrderUsecaseQuoteTTLInjection 校验报价 TTL 由装配层注入，
// 领域层不再直接读取环境变量（ADR-0001 P1）。
func TestOrderUsecaseQuoteTTLInjection(t *testing.T) {
	cases := []struct {
		name string
		ttl  []time.Duration
		want time.Duration
	}{
		{"省略参数使用默认值", nil, DefaultQuoteTTL},
		{"非正数使用默认值", []time.Duration{0}, DefaultQuoteTTL},
		{"注入值生效", []time.Duration{30 * time.Second}, 30 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := NewOrderUsecase(nil, nil, nil, tc.ttl...)
			if uc.quoteTTL != tc.want {
				t.Errorf("quoteTTL = %v, want %v", uc.quoteTTL, tc.want)
			}
			if uc.events == nil || uc.log == nil {
				t.Error("构造函数应补齐 events 与 logger 的非空默认值")
			}
		})
	}
}
