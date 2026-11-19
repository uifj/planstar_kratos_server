package biz

import (
	"context"
	"strings"
	"time"
)

// Idea is a community idea (planstar_catalog.project_idea).
type Idea struct {
	ID            int64
	UserID        int64
	Title         string
	Summary       string
	Cover         string
	CategoryID    int64
	CategoryName  string
	Stage         int32
	PublishStatus int32
	Status        int32
	SupportCount  int32
	OpposeCount   int32
	CoinIntentSum int64
	CreatedAt     time.Time
}

// IdeaVote is one vote on an idea (planstar_catalog.project_idea_vote).
type IdeaVote struct {
	ID         int64
	IdeaID     int64
	UserID     int64
	VoteType   int32
	CoinTier   int32
	ReasonCode string
	ReasonText string
	CreatedAt  time.Time
}

// IdeaDetail is an idea plus its vote aggregation and recent votes.
type IdeaDetail struct {
	Idea         Idea
	Description  string
	VotesTotal   int64
	VotesSupport int64
	VotesOppose  int64
	RecentVotes  []*IdeaVote
}

// Goods is a shop item (planstar_shop.goods).
type Goods struct {
	ID        int64
	GoodsSN   string
	Name      string
	Brief     string
	Cover     string
	Category  string
	Status    int32
	Sort      int32
	CreatedAt time.Time
}

// GoodsSku is one sellable unit (planstar_shop.goods_sku).
type GoodsSku struct {
	ID         int64
	GoodsID    int64
	SkuName    string
	PriceCoins int32
	Stock      int32
	Sold       int32
	Status     int32
}

// GoodsDetail is a goods item plus its SKUs.
type GoodsDetail struct {
	Goods       Goods
	Description string
	Skus        []*GoodsSku
}

// GoodsOrder is a coin purchase (planstar_shop.goods_order).
type GoodsOrder struct {
	ID         int64
	OrderNo    string
	UserID     int64
	GoodsID    int64
	SkuID      int64
	SkuName    string
	PriceCoins int32
	Status     string
	CreatedAt  time.Time
}

// PayOrder is a payment attempt (planstar_payment.pay_order).
type PayOrder struct {
	ID           int64
	PayNo        string
	OrderNo      string
	OrderID      int64
	UserID       int64
	Channel      string
	Amount       int64
	Status       string
	ThirdTradeNo string
	PaidAt       *time.Time
	ExpireAt     time.Time
	CreatedAt    time.Time
}

// Refund is a refund request (planstar_payment.order_refund).
type Refund struct {
	ID                  int64
	RefundNo            string
	OrderID             int64
	OrderNo             string
	UserID              int64
	ReasonCode          string
	ReasonText          string
	RefundableAmount    int64
	Status              string
	ExpectedArrivalDays int32
	CreatedAt           time.Time
}

// OrderDetail is a unified order plus its payment attempts and refunds.
type OrderDetail struct {
	Order        Order
	Title        string
	TotalAmount  int64
	RefundAmount int64
	PayTime      *time.Time
	PayOrders    []*PayOrder
	Refunds      []*Refund
}

// Feed is a community post (planstar_community.feed).
type Feed struct {
	ID         int64
	Type       int32
	UserID     int64
	Title      string
	Content    string
	Likes      int32
	ReplyCount int32
	Views      int32
	Status     int32
	IsQuality  int32
	CreatedAt  time.Time
}

// FeedDetail is a feed plus its media and author snapshot.
type FeedDetail struct {
	Feed       Feed
	Images     []string
	Videos     []string
	Tags       []string
	AuthorName string
}

// IdeaQuery filters and pages the idea list. A nil Status means "all".
type IdeaQuery struct {
	Keyword string
	Status  *int32
	Offset  int
	Limit   int
}

// GoodsQuery filters and pages the goods list. A nil Status means "all".
type GoodsQuery struct {
	Keyword string
	Status  *int32
	Offset  int
	Limit   int
}

// GoodsOrderQuery filters and pages the goods-order list. Empty Status means "all".
type GoodsOrderQuery struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

// PayOrderQuery filters and pages the pay-order list. Empty Status means "all".
type PayOrderQuery struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

// RefundQuery filters and pages the refund list. Empty Status means "all".
type RefundQuery struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

// FeedQuery filters and pages the feed list. A nil Status means "all";
// 1 selects normal feeds and 0 selects hidden ones (status <> 1).
type FeedQuery struct {
	Keyword string
	Status  *int32
	Offset  int
	Limit   int
}

// ListIdeas lists community ideas.
func (uc *ConsoleUsecase) ListIdeas(ctx context.Context, q IdeaQuery) ([]*Idea, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Status != nil && (*q.Status < 0 || *q.Status > 3) {
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListIdeas(ctx, q)
}

// GetIdea returns one idea with its vote aggregation and recent votes.
func (uc *ConsoleUsecase) GetIdea(ctx context.Context, id int64) (*IdeaDetail, error) {
	if id <= 0 {
		return nil, ErrIdeaNotFound
	}
	return uc.repo.GetIdea(ctx, id)
}

// ListGoods lists shop goods.
func (uc *ConsoleUsecase) ListGoods(ctx context.Context, q GoodsQuery) ([]*Goods, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Status != nil && (*q.Status < 0 || *q.Status > 1) {
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListGoods(ctx, q)
}

// GetGoods returns one goods item with its SKUs.
func (uc *ConsoleUsecase) GetGoods(ctx context.Context, id int64) (*GoodsDetail, error) {
	if id <= 0 {
		return nil, ErrGoodsNotFound
	}
	return uc.repo.GetGoods(ctx, id)
}

// ListGoodsOrders lists coin purchases.
func (uc *ConsoleUsecase) ListGoodsOrders(ctx context.Context, q GoodsOrderQuery) ([]*GoodsOrder, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	q.Status = strings.TrimSpace(q.Status)
	switch q.Status {
	case "", "paid", "completed", "cancelled":
	default:
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListGoodsOrders(ctx, q)
}

// GetOrder returns one unified order with its payments and refunds.
func (uc *ConsoleUsecase) GetOrder(ctx context.Context, id int64) (*OrderDetail, error) {
	if id <= 0 {
		return nil, ErrOrderNotFound
	}
	return uc.repo.GetOrder(ctx, id)
}

// ListPayOrders lists payment attempts.
func (uc *ConsoleUsecase) ListPayOrders(ctx context.Context, q PayOrderQuery) ([]*PayOrder, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	q.Status = strings.TrimSpace(q.Status)
	switch q.Status {
	case "", "pending", "success", "failed", "expired":
	default:
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListPayOrders(ctx, q)
}

// ListRefunds lists refund requests.
func (uc *ConsoleUsecase) ListRefunds(ctx context.Context, q RefundQuery) ([]*Refund, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	q.Status = strings.TrimSpace(q.Status)
	switch q.Status {
	case "", "applied", "processing", "success", "rejected":
	default:
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListRefunds(ctx, q)
}

// ListFeeds lists community feeds.
func (uc *ConsoleUsecase) ListFeeds(ctx context.Context, q FeedQuery) ([]*Feed, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Status != nil && (*q.Status < 0 || *q.Status > 1) {
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListFeeds(ctx, q)
}

// GetFeed returns one feed with its media and author snapshot.
func (uc *ConsoleUsecase) GetFeed(ctx context.Context, id int64) (*FeedDetail, error) {
	if id <= 0 {
		return nil, ErrFeedNotFound
	}
	return uc.repo.GetFeed(ctx, id)
}