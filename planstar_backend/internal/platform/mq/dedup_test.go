package mq

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestDedupCacheSeenAndMark(t *testing.T) {
	d := newDedupCache(time.Minute, 100)
	if d.Seen("evt-1") {
		t.Fatal("未处理过的 ID 不应命中")
	}
	d.Mark("evt-1")
	if !d.Seen("evt-1") {
		t.Fatal("已处理的 ID 应命中")
	}
	if d.Seen("") {
		t.Fatal("空 ID 不参与去重")
	}
}

func TestDedupCacheTTLExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	d := newDedupCache(10*time.Second, 100)
	d.now = func() time.Time { return now }
	d.Mark("evt-1")
	now = now.Add(11 * time.Second)
	if d.Seen("evt-1") {
		t.Fatal("TTL 过期后应可再次处理")
	}
	if d.Len() != 0 {
		t.Errorf("过期项应被惰性清理，剩余 %d", d.Len())
	}
}

func TestDedupCacheBounded(t *testing.T) {
	d := newDedupCache(time.Hour, 3)
	for i := 0; i < 50; i++ {
		d.Mark(string(rune('a' + i)))
	}
	if d.Len() > 3 {
		t.Errorf("去重表超过容量上限: %d", d.Len())
	}
}

func TestRegisterMQMetricsIsIdempotent(t *testing.T) {
	reg := prometheus.NewRegistry()
	RegisterMQMetrics(reg)
	RegisterMQMetrics(reg)
	families, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	if len(families) == 0 {
		t.Fatal("MQ 指标未注册")
	}
}
