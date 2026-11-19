package conf

import (
	"os"
	"strconv"
	"time"
)

// QuoteTTL 返回报价快照有效期，默认 15 分钟，可用 PLANSTAR_QUOTE_TTL_SECONDS 覆盖。
//
// 配置读取集中在 conf 包（ADR-0001 P1）：领域层 biz 不再 import os/strconv，
// 由 cmd/planstar 在装配时通过 NewOrderUsecase 的 ttl 参数注入。
func QuoteTTL() time.Duration {
	if v := os.Getenv("PLANSTAR_QUOTE_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 15 * time.Minute
}
