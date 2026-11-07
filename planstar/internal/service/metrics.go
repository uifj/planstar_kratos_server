package service

import (
	"github.com/prometheus/client_golang/prometheus"
)

// 业务指标：与 HTTP 指标分离，便于按业务结果做告警与看板。
// 统一注册到 /metrics 使用的独立 Registry（见 main.go）。
var (
	loginTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "planstar",
		Subsystem: "biz",
		Name:      "login_total",
		Help:      "登录次数，按结果分组（success/failure/captcha）",
	}, []string{"result"})

	orderConfirmTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "planstar",
		Subsystem: "biz",
		Name:      "order_confirm_total",
		Help:      "订单报价次数，按结果分组",
	}, []string{"result"})

	orderCreatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "planstar",
		Subsystem: "biz",
		Name:      "order_created_total",
		Help:      "订单创建次数，按结果分组（success/price_changed/sold_out/limit_exceeded/quote_expired/error）",
	}, []string{"result"})
)

// RegisterBizMetrics 把业务指标注册到指定注册表（/metrics 使用独立 Registry）。
func RegisterBizMetrics(reg prometheus.Registerer) {
	if reg == nil {
		return
	}
	reg.MustRegister(loginTotal, orderConfirmTotal, orderCreatedTotal)
}
