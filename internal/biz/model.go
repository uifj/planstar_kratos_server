package biz

import "time"

// All ids in the mall are application-generated UUIDv7 strings. Business keys
// (order_sn, payment_no, goods_sn, sku_code, mobile, code, event_id) stay
// unique strings. Prices are int64 in cents; timestamps are unix seconds
// where the storage column is an int, time.Time where it comes from a mixin.

// Order lifecycle. Values match api/mall/v1/common.proto OrderStatus.
const (
	OrderStatusUnspecified int32 = 0
	OrderStatusPending     int32 = 1 // created, waiting for inventory lock
	OrderStatusLocked      int32 = 2 // inventory locked, waiting for payment
	OrderStatusPaid        int32 = 3 // paid, waiting for shipment
	OrderStatusShipped     int32 = 4
	OrderStatusCompleted   int32 = 5
	OrderStatusCancelled   int32 = 6 // inventory failure, user cancel, timeout
	OrderStatusRefunding   int32 = 7
	OrderStatusRefunded    int32 = 8
)

// Payment lifecycle. Values match PaymentStatus in common.proto.
const (
	PaymentStatusPending  int32 = 1
	PaymentStatusSuccess  int32 = 2
	PaymentStatusFailed   int32 = 3
	PaymentStatusRefunded int32 = 4
)

// Row lifecycle for soft-deletable tables. Orders/payments/outbox use their
// own state column instead.
const (
	RowActive  int32 = 1
	RowDeleted int32 = 2
)

// Outbox state.
const (
	OutboxPending   int32 = 0
	OutboxPublished int32 = 1
	OutboxFailed    int32 = 2
)

// Inventory lock state.
const (
	LockHeld      int32 = 1
	LockConfirmed int32 = 2
	LockReleased  int32 = 3
)

// Inventory flow type.
const (
	FlowLock    int32 = 1
	FlowConfirm int32 = 2
	FlowRelease int32 = 3
)

// ---------------------------------------------------------------------------
// User / address
// ---------------------------------------------------------------------------

type User struct {
	ID           string
	Mobile       string
	Username     string
	PasswordHash string
	NickName     string
	Birthday     int64
	Gender       string
	Role         int32
	Status       int32
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Address struct {
	ID        string
	UserID    string
	Name      string
	Mobile    string
	Province  string
	City      string
	Districts string
	Address   string
	PostCode  string
	IsDefault int32
	Status    int32
}

// ---------------------------------------------------------------------------
// Catalogue
// ---------------------------------------------------------------------------

type Category struct {
	ID             string
	Name           string
	ParentCategory string
	Level          int32
	Sort           int32
	Status         int32
}

type Brand struct {
	ID     string
	Name   string
	Logo   string
	Status int32
}

type Goods struct {
	ID              string
	CategoryID      string
	BrandID         string
	TypeID          string
	Name            string
	NameAlias       string
	GoodsSn         string
	GoodsBrief      string
	GoodsFrontImage string
	MarketPrice     int64
	OnSale          bool
	ShipFree        bool
	IsNew           bool
	IsHot           bool
	SoldNum         int64
	Status          int32
	Skus            []*GoodsSku
	Images          []*GoodsImage
	CategoryName    string
	BrandName       string
}

type GoodsSku struct {
	ID             string
	GoodsID        string
	SkuName        string
	SkuCode        string
	BarCode        string
	Price          int64
	PromotionPrice int64
	Inventory      int64
	OnSale         bool
	Pic            string
	Specs          map[string]any
	Status         int32
}

type GoodsImage struct {
	ID       string
	GoodsID  string
	URL      string
	Position int32
	IsMaster bool
	Status   int32
}

// GoodsDocument is the shape indexed into Elasticsearch. It is a DO, not a
// DTO: data converts it into the ES JSON document.
type GoodsDocument struct {
	ID          string
	Name        string
	GoodsSn     string
	MarketPrice int64
	OnSale      bool
	IsNew       bool
	IsHot       bool
	CategoryID  string
	BrandID     string
	CategoryName string
	BrandName   string
	SoldNum     int64
	SkuCodes    []string
	SkuIDs      []string
}

// GoodsQuery is the C-end / admin goods list filter.
type GoodsQuery struct {
	Page       int
	PageSize   int
	Keywords   string
	CategoryID string
	BrandID    string
	SkuCode    string
	OnlyOnSale bool
	// IDs is set by GoodsUsecase after an Elasticsearch search so the repo can
	// hydrate exactly the winning documents without re-paginating.
	IDs []string
}

// ---------------------------------------------------------------------------
// Cart
// ---------------------------------------------------------------------------

type CartItem struct {
	ID        string
	UserID    string
	GoodsID   string
	GoodsSn   string
	GoodsName string
	SkuID     string
	GoodsPrice int64
	GoodsNum  int32
	IsSelect  bool
	Status    int32
}

// ---------------------------------------------------------------------------
// Order
// ---------------------------------------------------------------------------

type Order struct {
	ID       string
	UserID   string
	OrderSn  string
	Status   int32
	Post     string
	Total    int64
	Address  string
	Name     string
	Mobile   string
	AddTime  int64
	PayTime  int64
	Goods    []*OrderGoods
	Address2 *OrderAddress
}

type OrderAddress struct {
	ID        string
	OrderID   string
	Name      string
	Mobile    string
	Province  string
	City      string
	Districts string
	Address   string
	PostCode  string
}

type OrderGoods struct {
	ID         string
	OrderID    string
	SkuID      string
	SkuName    string
	SkuPrice   int64
	Num        int32
	TotalPrice int64
}

// OrderItemInput is the client's requested line, before server-side price
// re-validation. Prices are NEVER trusted from the client.
type OrderItemInput struct {
	CartID string
	SkuID  string
	Num    int32
}

// ---------------------------------------------------------------------------
// Payment
// ---------------------------------------------------------------------------

type Payment struct {
	ID        string
	PaymentNo string
	OrderSn   string
	UserID    string
	Amount    int64
	Channel   string
	Status    int32
}

// ---------------------------------------------------------------------------
// Inventory
// ---------------------------------------------------------------------------

type Inventory struct {
	ID      string
	SkuID   string
	Total   int64
	Locked  int64
	Sold    int64
	Version int64
}

type InventoryLock struct {
	ID      string
	OrderSn string
	SkuID   string
	Num     int32
	Status  int32
}

type InventoryFlow struct {
	ID       string
	SkuID    string
	OrderSn  string
	Num      int32
	FlowType int32
}

// StockItem is one line of an inventory lock attempt.
type StockItem struct {
	SkuID string
	Num   int32
}

// ---------------------------------------------------------------------------
// Outbox / idempotency / permissions
// ---------------------------------------------------------------------------

type OutboxEvent struct {
	ID         string
	EventID    string
	EventType  string
	OrderSn    string
	Payload    string
	Status     int32
	RetryCount int32
}

type Permission struct {
	ID        string
	Code      string
	Name      string
	GroupName string
	Sort      int32
	Status    int32
}

type RolePermission struct {
	ID             string
	Role           int32
	PermissionCode string
	Status         int32
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

type DashboardStats struct {
	TotalOrders int64
	TotalSales  int64
	TodayOrders int64
	TodaySales  int64
	TotalUsers  int64
	StatusCounts []StatusCount
	Last30Days   []DailySales
	TopGoods     []TopGoods
}

type StatusCount struct {
	Status int32
	Count  int64
}

type DailySales struct {
	Date       string
	OrderCount int64
	Amount     int64
}

type TopGoods struct {
	SkuID   string
	SkuName string
	Num     int64
	Amount  int64
}
