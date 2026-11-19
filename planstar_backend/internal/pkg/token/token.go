// Package token 提供纯算法的安全令牌生成，刻意不依赖 net/http。
//
// biz 层只需生成会话令牌，不应因此传递依赖 net/http、Cookie 或任何 HTTP 适配器；
// Cookie 读写保留在 internal/pkg/session。
package token

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// SessionTTL 会话默认有效期（契约：Cookie Max-Age 固定 30 天）。
const SessionTTL = 30 * 24 * time.Hour

// DeviceTTL 设备标识有效期（7 天）。
const DeviceTTL = 7 * 24 * time.Hour

// New 生成随机会话令牌（32 字节 -> 64 位 hex）。
// 随机源失败必须返回错误，不允许 panic，也不允许回退到可预测值。
func New() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
