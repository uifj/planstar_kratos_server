package biz

import "context"

// ---------------------------------------------------------------------------
// Repo interfaces (dependency inversion seam). data implements every one of
// these; biz never imports ent, redis, elastic or amqp.
// ---------------------------------------------------------------------------

type UserRepo interface {
	Create(ctx context.Context, u *User) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	FindByMobile(ctx context.Context, mobile string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	List(ctx context.Context, page, pageSize int) ([]*User, int32, error)
	Count(ctx context.Context) (int64, error)
}

type AddressRepo interface {
	Create(ctx context.Context, a *Address) (*Address, error)
	FindByID(ctx context.Context, id string) (*Address, error)
	ListByUser(ctx context.Context, userID string) ([]*Address, error)
	Update(ctx context.Context, a *Address) (*Address, error)
	SetDefault(ctx context.Context, userID, id string) error
	Delete(ctx context.Context, userID, id string) error
	DeleteByIDForAdmin(ctx context.Context, id string) error
	ListByUserForAdmin(ctx context.Context, userID string) ([]*Address, error)
}

type PermissionRepo interface {
	List(ctx context.Context) ([]*Permission, error)
	ListCodesByRole(ctx context.Context, role int32) ([]string, error)
	ReplaceRole(ctx context.Context, role int32, codes []string) error
}

type GoodsRepo interface {
	Create(ctx context.Context, g *Goods) (*Goods, error)
	FindByID(ctx context.Context, id string) (*Goods, error)
	FindSkuByID(ctx context.Context, id string) (*GoodsSku, error)
	List(ctx context.Context, q GoodsQuery) ([]*Goods, int32, error)
	Update(ctx context.Context, g *Goods) error
	SetOnSale(ctx context.Context, id string, onSale bool) error
	Delete(ctx context.Context, id string) error
	AllForIndex(ctx context.Context) ([]*Goods, error)
	IncrSoldNum(ctx context.Context, skuID string, delta int64) error
	Categories(ctx context.Context) ([]*Category, error)
	Brands(ctx context.Context, page, pageSize int) ([]*Brand, error)
	CreateCategory(ctx context.Context, c *Category) (*Category, error)
	UpdateCategory(ctx context.Context, c *Category) error
	DeleteCategory(ctx context.Context, id string) error
	CategoryName(ctx context.Context, id string) string
	BrandName(ctx context.Context, id string) string
}

type CartRepo interface {
	Add(ctx context.Context, c *CartItem) (*CartItem, error)
	ListByUser(ctx context.Context, userID string) ([]*CartItem, error)
	FindByID(ctx context.Context, id string) (*CartItem, error)
	UpdateNum(ctx context.Context, userID, id string, num int32, selected *bool) error
	Delete(ctx context.Context, userID, id string) error
	DeleteMany(ctx context.Context, userID string, ids []string) error
}

type OrderRepo interface {
	// CreateOrderTx writes order, address snapshot, line items and the outbox
	// row in one database transaction. biz never sees *ent.Tx.
	CreateOrderTx(ctx context.Context, o *Order, a *OrderAddress, items []*OrderGoods, outbox *OutboxEvent) (*Order, error)
	FindBySn(ctx context.Context, orderSn string) (*Order, error)
	FindByID(ctx context.Context, id string) (*Order, error)
	List(ctx context.Context, userID string, status, page, pageSize int) ([]*Order, int32, error)
	UpdateStatusIf(ctx context.Context, orderSn string, from, to int32) (bool, error)
	SetPost(ctx context.Context, orderSn, post string) error
	ListTimedOut(ctx context.Context, before int64, limit int) ([]*Order, error)
	Dashboard(ctx context.Context) (*DashboardStats, error)
	OrderGoods(ctx context.Context, orderID string) ([]*OrderGoods, error)
	OrderAddress(ctx context.Context, orderID string) (*OrderAddress, error)
}

type InventoryRepo interface {
	// TryLock uses a row lock (FOR UPDATE) plus a version CAS. It returns
	// ErrInventoryNotEnough when any line cannot be satisfied.
	TryLock(ctx context.Context, orderSn string, items []StockItem) error
	ConfirmDeduct(ctx context.Context, orderSn string, items []StockItem) error
	Release(ctx context.Context, orderSn string) error
	EnsureSku(ctx context.Context, skuID string, total int64) error
	Available(ctx context.Context, skuID string) (int64, error)
}

type PaymentRepo interface {
	Create(ctx context.Context, p *Payment) (*Payment, error)
	FindByNo(ctx context.Context, paymentNo string) (*Payment, error)
	// MarkSuccessWithEvent flips the payment and writes the payment.success
	// outbox row in one database transaction.
	MarkSuccessWithEvent(ctx context.Context, paymentNo string, outbox *OutboxEvent) error
}

type OutboxRepo interface {
	ListPending(ctx context.Context, limit int) ([]*OutboxEvent, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string) error
	IncrRetry(ctx context.Context, id string) error
	Create(ctx context.Context, e *OutboxEvent) (*OutboxEvent, error)
}

type ConsumedEventRepo interface {
	Exists(ctx context.Context, eventID, consumer string) (bool, error)
	Mark(ctx context.Context, eventID, consumer string) error
}

// ---------------------------------------------------------------------------
// Infrastructure ports. data implements them with the long-lived clients it
// already owns; biz declares only the behaviour it needs.
// ---------------------------------------------------------------------------

type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(hash, password string) bool
}

type TokenManager interface {
	Generate(ctx context.Context, userID, nickName string, authorityID int32) (token string, expiresAt int64, err error)
	Parse(ctx context.Context, token string) (*Claims, error)
}

type CaptchaStore interface {
	// Create returns the captcha id and a base64 PNG data URL.
	Create(ctx context.Context) (id, image string, err error)
	Verify(ctx context.Context, id, value string) (bool, error)
}

type GoodsSearch interface {
	EnsureIndex(ctx context.Context) error
	Index(ctx context.Context, doc *GoodsDocument) error
	Search(ctx context.Context, q GoodsQuery) ([]string, int32, error)
	UpdateSoldNum(ctx context.Context, skuID string, delta int64) error
}

type StockCache interface {
	Get(ctx context.Context, skuID string) (int64, bool, error)
	Set(ctx context.Context, skuID string, n int64) error
	Invalidate(ctx context.Context, skuID string) error
}
