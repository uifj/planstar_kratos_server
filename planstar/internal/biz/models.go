package biz

import (
	"math"
	"time"
)

// fen 把元（DECIMAL(12,2)）转成客户端期望的分（×100，四舍五入为整数）。
func fen(v float64) int64 {
	if v == 0 {
		return 0
	}
	return int64(math.Round(v * 100))
}

// fmtDT 输出客户端可解析的 ISO 时间（契约：DATETIME -> ISO-8601）。
func fmtDT(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// fmtDTp 空时间返回空串。
func fmtDTp(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtDT(*t)
}

// birthTs 生日 DATE -> 毫秒时间戳（0 = 未设置，契约要求）。
func birthTs(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

// 导出给 data 层使用的转换助手。

func Fen(v float64) int64        { return fen(v) }
func FmtDT(t time.Time) string   { return fmtDT(t) }
func FmtDTp(t *time.Time) string { return fmtDTp(t) }
func BirthTs(t *time.Time) int64 { return birthTs(t) }

// Banner 首页轮播（契约 4.2 v1/home/banners）。
type Banner struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Image  string `json:"image"`
	Poster string `json:"poster,omitempty"`
	URL    string `json:"url,omitempty"`
	ShowID int64  `json:"showId,omitempty"`
	Sort   int    `json:"sort,omitempty"`
}

// AuthUser 登录返回的用户主档（契约 4.1 的 AuthUser）。
type AuthUser struct {
	UserID       int64  `json:"userId"`
	UserName     string `json:"userName"`
	Avatar       string `json:"avatar"`
	Bio          string `json:"bio,omitempty"`
	Gender       int    `json:"gender"`
	Birthday     int64  `json:"birthday"`
	MobileNo     string `json:"mobileNo,omitempty"`
	Level        int    `json:"level"`
	FollowerNum  int    `json:"followerNum"`
	FollowingNum int    `json:"followingNum"`
	WantWatchNum int    `json:"wantWatchNum"`
	IsDouliUser  bool   `json:"isDouliUser"`
	BadgeIcon    string `json:"badgeIcon,omitempty"`
}

// LoginBean 登录响应（裸对象，无信封）。
type LoginBean struct {
	NeedCaptcha bool      `json:"needCaptcha"`
	NewUser     bool      `json:"newUser"`
	User        *AuthUser `json:"user"`
}

// SendCodeResp 发码响应（契约：HTTP 200 + {success, needCaptcha, errorMsg}）。
type SendCodeResp struct {
	Success     bool   `json:"success"`
	NeedCaptcha bool   `json:"needCaptcha"`
	ErrorMsg    string `json:"errorMsg,omitempty"`
}

// ShowSummary 演出库条目（契约 4.2，字段给全两个别名）。
type ShowSummary struct {
	ID               int64            `json:"id"`
	ActivityID       int64            `json:"activityId"`
	Title            string           `json:"title"`
	Name             string           `json:"name"`
	Artist           string           `json:"artist"`
	Star             string           `json:"star"`
	CategoryID       int64            `json:"categoryId"`
	Category         int64            `json:"category"`
	CategoryName     string           `json:"categoryName"`
	City             string           `json:"city"`
	Venue            string           `json:"venue"`
	VenueName        string           `json:"venueName"`
	VenueAddress     string           `json:"venueAddress"`
	FirstSessionAt   string           `json:"firstSessionAt"`
	ShowTime         string           `json:"showTime"`
	PriceFrom        int64            `json:"priceFrom"`
	MinPrice         int64            `json:"minPrice"`
	ListPriceFrom    int64            `json:"listPriceFrom"`
	OriginalPrice    int64            `json:"originalPrice"`
	DiscountPermille int              `json:"discountPermille"`
	Discount         int              `json:"discount"`
	Tags             []string         `json:"tags"`
	WishCount        int              `json:"wishCount"`
	WannaSeeCount    int              `json:"wannaSeeCount"`
	SoldOut          bool             `json:"soldOut"`
	Status           bool             `json:"status"`
	ProxyBuy         bool             `json:"proxyBuy"`
	Rating           float64          `json:"rating"`
	RatingCount      int              `json:"ratingCount"`
	Duration         string           `json:"duration"`
	Intro            string           `json:"intro"`
	Description      string           `json:"description"`
	Poster           string           `json:"poster"`
	Sessions         []SessionSummary `json:"sessions"`
	Tiers            []TierSummary    `json:"tiers"`
	Lineup           []string         `json:"lineup"`
	PurchaseNotice   []string         `json:"purchaseNotice"`
}

// SessionSummary 场次摘要。
type SessionSummary struct {
	StartsAt    string `json:"startsAt"`
	ShowTime    string `json:"showTime"`
	TierSummary string `json:"tierSummary"`
	SoldOut     bool   `json:"soldOut"`
}

// TierSummary 票档摘要。
type TierSummary struct {
	Name      string `json:"name"`
	TierName  string `json:"tierName"`
	Price     int64  `json:"price"`
	Note      string `json:"note,omitempty"`
	Remaining int    `json:"remaining"`
}

// FeedPost 动态（契约 4.3）。
type FeedPost struct {
	ID          int64    `json:"id"`
	TweetID     int64    `json:"tweetId"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Author      string   `json:"author"`
	UserName    string   `json:"userName"`
	ViewCount   int      `json:"viewCount"`
	ReadCount   int      `json:"readCount"`
	PublishedAt string   `json:"publishedAt"`
	CreateTime  string   `json:"createTime"`
	Tag         string   `json:"tag"`
	ShowID      int64    `json:"showId"`
	ActivityID  int64    `json:"activityId"`
	LikeCount   int      `json:"likeCount"`
	LikeNum     int      `json:"likeNum"`
	HasVideo    bool     `json:"hasVideo"`
	Video       bool     `json:"video"`
	Body        string   `json:"body,omitempty"`
	Images      []string `json:"images,omitempty"`
}

// FeedComment 动态评论（契约 4.3）。
type FeedComment struct {
	ID          int64  `json:"id"`
	Author      string `json:"author"`
	UserName    string `json:"userName"`
	Text        string `json:"text"`
	PublishedAt string `json:"publishedAt"`
	CreateTime  string `json:"createTime"`
	LikeCount   int    `json:"likeCount"`
}

// FeedPage 动态分页响应（契约：裸数组 或 {data, hasMore, nextPageKey}）。
type FeedPage struct {
	Data        []*FeedPost `json:"data"`
	HasMore     bool        `json:"hasMore"`
	NextPageKey int64       `json:"nextPageKey"`
}

// MessageThread 消息会话（契约 4.4）。
type MessageThread struct {
	ID          int64  `json:"id"`
	MessageID   int64  `json:"messageId"`
	Title       string `json:"title"`
	Subject     string `json:"subject"`
	Preview     string `json:"preview"`
	Content     string `json:"content"`
	UpdatedAt   string `json:"updatedAt"`
	UpdateTime  string `json:"updateTime"`
	CreateTime  string `json:"createTime"`
	UnreadCount int    `json:"unreadCount"`
	Unread      int    `json:"unread"`
	Avatar      string `json:"avatar,omitempty"`
	IsOfficial  bool   `json:"isOfficial"`
	Official    bool   `json:"official"`
}

// TicketOrder 订单条目（契约 4.5）。
type TicketOrder struct {
	OrderID     int64  `json:"orderId"`
	ID          int64  `json:"id"`
	OrderNumber string `json:"orderNumber"`
	ShowID      int64  `json:"showId"`
	SessionID   int64  `json:"sessionId"`
	CreateTime  string `json:"createTime"`
	OrderType   int    `json:"orderType"`
	BuyType     int    `json:"buyType"`
	Status      int    `json:"status"`
	Title       string `json:"title"`
	Poster      string `json:"poster"`
	PayAmount   int64  `json:"payAmount"`
}

// ShippingAddress 收货地址（契约 4.6，兼容多字段别名）。
type ShippingAddress struct {
	ID        int64  `json:"id"`
	AddressID int64  `json:"addressId"`
	Name      string `json:"name"`
	Consignee string `json:"consignee"`
	Mobile    string `json:"mobile"`
	Phone     string `json:"phone"`
	Province  string `json:"province"`
	City      string `json:"city"`
	District  string `json:"district"`
	Region    string `json:"region"`
	Detail    string `json:"detail"`
	Address   string `json:"address"`
	IsDefault bool   `json:"isDefault"`
	Default   bool   `json:"default"`
}

// UserProfile 用户资料（契约 4.6 UserProfileDto）。
type UserProfile struct {
	UserID              int64  `json:"userId"`
	ID                  int64  `json:"id"`
	Nickname            string `json:"nickname"`
	UserName            string `json:"userName"`
	Avatar              string `json:"avatar"`
	HeadImg             string `json:"headImg"`
	Mobile              string `json:"mobile"`
	Phone               string `json:"phone"`
	Level               int    `json:"level"`
	UserLevel           int    `json:"userLevel"`
	LevelLabel          string `json:"levelLabel,omitempty"`
	WannaSeeCount       int    `json:"wannaSeeCount"`
	FollowActivityCount int    `json:"followActivityCount"`
	FollowCount         int    `json:"followCount"`
	FollowUserCount     int    `json:"followUserCount"`
	CouponCount         int    `json:"couponCount"`
	WalletBalance       int64  `json:"walletBalance"`
	Balance             int64  `json:"balance"`
}
