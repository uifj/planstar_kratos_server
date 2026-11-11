package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

// MaxQuantityPerOrder 是单笔订单的最大购买数量（限购规则，后续可按回报档位 sell_limit 细化）。
const MaxQuantityPerOrder = 6

// defaultQuoteTTL 是报价快照的默认有效期（15 分钟）。
const defaultQuoteTTL = 15 * time.Minute

// quoteVersion 是当前报价快照的结构版本；订单必须携带一致版本才可创建。
const quoteVersion = 1

// QuoteInput 是确认订单（报价）入参。
type QuoteInput struct {
	ActivityID       int64
	ActivityEventID  int64
	TicketCategoryID int64
	Quantity         int
}

// QuoteSnapshot 是服务端权威报价快照（金额单位为分）。
// 落库只使用服务端计算结果，客户端传入金额仅用于一致性比对。
type QuoteSnapshot struct {
	QuoteID          string `json:"quoteId"`
	UserID           int64  `json:"userId,omitempty"`
	ActivityID       int64  `json:"activityId"`
	ActivityEventID  int64  `json:"activityEventId"`
	TicketCategoryID int64  `json:"ticketCategoryId,omitempty"`
	Title            string `json:"title,omitempty"`
	Poster           string `json:"poster,omitempty"`
	UnitPrice        int64  `json:"unitPrice"`
	Quantity         int    `json:"quantity"`
	TotalAmount      int64  `json:"totalAmount"`
	Version          int    `json:"version"`
	ExpiresAt        int64  `json:"expiresAt"`
}

// Quote 是确认接口对外返回的报价快照（字段命名 camelCase，与前端契约一致）。
type Quote = QuoteSnapshot

// QuoteStore 是报价快照存储端口（Redis 优先，未配置时由进程内 TTL 缓存实现）。
type QuoteStore interface {
	SaveQuote(ctx context.Context, snap *QuoteSnapshot) error
	GetQuote(ctx context.Context, quoteID string) (*QuoteSnapshot, error)
}

// QuoteTTLFromEnv 返回报价有效期，默认 15 分钟，可用 PLANSTAR_QUOTE_TTL_SECONDS 覆盖。
func QuoteTTLFromEnv() time.Duration {
	if v := os.Getenv("PLANSTAR_QUOTE_TTL_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return defaultQuoteTTL
}

// CalcQuoteTotal 按单价（分）与数量计算总额，并校验单价与限购数量。
func CalcQuoteTotal(unitPrice int64, quantity int) (int64, error) {
	if unitPrice <= 0 {
		return 0, ErrBadRequest
	}
	if quantity <= 0 {
		return 0, ErrBadRequest
	}
	if quantity > MaxQuantityPerOrder {
		return 0, ErrLimitExceeded
	}
	if int64(quantity) > math.MaxInt64/unitPrice {
		return 0, ErrLimitExceeded
	}
	return unitPrice * int64(quantity), nil
}

// VerifyQuote 校验报价快照：过期、版本、服务端重算金额与客户端金额一致性。
// amount <= 0 表示客户端未传金额，只做服务端重算校验。
func VerifyQuote(snap *QuoteSnapshot, amount int64, version int, now time.Time) error {
	if snap == nil {
		return ErrQuoteExpired
	}
	if snap.ExpiresAt > 0 && snap.ExpiresAt <= now.UnixMilli() {
		return ErrQuoteExpired
	}
	if version > 0 && version != snap.Version {
		return ErrPriceChanged
	}
	total, err := CalcQuoteTotal(snap.UnitPrice, snap.Quantity)
	if err != nil {
		return err
	}
	if total != snap.TotalAmount {
		return ErrPriceChanged
	}
	if amount > 0 && amount != total {
		return ErrPriceChanged
	}
	return nil
}

// Quote 生成服务端权威报价并写入快照存储（Redis 或进程内 TTL 缓存）。
func (uc *OrderUsecase) Quote(ctx context.Context, userID int64, in *QuoteInput) (*Quote, error) {
	snap, err := uc.buildQuote(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.SaveQuote(ctx, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

// buildQuote 从权威表读取企划/回报档位价格并生成快照；不信任任何客户端金额。
func (uc *OrderUsecase) buildQuote(ctx context.Context, userID int64, in *QuoteInput) (*QuoteSnapshot, error) {
	if in == nil || in.ActivityID <= 0 {
		return nil, ErrBadRequest
	}
	qty := in.Quantity
	if qty <= 0 {
		qty = 1
	}
	if qty > MaxQuantityPerOrder {
		return nil, ErrLimitExceeded
	}
	act, err := uc.repo.GetActivitySnapshot(ctx, in.ActivityID)
	if err != nil {
		return nil, err
	}
	if act == nil || act.SoldOut {
		return nil, ErrSoldOut
	}
	total, err := CalcQuoteTotal(act.UnitPrice, qty)
	if err != nil {
		return nil, err
	}
	quoteID, err := newQuoteID()
	if err != nil {
		return nil, err
	}
	// SKU 级定价待接入：当前由服务端在在售回报档位中选取最低价 SKU，不接受客户端指定回报档位参与定价/扣库存。
	categoryID := act.TicketCategoryID
	return &QuoteSnapshot{
		QuoteID:          quoteID,
		UserID:           userID,
		ActivityID:       act.ActivityID,
		ActivityEventID:  in.ActivityEventID,
		TicketCategoryID: categoryID,
		Title:            act.Title,
		Poster:           act.Poster,
		UnitPrice:        act.UnitPrice,
		Quantity:         qty,
		TotalAmount:      total,
		Version:          quoteVersion,
		ExpiresAt:        time.Now().Add(QuoteTTLFromEnv()).UnixMilli(),
	}, nil
}

// resolveQuote 解析下单使用的权威报价：优先使用 quoteId 快照，缺失时按当前价格重新报价。
func (uc *OrderUsecase) resolveQuote(ctx context.Context, userID int64, in *OrderInput) (*QuoteSnapshot, error) {
	if in.QuoteID != "" {
		snap, err := uc.repo.GetQuote(ctx, in.QuoteID)
		if err != nil {
			return nil, err
		}
		if snap.UserID != 0 && snap.UserID != userID {
			return nil, ErrQuoteExpired
		}
		if err := VerifyQuote(snap, in.Amount, in.QuoteVersion, time.Now()); err != nil {
			return nil, err
		}
		if in.ActivityID > 0 && snap.ActivityID != in.ActivityID {
			return nil, ErrPriceChanged
		}
		if in.ActivityEventID > 0 && snap.ActivityEventID > 0 && snap.ActivityEventID != in.ActivityEventID {
			return nil, ErrPriceChanged
		}
		if in.Quantity > 0 && in.Quantity != snap.Quantity {
			return nil, ErrBadRequest
		}
		return snap, nil
	}
	// 未携带 quoteId 时必须携带 amount，按当前权威价格重新报价后比对。
	if in.Amount <= 0 {
		return nil, ErrBadRequest
	}
	snap, err := uc.buildQuote(ctx, userID, &QuoteInput{
		ActivityID:       in.ActivityID,
		ActivityEventID:  in.ActivityEventID,
		TicketCategoryID: in.TicketCategoryID,
		Quantity:         in.Quantity,
	})
	if err != nil {
		return nil, err
	}
	if in.Amount != snap.TotalAmount {
		return nil, ErrPriceChanged
	}
	return snap, nil
}

// newQuoteID 生成 128 位随机报价 ID。
func newQuoteID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate quote id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
