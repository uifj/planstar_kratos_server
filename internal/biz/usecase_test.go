package biz

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// Test doubles.
//
// The large repo interfaces are stubbed by embedding the interface itself: a
// test only overrides the methods its scenario actually calls, and an
// unexpected call panics loudly instead of silently succeeding. The small
// infrastructure ports get explicit function-field fakes with sane defaults.
// Everything lives in package biz, so no external service (MySQL/Redis/ES/MQ)
// is required to run these tests.
// ---------------------------------------------------------------------------

type stubUserRepo struct {
	UserRepo
	byMobile   func(context.Context, string) (*User, error)
	byUsername func(context.Context, string) (*User, error)
	byID       func(context.Context, string) (*User, error)
	create     func(context.Context, *User) (*User, error)
}

func (s *stubUserRepo) FindByMobile(ctx context.Context, m string) (*User, error) {
	if s.byMobile != nil {
		return s.byMobile(ctx, m)
	}
	return nil, ErrUserNotFound
}
func (s *stubUserRepo) FindByUsername(ctx context.Context, u string) (*User, error) {
	if s.byUsername != nil {
		return s.byUsername(ctx, u)
	}
	return nil, ErrUserNotFound
}
func (s *stubUserRepo) FindByID(ctx context.Context, id string) (*User, error) {
	if s.byID != nil {
		return s.byID(ctx, id)
	}
	return nil, ErrUserNotFound
}
func (s *stubUserRepo) Create(ctx context.Context, u *User) (*User, error) {
	if s.create != nil {
		return s.create(ctx, u)
	}
	return u, nil
}

type stubPermRepo struct {
	PermissionRepo
	codes []string
}

func (s *stubPermRepo) ListCodesByRole(context.Context, int32) ([]string, error) {
	return s.codes, nil
}

type stubGoodsRepo struct {
	GoodsRepo
	findSku func(context.Context, string) (*GoodsSku, error)
	list    func(context.Context, GoodsQuery) ([]*Goods, int32, error)
	create  func(context.Context, *Goods) (*Goods, error)
}

func (s *stubGoodsRepo) FindSkuByID(ctx context.Context, id string) (*GoodsSku, error) {
	if s.findSku != nil {
		return s.findSku(ctx, id)
	}
	return nil, ErrGoodsSkuNotFound
}
func (s *stubGoodsRepo) List(ctx context.Context, q GoodsQuery) ([]*Goods, int32, error) {
	if s.list != nil {
		return s.list(ctx, q)
	}
	return nil, 0, nil
}
func (s *stubGoodsRepo) Create(ctx context.Context, g *Goods) (*Goods, error) {
	if s.create != nil {
		return s.create(ctx, g)
	}
	return g, nil
}

type stubCartRepo struct{ CartRepo }
type stubAddressRepo struct {
	AddressRepo
	findByID func(context.Context, string) (*Address, error)
}

func (s *stubAddressRepo) FindByID(ctx context.Context, id string) (*Address, error) {
	if s.findByID != nil {
		return s.findByID(ctx, id)
	}
	return nil, ErrAddressNotFound
}

type stubOrderRepo struct {
	OrderRepo
	findBySn       func(context.Context, string) (*Order, error)
	orderGoods     func(context.Context, string) ([]*OrderGoods, error)
	updateStatusIf func(context.Context, string, int32, int32) (bool, error)
	createOrderTx  func(context.Context, *Order, *OrderAddress, []*OrderGoods, *OutboxEvent) (*Order, error)
}

func (s *stubOrderRepo) FindBySn(ctx context.Context, sn string) (*Order, error) {
	if s.findBySn != nil {
		return s.findBySn(ctx, sn)
	}
	return nil, ErrOrderNotFound
}
func (s *stubOrderRepo) OrderGoods(ctx context.Context, id string) ([]*OrderGoods, error) {
	if s.orderGoods != nil {
		return s.orderGoods(ctx, id)
	}
	return nil, nil
}
func (s *stubOrderRepo) UpdateStatusIf(ctx context.Context, sn string, from, to int32) (bool, error) {
	if s.updateStatusIf != nil {
		return s.updateStatusIf(ctx, sn, from, to)
	}
	return true, nil
}
func (s *stubOrderRepo) CreateOrderTx(ctx context.Context, o *Order, a *OrderAddress, items []*OrderGoods, outbox *OutboxEvent) (*Order, error) {
	if s.createOrderTx != nil {
		return s.createOrderTx(ctx, o, a, items, outbox)
	}
	return o, nil
}

type stubInventoryRepo struct {
	InventoryRepo
	tryLock  func(context.Context, string, []StockItem) error
	avail    func(context.Context, string) (int64, error)
	ensure   func(context.Context, string, int64) error
	confirm  func(context.Context, string, []StockItem) error
	released []string
}

func (s *stubInventoryRepo) TryLock(ctx context.Context, sn string, items []StockItem) error {
	if s.tryLock != nil {
		return s.tryLock(ctx, sn, items)
	}
	return nil
}
func (s *stubInventoryRepo) Available(ctx context.Context, skuID string) (int64, error) {
	if s.avail != nil {
		return s.avail(ctx, skuID)
	}
	return 100, nil
}
func (s *stubInventoryRepo) EnsureSku(ctx context.Context, skuID string, total int64) error {
	if s.ensure != nil {
		return s.ensure(ctx, skuID, total)
	}
	return nil
}
func (s *stubInventoryRepo) ConfirmDeduct(ctx context.Context, sn string, items []StockItem) error {
	if s.confirm != nil {
		return s.confirm(ctx, sn, items)
	}
	return nil
}
func (s *stubInventoryRepo) Release(ctx context.Context, sn string) error {
	s.released = append(s.released, sn)
	return nil
}

type stubPaymentRepo struct {
	PaymentRepo
	create      func(context.Context, *Payment) (*Payment, error)
	findByNo    func(context.Context, string) (*Payment, error)
	markSuccess func(context.Context, string, *OutboxEvent) error
}

func (s *stubPaymentRepo) Create(ctx context.Context, p *Payment) (*Payment, error) {
	if s.create != nil {
		return s.create(ctx, p)
	}
	return p, nil
}
func (s *stubPaymentRepo) FindByNo(ctx context.Context, no string) (*Payment, error) {
	if s.findByNo != nil {
		return s.findByNo(ctx, no)
	}
	return nil, ErrPaymentNotFound
}
func (s *stubPaymentRepo) MarkSuccessWithEvent(ctx context.Context, no string, e *OutboxEvent) error {
	if s.markSuccess != nil {
		return s.markSuccess(ctx, no, e)
	}
	return nil
}

type stubOutboxRepo struct {
	OutboxRepo
	create func(context.Context, *OutboxEvent) (*OutboxEvent, error)
}

func (s *stubOutboxRepo) Create(ctx context.Context, e *OutboxEvent) (*OutboxEvent, error) {
	if s.create != nil {
		return s.create(ctx, e)
	}
	return e, nil
}

// ---- small infrastructure ports ------------------------------------------

type fakeHasher struct {
	hash   func(string) (string, error)
	verify func(string, string) bool
}

func (f fakeHasher) Hash(p string) (string, error) {
	if f.hash != nil {
		return f.hash(p)
	}
	return "hashed:" + p, nil
}
func (f fakeHasher) Verify(h, p string) bool {
	if f.verify != nil {
		return f.verify(h, p)
	}
	return h == "hashed:"+p
}

type fakeTokens struct {
	generate func(context.Context, string, string, int32) (string, int64, error)
	parse    func(context.Context, string) (*Claims, error)
}

func (f fakeTokens) Generate(ctx context.Context, id, nick string, auth int32) (string, int64, error) {
	if f.generate != nil {
		return f.generate(ctx, id, nick, auth)
	}
	return "token:" + id, 999, nil
}
func (f fakeTokens) Parse(ctx context.Context, token string) (*Claims, error) {
	if f.parse != nil {
		return f.parse(ctx, token)
	}
	return nil, errors.New("invalid or expired token")
}

type fakeCaptcha struct {
	create func(context.Context) (string, string, error)
	verify func(context.Context, string, string) (bool, error)
}

func (f fakeCaptcha) Create(ctx context.Context) (string, string, error) {
	if f.create != nil {
		return f.create(ctx)
	}
	return "cid", "data:image/png;base64,AA", nil
}
func (f fakeCaptcha) Verify(ctx context.Context, id, value string) (bool, error) {
	if f.verify != nil {
		return f.verify(ctx, id, value)
	}
	return true, nil
}

type fakeSearch struct {
	ensureErr  error
	search     func(context.Context, GoodsQuery) ([]string, int32, error)
	indexCalls int
}

func (f *fakeSearch) EnsureIndex(context.Context) error { return f.ensureErr }
func (f *fakeSearch) Index(_ context.Context, _ *GoodsDocument) error {
	f.indexCalls++
	return nil
}
func (f *fakeSearch) Search(ctx context.Context, q GoodsQuery) ([]string, int32, error) {
	if f.search != nil {
		return f.search(ctx, q)
	}
	return nil, 0, nil
}
func (f *fakeSearch) UpdateSoldNum(context.Context, string, int64) error { return nil }

type fakeStock struct {
	sets        map[string]int64
	invalidated []string
}

func (f *fakeStock) Get(context.Context, string) (int64, bool, error) { return 0, false, nil }
func (f *fakeStock) Set(_ context.Context, skuID string, n int64) error {
	if f.sets == nil {
		f.sets = map[string]int64{}
	}
	f.sets[skuID] = n
	return nil
}
func (f *fakeStock) Invalidate(_ context.Context, skuID string) error {
	f.invalidated = append(f.invalidated, skuID)
	return nil
}

type fakePublisher struct {
	exchanges []string
	keys      []string
	events    []*DomainEvent
}

func (f *fakePublisher) Publish(_ context.Context, exchange, routingKey string, e *DomainEvent) error {
	f.exchanges = append(f.exchanges, exchange)
	f.keys = append(f.keys, routingKey)
	f.events = append(f.events, e)
	return nil
}

// ---------------------------------------------------------------------------
// UserUsecase
// ---------------------------------------------------------------------------

func TestUserUsecaseRegister(t *testing.T) {
	repo := &stubUserRepo{}
	uc := NewUserUsecase(repo, &stubPermRepo{}, fakeHasher{}, fakeTokens{}, fakeCaptcha{})

	u, token, exp, err := uc.Register(context.Background(), "13800138000", "alice", "pass123456", AuthorityUser)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if u.ID == "" || u.Mobile != "13800138000" || u.Role != AuthorityUser || u.Status != RowActive {
		t.Fatalf("unexpected user DO: %+v", u)
	}
	if u.PasswordHash != "hashed:pass123456" {
		t.Fatalf("password was not hashed: %q", u.PasswordHash)
	}
	if token != "token:"+u.ID || exp != 999 {
		t.Fatalf("unexpected token: %q exp=%d", token, exp)
	}
}

func TestUserUsecaseRegisterRejectsInvalidInput(t *testing.T) {
	uc := NewUserUsecase(&stubUserRepo{}, &stubPermRepo{}, fakeHasher{}, fakeTokens{}, fakeCaptcha{})
	for _, tc := range []struct {
		name, mobile, username, password string
	}{
		{"bad mobile", "123", "alice", "pass123456"},
		{"empty username", "13800138000", "", "pass123456"},
		{"short password", "13800138000", "alice", "short"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, _, err := uc.Register(context.Background(), tc.mobile, tc.username, tc.password, AuthorityUser); err != ErrUserInvalidArgument {
				t.Fatalf("err = %v, want ErrUserInvalidArgument", err)
			}
		})
	}
}

func TestUserUsecaseRegisterDuplicateMobile(t *testing.T) {
	repo := &stubUserRepo{byMobile: func(context.Context, string) (*User, error) {
		return &User{ID: "existing"}, nil
	}}
	uc := NewUserUsecase(repo, &stubPermRepo{}, fakeHasher{}, fakeTokens{}, fakeCaptcha{})
	if _, _, _, err := uc.Register(context.Background(), "13800138000", "alice", "pass123456", AuthorityUser); err != ErrUserMobileExists {
		t.Fatalf("err = %v, want ErrUserMobileExists", err)
	}
}

// Regression for source defect #1: CreateUser / repo errors must never be
// swallowed. Every error from a port has to reach the caller unchanged.
func TestUserUsecaseRegisterPropagatesPortErrors(t *testing.T) {
	boom := errors.New("db down")
	ctx := context.Background()

	t.Run("lookup error", func(t *testing.T) {
		repo := &stubUserRepo{byMobile: func(context.Context, string) (*User, error) { return nil, boom }}
		uc := NewUserUsecase(repo, &stubPermRepo{}, fakeHasher{}, fakeTokens{}, fakeCaptcha{})
		if _, _, _, err := uc.Register(ctx, "13800138000", "alice", "pass123456", AuthorityUser); err != boom {
			t.Fatalf("err = %v, want %v", err, boom)
		}
	})
	t.Run("hash error", func(t *testing.T) {
		uc := NewUserUsecase(&stubUserRepo{}, &stubPermRepo{},
			fakeHasher{hash: func(string) (string, error) { return "", boom }}, fakeTokens{}, fakeCaptcha{})
		if _, _, _, err := uc.Register(ctx, "13800138000", "alice", "pass123456", AuthorityUser); err != boom {
			t.Fatalf("err = %v, want %v", err, boom)
		}
	})
	t.Run("create error", func(t *testing.T) {
		repo := &stubUserRepo{create: func(context.Context, *User) (*User, error) { return nil, boom }}
		uc := NewUserUsecase(repo, &stubPermRepo{}, fakeHasher{}, fakeTokens{}, fakeCaptcha{})
		if _, _, _, err := uc.Register(ctx, "13800138000", "alice", "pass123456", AuthorityUser); err != boom {
			t.Fatalf("err = %v, want %v", err, boom)
		}
	})
}

func TestUserUsecaseLogin(t *testing.T) {
	ctx := context.Background()
	user := &User{ID: "u1", NickName: "alice", PasswordHash: "hashed:right", Role: AuthorityUser}
	repo := &stubUserRepo{
		byMobile:   func(context.Context, string) (*User, error) { return user, nil },
		byUsername: func(context.Context, string) (*User, error) { return user, nil },
	}
	uc := NewUserUsecase(repo, &stubPermRepo{}, fakeHasher{}, fakeTokens{}, fakeCaptcha{})

	t.Run("wrong password", func(t *testing.T) {
		if _, _, _, err := uc.Login(ctx, "13800138000", "", "wrong", "", "", false); err != ErrUserPasswordInvalid {
			t.Fatalf("err = %v, want ErrUserPasswordInvalid", err)
		}
	})
	t.Run("admin path skips captcha", func(t *testing.T) {
		_, token, _, err := uc.Login(ctx, "", "admin", "right", "", "", false)
		if err != nil || token == "" {
			t.Fatalf("Login(admin) = %q, %v", token, err)
		}
	})
	t.Run("captcha required on C-end", func(t *testing.T) {
		cap := fakeCaptcha{verify: func(context.Context, string, string) (bool, error) { return false, nil }}
		uc2 := NewUserUsecase(repo, &stubPermRepo{}, fakeHasher{}, fakeTokens{}, cap)
		if _, _, _, err := uc2.Login(ctx, "13800138000", "", "right", "bad", "cid", true); err != ErrCaptchaInvalid {
			t.Fatalf("err = %v, want ErrCaptchaInvalid", err)
		}
	})
}

// Regression for source defect #6: RefreshToken must reject a token whose
// full claim set (including expiry) failed validation.
func TestUserUsecaseRefreshTokenValidatesClaims(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("token expired")

	ucBad := NewUserUsecase(&stubUserRepo{}, &stubPermRepo{}, fakeHasher{},
		fakeTokens{parse: func(context.Context, string) (*Claims, error) { return nil, boom }}, fakeCaptcha{})
	if _, _, _, err := ucBad.RefreshToken(ctx, "expired"); err != ErrTokenInvalid {
		t.Fatalf("err = %v, want ErrTokenInvalid", err)
	}

	repo := &stubUserRepo{byID: func(context.Context, string) (*User, error) {
		return &User{ID: "u1", NickName: "alice", Role: AuthorityUser}, nil
	}}
	ucOK := NewUserUsecase(repo, &stubPermRepo{}, fakeHasher{},
		fakeTokens{parse: func(context.Context, string) (*Claims, error) {
			return &Claims{ID: "u1", NickName: "alice", AuthorityID: AuthorityUser}, nil
		}}, fakeCaptcha{})
	_, token, _, err := ucOK.RefreshToken(ctx, "valid")
	if err != nil || token != "token:u1" {
		t.Fatalf("RefreshToken = %q, %v", token, err)
	}
}

// ---------------------------------------------------------------------------
// CartUsecase.BuildOrderLines - prices are re-read from the catalogue.
// ---------------------------------------------------------------------------

func TestCartUsecaseBuildOrderLinesRevalidatesPrice(t *testing.T) {
	ctx := context.Background()
	goods := &stubGoodsRepo{findSku: func(context.Context, string) (*GoodsSku, error) {
		return &GoodsSku{ID: "sku1", GoodsID: "g1", SkuName: "SKU-A", SkuCode: "C1", Price: 1999, OnSale: true}, nil
	}}
	cart := NewCartUsecase(&stubCartRepo{}, goods)

	lines, stock, total, err := cart.BuildOrderLines(ctx, "u1", []OrderItemInput{{SkuID: "sku1", Num: 2}})
	if err != nil {
		t.Fatalf("BuildOrderLines: %v", err)
	}
	if total != 3998 {
		t.Fatalf("total = %d, want 3998 (2 x 1999)", total)
	}
	if len(lines) != 1 || lines[0].SkuPrice != 1999 || lines[0].Num != 2 || lines[0].TotalPrice != 3998 {
		t.Fatalf("unexpected lines: %+v", lines)
	}
	if len(stock) != 1 || stock[0].SkuID != "sku1" || stock[0].Num != 2 {
		t.Fatalf("unexpected stock items: %+v", stock)
	}
}

func TestCartUsecaseBuildOrderLinesRejectsBadInput(t *testing.T) {
	ctx := context.Background()
	cart := NewCartUsecase(&stubCartRepo{}, &stubGoodsRepo{})
	if _, _, _, err := cart.BuildOrderLines(ctx, "u1", nil); err != ErrOrderInvalidArgument {
		t.Fatalf("empty inputs: err = %v", err)
	}
	if _, _, _, err := cart.BuildOrderLines(ctx, "u1", []OrderItemInput{{SkuID: "sku1", Num: 0}}); err != ErrOrderInvalidArgument {
		t.Fatalf("zero num: err = %v", err)
	}
	offSale := NewCartUsecase(&stubCartRepo{}, &stubGoodsRepo{findSku: func(context.Context, string) (*GoodsSku, error) {
		return &GoodsSku{ID: "sku1", Price: 100, OnSale: false}, nil
	}})
	if _, _, _, err := offSale.BuildOrderLines(ctx, "u1", []OrderItemInput{{SkuID: "sku1", Num: 1}}); err != ErrGoodsNotFound {
		t.Fatalf("off-sale sku: err = %v, want ErrGoodsNotFound", err)
	}
}

// ---------------------------------------------------------------------------
// PaymentUsecase - typed status constants (source defect #4).
// ---------------------------------------------------------------------------

func TestPaymentUsecaseCreate(t *testing.T) {
	ctx := context.Background()
	base := &Order{ID: "o1", OrderSn: "SN1", UserID: "u1", Total: 3998, Status: OrderStatusLocked}
	orders := &stubOrderRepo{findBySn: func(context.Context, string) (*Order, error) { return base, nil }}
	pay := &stubPaymentRepo{}
	uc := NewPaymentUsecase(pay, orders)

	p, err := uc.Create(ctx, "u1", "SN1", 3998, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.Status != PaymentStatusPending || p.Channel != "mock" || p.Amount != 3998 {
		t.Fatalf("unexpected payment: %+v", p)
	}

	if _, err := uc.Create(ctx, "u1", "SN1", 1, ""); err != ErrPaymentAmountMismatch {
		t.Fatalf("amount mismatch: err = %v", err)
	}
	if _, err := uc.Create(ctx, "u1", "SN1", 0, ""); err != ErrOrderInvalidArgument {
		t.Fatalf("zero amount: err = %v", err)
	}
	if _, err := uc.Create(ctx, "other", "SN1", 3998, ""); err != ErrOrderNotFound {
		t.Fatalf("wrong user: err = %v", err)
	}
	base.Status = OrderStatusPending
	if _, err := uc.Create(ctx, "u1", "SN1", 3998, ""); err != ErrOrderStatusConflict {
		t.Fatalf("wrong order status: err = %v", err)
	}
}

func TestPaymentUsecaseMockPayWritesOutboxEvent(t *testing.T) {
	ctx := context.Background()
	var captured *OutboxEvent
	pay := &stubPaymentRepo{
		findByNo: func(context.Context, string) (*Payment, error) {
			return &Payment{PaymentNo: "P1", OrderSn: "SN1", UserID: "u1", Amount: 3998, Status: PaymentStatusPending}, nil
		},
		markSuccess: func(_ context.Context, _ string, e *OutboxEvent) error {
			captured = e
			return nil
		},
	}
	orders := &stubOrderRepo{
		findBySn: func(context.Context, string) (*Order, error) {
			return &Order{ID: "o1", OrderSn: "SN1", UserID: "u1", Total: 3998, Status: OrderStatusLocked}, nil
		},
		orderGoods: func(context.Context, string) ([]*OrderGoods, error) {
			return []*OrderGoods{{SkuID: "sku1", SkuName: "SKU-A", SkuPrice: 1999, Num: 2}}, nil
		},
	}
	uc := NewPaymentUsecase(pay, orders)
	if err := uc.MockPay(ctx, "u1", "P1"); err != nil {
		t.Fatalf("MockPay: %v", err)
	}
	if captured == nil || captured.EventType != EventPaymentSuccess || captured.Status != OutboxPending {
		t.Fatalf("unexpected outbox row: %+v", captured)
	}
	ev, err := decodeEvent(captured.Payload)
	if err != nil {
		t.Fatalf("decodeEvent: %v", err)
	}
	if ev.Type != EventPaymentSuccess || ev.PaymentNo != "P1" || ev.Amount != 3998 || len(ev.Items) != 1 || ev.Items[0].Num != 2 {
		t.Fatalf("unexpected event payload: %+v", ev)
	}
}

// ---------------------------------------------------------------------------
// OrderUsecase saga handlers - CAS transitions + order.paid outbox.
// ---------------------------------------------------------------------------

func TestOrderUsecaseSagaHandlers(t *testing.T) {
	ctx := context.Background()
	type transition struct{ from, to int32 }
	var transitions []transition
	var created []*OutboxEvent

	orders := &stubOrderRepo{
		findBySn: func(context.Context, string) (*Order, error) {
			return &Order{ID: "o1", OrderSn: "SN1", Status: OrderStatusLocked}, nil
		},
		updateStatusIf: func(_ context.Context, _ string, from, to int32) (bool, error) {
			transitions = append(transitions, transition{from, to})
			return true, nil
		},
	}
	outbox := &stubOutboxRepo{create: func(_ context.Context, e *OutboxEvent) (*OutboxEvent, error) {
		created = append(created, e)
		return e, nil
	}}
	uc := NewOrderUsecase(orders, &stubAddressRepo{},
		NewCartUsecase(&stubCartRepo{}, &stubGoodsRepo{}), outbox, &stubInventoryRepo{})

	ev := &DomainEvent{OrderSn: "SN1", UserID: "u1", Amount: 3998, PaymentNo: "P1",
		Items: []EventItem{{SkuID: "sku1", Num: 2}}}
	if err := uc.HandleInventoryLocked(ctx, ev); err != nil {
		t.Fatalf("HandleInventoryLocked: %v", err)
	}
	if err := uc.HandleInventoryFailed(ctx, ev); err != nil {
		t.Fatalf("HandleInventoryFailed: %v", err)
	}
	if err := uc.HandlePaymentSuccess(ctx, ev); err != nil {
		t.Fatalf("HandlePaymentSuccess: %v", err)
	}

	wantTransitions := []transition{
		{OrderStatusPending, OrderStatusLocked},
		{OrderStatusPending, OrderStatusCancelled},
		{OrderStatusLocked, OrderStatusPaid},
	}
	if !reflect.DeepEqual(transitions, wantTransitions) {
		t.Fatalf("transitions = %+v, want %+v", transitions, wantTransitions)
	}
	if len(created) != 1 || created[0].EventType != EventOrderPaid {
		t.Fatalf("expected one order.paid outbox row, got %+v", created)
	}
	paid, err := decodeEvent(created[0].Payload)
	if err != nil || paid.PaymentNo != "P1" || len(paid.Items) != 1 {
		t.Fatalf("order.paid payload = %+v, %v", paid, err)
	}
}

func TestOrderUsecaseHandlePaymentSuccessIsIdempotent(t *testing.T) {
	created := 0
	orders := &stubOrderRepo{findBySn: func(context.Context, string) (*Order, error) {
		return &Order{ID: "o1", OrderSn: "SN1", Status: OrderStatusPaid}, nil
	}}
	outbox := &stubOutboxRepo{create: func(_ context.Context, e *OutboxEvent) (*OutboxEvent, error) {
		created++
		return e, nil
	}}
	uc := NewOrderUsecase(orders, &stubAddressRepo{},
		NewCartUsecase(&stubCartRepo{}, &stubGoodsRepo{}), outbox, &stubInventoryRepo{})
	if err := uc.HandlePaymentSuccess(context.Background(), &DomainEvent{OrderSn: "SN1"}); err != nil {
		t.Fatalf("HandlePaymentSuccess: %v", err)
	}
	if created != 0 {
		t.Fatalf("already-paid order created %d outbox rows, want 0", created)
	}
}

// ---------------------------------------------------------------------------
// InventoryUsecase - Saga lock result events.
// ---------------------------------------------------------------------------

func TestInventoryUsecaseHandleOrderCreated(t *testing.T) {
	ctx := context.Background()
	event := &DomainEvent{OrderSn: "SN1", UserID: "u1", Items: []EventItem{{SkuID: "sku1", Num: 2}}}

	t.Run("lock success publishes inventory.lock.success", func(t *testing.T) {
		pub := &fakePublisher{}
		inv := &stubInventoryRepo{avail: func(context.Context, string) (int64, error) { return 8, nil }}
		uc := NewInventoryUsecase(inv, &fakeStock{}, pub)
		if err := uc.HandleOrderCreated(ctx, event); err != nil {
			t.Fatalf("HandleOrderCreated: %v", err)
		}
		if len(pub.keys) != 1 || pub.keys[0] != EventInventoryLockOK || pub.exchanges[0] != ExchangeInventory {
			t.Fatalf("unexpected publish: %+v / %+v", pub.exchanges, pub.keys)
		}
	})
	t.Run("lock failure publishes inventory.lock.failed with reason", func(t *testing.T) {
		pub := &fakePublisher{}
		inv := &stubInventoryRepo{tryLock: func(context.Context, string, []StockItem) error { return ErrInventoryNotEnough }}
		uc := NewInventoryUsecase(inv, &fakeStock{}, pub)
		if err := uc.HandleOrderCreated(ctx, event); err != nil {
			t.Fatalf("HandleOrderCreated: %v", err)
		}
		if len(pub.keys) != 1 || pub.keys[0] != EventInventoryLockFailed {
			t.Fatalf("unexpected publish: %+v", pub.keys)
		}
		if pub.events[0].Reason != ErrInventoryNotEnough.Error() {
			t.Fatalf("reason = %q", pub.events[0].Reason)
		}
	})
}

// ---------------------------------------------------------------------------
// GoodsUsecase - ES read path, SQL fallback, aggregate child defaults.
// ---------------------------------------------------------------------------

func TestGoodsUsecaseListUsesElasticsearchHits(t *testing.T) {
	var gotIDs []string
	var gotOnlyOnSale bool
	search := &fakeSearch{search: func(context.Context, GoodsQuery) ([]string, int32, error) {
		return []string{"g2", "g1"}, 7, nil
	}}
	goods := &stubGoodsRepo{list: func(_ context.Context, q GoodsQuery) ([]*Goods, int32, error) {
		gotIDs = q.IDs
		gotOnlyOnSale = q.OnlyOnSale
		return []*Goods{{ID: "g2"}, {ID: "g1"}}, 2, nil
	}}
	uc := NewGoodsUsecase(goods, search, &stubInventoryRepo{}, &fakeStock{})
	list, total, err := uc.List(context.Background(), GoodsQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 7 || len(list) != 2 {
		t.Fatalf("total=%d len=%d", total, len(list))
	}
	if !reflect.DeepEqual(gotIDs, []string{"g2", "g1"}) || !gotOnlyOnSale {
		t.Fatalf("repo query: IDs=%v onlyOnSale=%v", gotIDs, gotOnlyOnSale)
	}
}

func TestGoodsUsecaseListFallsBackToSQL(t *testing.T) {
	for _, tc := range []struct {
		name   string
		search func(context.Context, GoodsQuery) ([]string, int32, error)
	}{
		{"ES error", func(context.Context, GoodsQuery) ([]string, int32, error) {
			return nil, 0, errors.New("es down")
		}},
		{"ES empty", func(context.Context, GoodsQuery) ([]string, int32, error) {
			return nil, 0, nil
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var gotIDs []string
			goods := &stubGoodsRepo{list: func(_ context.Context, q GoodsQuery) ([]*Goods, int32, error) {
				gotIDs = q.IDs
				return []*Goods{{ID: "g1"}}, 1, nil
			}}
			uc := NewGoodsUsecase(goods, &fakeSearch{search: tc.search}, &stubInventoryRepo{}, &fakeStock{})
			list, total, err := uc.List(context.Background(), GoodsQuery{})
			if err != nil || total != 1 || len(list) != 1 {
				t.Fatalf("List = %v, %d, %v", list, total, err)
			}
			if gotIDs != nil {
				t.Fatalf("SQL fallback must clear IDs, got %v", gotIDs)
			}
		})
	}
}

func TestGoodsUsecaseCreateAssignsChildDefaults(t *testing.T) {
	var ensured []string
	goods := &stubGoodsRepo{}
	inv := &stubInventoryRepo{ensure: func(_ context.Context, skuID string, total int64) error {
		ensured = append(ensured, skuID)
		return nil
	}}
	search := &fakeSearch{}
	uc := NewGoodsUsecase(goods, search, inv, &fakeStock{})

	g := &Goods{
		Name: "Phone", GoodsSn: "SN-1",
		Skus:   []*GoodsSku{{SkuName: "128G", Inventory: 50}},
		Images: []*GoodsImage{{URL: "https://example.com/a.png"}},
	}
	if err := uc.Create(context.Background(), g); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if g.ID == "" || g.Status != RowActive {
		t.Fatalf("aggregate defaults: %+v", g)
	}
	sku := g.Skus[0]
	if sku.ID == "" || sku.GoodsID != g.ID || sku.SkuCode == "" || sku.Status != RowActive {
		t.Fatalf("sku defaults: %+v", sku)
	}
	im := g.Images[0]
	if im.ID == "" || im.GoodsID != g.ID || im.Status != RowActive {
		t.Fatalf("image defaults: %+v", im)
	}
	if !reflect.DeepEqual(ensured, []string{sku.ID}) {
		t.Fatalf("EnsureSku calls = %v", ensured)
	}
	if search.indexCalls != 1 {
		t.Fatalf("index calls = %d, want 1", search.indexCalls)
	}
}

func TestGoodsUsecaseCreateRejectsInvalidInput(t *testing.T) {
	uc := NewGoodsUsecase(&stubGoodsRepo{}, &fakeSearch{}, &stubInventoryRepo{}, &fakeStock{})
	if err := uc.Create(context.Background(), nil); err != ErrGoodsInvalidArgument {
		t.Fatalf("nil: err = %v", err)
	}
	if err := uc.Create(context.Background(), &Goods{Name: "x"}); err != ErrGoodsInvalidArgument {
		t.Fatalf("missing goods_sn: err = %v", err)
	}
}
