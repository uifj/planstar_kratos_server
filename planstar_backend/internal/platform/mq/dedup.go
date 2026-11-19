package mq

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// dedupTTL 覆盖 RabbitMQ 最长重试窗口（1+2+4+8+16s）与运维人工重放周期。
	dedupTTL = 24 * time.Hour
	// dedupMaxEntries 防止单实例长期运行导致去重表无限增长。
	dedupMaxEntries = 100_000
)

var (
	consumedEvents = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "planstar", Subsystem: "mq", Name: "consumed_total",
		Help: "成功处理的领域事件数",
	})
	duplicateEvents = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "planstar", Subsystem: "mq", Name: "duplicate_total",
		Help: "因事件 ID 重复而跳过的投递数",
	})
	failedEvents = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "planstar", Subsystem: "mq", Name: "consume_failed_total",
		Help: "处理失败并进入重试/死信链路的投递数",
	})
	dlqEvents = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "planstar", Subsystem: "mq", Name: "dlq_total",
		Help: "达到最大重试次数、被 Nack 到死信交换机的消息数",
	})
	publishFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "planstar", Subsystem: "mq", Name: "publish_failed_total",
		Help: "发布失败（仅 Warn，不阻断主流程）的事件数",
	})
)

// RegisterMQMetrics 把 MQ 指标注册到 /metrics 使用的独立 Registry。
// 重复注册（AlreadyRegisteredError）直接忽略，便于本地测试多次装配。
func RegisterMQMetrics(reg *prometheus.Registry) {
	for _, c := range []prometheus.Collector{consumedEvents, duplicateEvents, failedEvents, dlqEvents, publishFailed} {
		// 重复注册（AlreadyRegisteredError）与注册失败都不应影响进程启动；
		// 指标是运维辅助，不作为启动闸门。
		_ = reg.Register(c)
	}
}

// dedupCache 是单实例的带 TTL 事件 ID 去重表（ADR-0004）。
// 多实例部署时各实例独立去重；彻底消除跨实例重复需 Redis/MySQL 去重表（后续批次）。
type dedupCache struct {
	mu   sync.Mutex
	seen map[string]time.Time
	ttl  time.Duration
	max  int
	now  func() time.Time
}

func newDedupCache(ttl time.Duration, max int) *dedupCache {
	if ttl <= 0 {
		ttl = dedupTTL
	}
	if max <= 0 {
		max = dedupMaxEntries
	}
	return &dedupCache{seen: make(map[string]time.Time), ttl: ttl, max: max, now: time.Now}
}

// Seen 返回该事件 ID 是否已在 TTL 内被处理过；空 ID 不去重。
func (d *dedupCache) Seen(id string) bool {
	if id == "" {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.now()
	if ts, ok := d.seen[id]; ok {
		if now.Sub(ts) < d.ttl {
			return true
		}
		delete(d.seen, id)
	}
	d.sweepLocked(now)
	return false
}

// Mark 记录已成功处理的事件 ID。
func (d *dedupCache) Mark(id string) {
	if id == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	now := d.now()
	d.sweepLocked(now)
	// 达到容量上限时先清过期项，再按最早时间戳淘汰，保证不会无界增长。
	if len(d.seen) >= d.max {
		for k, ts := range d.seen {
			if now.Sub(ts) >= d.ttl {
				delete(d.seen, k)
			}
		}
		for len(d.seen) >= d.max {
			var oldestKey string
			var oldest time.Time
			for k, ts := range d.seen {
				if oldestKey == "" || ts.Before(oldest) {
					oldestKey, oldest = k, ts
				}
			}
			if oldestKey == "" {
				break
			}
			delete(d.seen, oldestKey)
		}
	}
	d.seen[id] = now
}

// sweepLocked 惰性清理过期项；调用方必须持锁。
func (d *dedupCache) sweepLocked(now time.Time) {
	for k, ts := range d.seen {
		if now.Sub(ts) >= d.ttl {
			delete(d.seen, k)
		}
	}
}

func (d *dedupCache) Len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.seen)
}
