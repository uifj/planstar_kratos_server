package biz

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func newID() string { return uuid.Must(uuid.NewV7()).String() }

func randomSuffix(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func newOrderSn() string {
	return "ORD" + time.Now().Format("20060102150405") + strings.ToUpper(randomSuffix(4))
}

func newPaymentNo() string {
	return "PAY" + time.Now().Format("20060102150405") + strings.ToUpper(randomSuffix(4))
}

func newDomainEvent(eventType, orderSn, userID string) *DomainEvent {
	return &DomainEvent{
		ID:         newID(),
		Type:       eventType,
		OrderSn:    orderSn,
		UserID:     userID,
		OccurredAt: time.Now().Unix(),
	}
}

func encodeEvent(e *DomainEvent) string {
	b, _ := json.Marshal(e)
	return string(b)
}

func decodeEvent(payload string) (*DomainEvent, error) {
	var e DomainEvent
	if err := json.Unmarshal([]byte(payload), &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func pageBounds(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func validMobile(mobile string) bool {
	if len(mobile) != 11 {
		return false
	}
	for _, r := range mobile {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
