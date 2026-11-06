package biz

import (
	"context"
	"errors"
	"time"
)

// OrderUsecase owns the order lifecycle. Create persists the order, its
// address snapshot, its line items and the order.created outbox row in one
// transaction, so the event can never be lost between the write and the
// publisher.
type OrderUsecase struct {
	repo      OrderRepo
	address   AddressRepo
	cart      *CartUsecase
	outbox    OutboxRepo
	inventory InventoryRepo
}

func NewOrderUsecase(repo OrderRepo, address AddressRepo, cart *CartUsecase, outbox OutboxRepo, inventory InventoryRepo) *OrderUsecase {
	return &OrderUsecase{repo: repo, address: address, cart: cart, outbox: outbox, inventory: inventory}
}

func (uc *OrderUsecase) Create(ctx context.Context, userID, addressID string, inputs []OrderItemInput) (*Order, error) {
	addr, err := uc.address.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}
	if addr.UserID != userID {
		return nil, ErrAddressNotFound
	}
	lines, stock, total, err := uc.cart.BuildOrderLines(ctx, userID, inputs)
	if err != nil {
		return nil, err
	}
	if total <= 0 {
		return nil, ErrOrderPriceChanged
	}
	o := &Order{
		ID: newID(), UserID: userID, OrderSn: newOrderSn(),
		Status: OrderStatusPending, Total: total,
		Address: addr.Province + addr.City + addr.Districts + addr.Address,
		Name: addr.Name, Mobile: addr.Mobile, AddTime: time.Now().Unix(),
	}
	snapshot := &OrderAddress{
		ID: newID(), Name: addr.Name, Mobile: addr.Mobile,
		Province: addr.Province, City: addr.City, Districts: addr.Districts,
		Address: addr.Address, PostCode: addr.PostCode,
	}
	event := newDomainEvent(EventOrderCreated, o.OrderSn, userID)
	event.Amount = total
	for _, s := range stock {
		line := findLine(lines, s.SkuID)
		if line != nil {
			event.Items = append(event.Items, EventItem{
				SkuID: s.SkuID, SkuName: line.SkuName, Price: line.SkuPrice, Num: s.Num,
			})
		}
	}
	outbox := &OutboxEvent{
		ID: newID(), EventID: event.ID, EventType: EventOrderCreated,
		OrderSn: o.OrderSn, Payload: encodeEvent(event), Status: OutboxPending,
	}
	created, err := uc.repo.CreateOrderTx(ctx, o, snapshot, lines, outbox)
	if err != nil {
		return nil, err
	}
	// Best-effort cart cleanup; a stale cart row is harmless because every
	// order line is priced from the SKU catalogue, not from the cart.
	ids := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if in.CartID != "" {
			ids = append(ids, in.CartID)
		}
	}
	if len(ids) > 0 {
		_ = uc.cart.Remove(ctx, userID, ids)
	}
	return created, nil
}

func findLine(lines []*OrderGoods, skuID string) *OrderGoods {
	for _, l := range lines {
		if l.SkuID == skuID {
			return l
		}
	}
	return nil
}

func (uc *OrderUsecase) List(ctx context.Context, userID string, status, page, pageSize int) ([]*Order, int32, error) {
	page, pageSize = pageBounds(page, pageSize)
	return uc.repo.List(ctx, userID, status, page, pageSize)
}

func (uc *OrderUsecase) Detail(ctx context.Context, userID, orderSn string) (*Order, error) {
	o, err := uc.repo.FindBySn(ctx, orderSn)
	if err != nil {
		return nil, err
	}
	if o.UserID != userID {
		return nil, ErrOrderNotFound
	}
	return o, nil
}

func (uc *OrderUsecase) AdminList(ctx context.Context, status, page, pageSize int) ([]*Order, int32, error) {
	page, pageSize = pageBounds(page, pageSize)
	return uc.repo.List(ctx, "", status, page, pageSize)
}

func (uc *OrderUsecase) AdminDetail(ctx context.Context, orderSn string) (*Order, error) {
	return uc.repo.FindBySn(ctx, orderSn)
}

func (uc *OrderUsecase) Dashboard(ctx context.Context) (*DashboardStats, error) {
	return uc.repo.Dashboard(ctx)
}

func (uc *OrderUsecase) Cancel(ctx context.Context, userID, orderSn string) error {
	o, err := uc.repo.FindBySn(ctx, orderSn)
	if err != nil {
		return err
	}
	if o.UserID != userID {
		return ErrOrderNotFound
	}
	if o.Status != OrderStatusPending && o.Status != OrderStatusLocked {
		return ErrOrderStatusConflict
	}
	ok, err := uc.repo.UpdateStatusIf(ctx, orderSn, o.Status, OrderStatusCancelled)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOrderStatusConflict
	}
	_ = uc.inventory.Release(ctx, orderSn)
	return nil
}

func (uc *OrderUsecase) Ship(ctx context.Context, orderSn, post string) error {
	ok, err := uc.repo.UpdateStatusIf(ctx, orderSn, OrderStatusPaid, OrderStatusShipped)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOrderStatusConflict
	}
	return uc.repo.SetPost(ctx, orderSn, post)
}

func (uc *OrderUsecase) Refund(ctx context.Context, orderSn string) error {
	o, err := uc.repo.FindBySn(ctx, orderSn)
	if err != nil {
		return err
	}
	if o.Status != OrderStatusPaid && o.Status != OrderStatusShipped && o.Status != OrderStatusCompleted {
		return ErrOrderStatusConflict
	}
	ok, err := uc.repo.UpdateStatusIf(ctx, orderSn, o.Status, OrderStatusRefunding)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOrderStatusConflict
	}
	return nil
}

// CancelTimedOut cancels orders that stayed Pending past the pay timeout and
// releases their inventory locks. Invoked by the MQ server's timeout loop.
func (uc *OrderUsecase) CancelTimedOut(ctx context.Context, before int64, limit int) (int, error) {
	orders, err := uc.repo.ListTimedOut(ctx, before, limit)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, o := range orders {
		ok, err := uc.repo.UpdateStatusIf(ctx, o.OrderSn, OrderStatusPending, OrderStatusCancelled)
		if err != nil {
			return n, err
		}
		if ok {
			_ = uc.inventory.Release(ctx, o.OrderSn)
			n++
		}
	}
	return n, nil
}

// --- event handlers (called by internal/service adapters) -----------------

func (uc *OrderUsecase) HandleInventoryLocked(ctx context.Context, event *DomainEvent) error {
	_, err := uc.repo.UpdateStatusIf(ctx, event.OrderSn, OrderStatusPending, OrderStatusLocked)
	return err
}

func (uc *OrderUsecase) HandleInventoryFailed(ctx context.Context, event *DomainEvent) error {
	_, err := uc.repo.UpdateStatusIf(ctx, event.OrderSn, OrderStatusPending, OrderStatusCancelled)
	return err
}

func (uc *OrderUsecase) HandlePaymentSuccess(ctx context.Context, event *DomainEvent) error {
	order, err := uc.repo.FindBySn(ctx, event.OrderSn)
	if err != nil {
		return err
	}
	if order.Status >= OrderStatusPaid {
		return nil // idempotent: already advanced
	}
	ok, err := uc.repo.UpdateStatusIf(ctx, event.OrderSn, OrderStatusLocked, OrderStatusPaid)
	if err != nil {
		return err
	}
	if !ok {
		return ErrOrderStatusConflict
	}
	paid := newDomainEvent(EventOrderPaid, event.OrderSn, event.UserID)
	paid.PaymentNo = event.PaymentNo
	paid.Amount = event.Amount
	paid.Items = event.Items
	_, err = uc.outbox.Create(ctx, &OutboxEvent{
		ID: newID(), EventID: paid.ID, EventType: EventOrderPaid,
		OrderSn: paid.OrderSn, Payload: encodeEvent(paid), Status: OutboxPending,
	})
	return err
}

// InventoryUsecase owns stock locking and implements the inventory side of
// the Saga. Locks are taken with FOR UPDATE + a version CAS in one tx.
type InventoryUsecase struct {
	repo      InventoryRepo
	stock     StockCache
	publisher EventPublisher
}

func NewInventoryUsecase(repo InventoryRepo, stock StockCache, publisher EventPublisher) *InventoryUsecase {
	return &InventoryUsecase{repo: repo, stock: stock, publisher: publisher}
}

func (uc *InventoryUsecase) EnsureStock(ctx context.Context, skuID string, total int64) error {
	if err := uc.repo.EnsureSku(ctx, skuID, total); err != nil {
		return err
	}
	return uc.stock.Set(ctx, skuID, total)
}

func (uc *InventoryUsecase) HandleOrderCreated(ctx context.Context, event *DomainEvent) error {
	items := make([]StockItem, 0, len(event.Items))
	for _, it := range event.Items {
		items = append(items, StockItem{SkuID: it.SkuID, Num: it.Num})
	}
	if err := uc.repo.TryLock(ctx, event.OrderSn, items); err != nil {
		failed := newDomainEvent(EventInventoryLockFailed, event.OrderSn, event.UserID)
		failed.Reason = err.Error()
		failed.Items = event.Items
		return uc.publisher.Publish(ctx, ExchangeInventory, EventInventoryLockFailed, failed)
	}
	ok := newDomainEvent(EventInventoryLockOK, event.OrderSn, event.UserID)
	ok.Items = event.Items
	for _, it := range items {
		if avail, err := uc.repo.Available(ctx, it.SkuID); err == nil {
			_ = uc.stock.Set(ctx, it.SkuID, avail)
		}
	}
	return uc.publisher.Publish(ctx, ExchangeInventory, EventInventoryLockOK, ok)
}

func (uc *InventoryUsecase) HandleOrderPaid(ctx context.Context, event *DomainEvent) error {
	items := make([]StockItem, 0, len(event.Items))
	for _, it := range event.Items {
		items = append(items, StockItem{SkuID: it.SkuID, Num: it.Num})
	}
	if err := uc.repo.ConfirmDeduct(ctx, event.OrderSn, items); err != nil {
		if errors.Is(err, ErrInventoryNotEnough) {
			return err
		}
		return err
	}
	for _, it := range event.Items {
		_ = uc.stock.Invalidate(ctx, it.SkuID)
	}
	return nil
}
