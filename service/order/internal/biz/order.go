package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"log/slog"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	cartV1 "order/api/cart/v1"
	goodsV1 "order/api/goods/v1"
	userV1 "order/api/user/v1"
	"order/internal/domain"
	"order/internal/pkg/mq"
)

type OrderRepo interface {
	Create(ctx context.Context, order *domain.Order, address *domain.OrderAddress, items []*domain.OrderGoods, outbox *domain.OutboxEvent) error
	ListPendingOutbox(ctx context.Context, limit int) ([]*domain.OutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, id int64) error
	CreateOutbox(ctx context.Context, outbox *domain.OutboxEvent) error
	UpdateStatusIf(ctx context.Context, orderSn string, from, to int) (bool, error)
	ListItemsByOrderSn(ctx context.Context, orderSn string) ([]*domain.OrderGoods, error)
	ListPendingTimeout(ctx context.Context, minutes int) ([]*domain.Order, error)
	GetDetail(ctx context.Context, userId int64, orderSn string) (*domain.Order, error)
	ListByUser(ctx context.Context, userId int64, page, pageSize int) ([]*domain.Order, int64, error)
	AdminList(ctx context.Context, page, pageSize, status int) ([]*domain.Order, int64, error)
	GetByOrderSn(ctx context.Context, orderSn string) (*domain.Order, error)
	Ship(ctx context.Context, orderSn, post string) (bool, error)
	Refund(ctx context.Context, orderSn string) (bool, error)
	DashboardStats(ctx context.Context) (*domain.DashboardStats, error)
	IncrementOutboxRetry(ctx context.Context, id int64) error
	MarkOutboxFailed(ctx context.Context, id int64) error
}

type OrderUsecase struct {
	repo      OrderRepo
	userRPC   userV1.UserClient
	cartRPC   cartV1.CartClient
	goodsRPC  goodsV1.GoodsClient
	publisher *mq.Publisher
	log       *slog.Logger
}

func NewOrderUsecase(repo OrderRepo, userRPC userV1.UserClient, cartRPC cartV1.CartClient, goodsRPC goodsV1.GoodsClient, publisher *mq.Publisher,
	logger *slog.Logger) *OrderUsecase {

	uc := &OrderUsecase{
		repo:      repo,
		userRPC:   userRPC,
		cartRPC:   cartRPC,
		goodsRPC:  goodsRPC,
		publisher: publisher,
		log:       logger,
	}
	if publisher != nil {
		go uc.relayLoop()
	}
	go uc.timeoutLoop()
	return uc
}

func (oc *OrderUsecase) CreateOrder(ctx context.Context, order *domain.CreateOrder) (*domain.Order, error) {
	if order == nil || len(order.CartItem) == 0 {
		return nil, kerrors.New(400, "ORDER_ITEM_EMPTY", "订单商品不能为空")
	}

	// 跨服务查询购物车，校验本次下单条目
	cartList, err := oc.cartRPC.ListCart(ctx, &cartV1.ListCartRequest{UserId: order.UserId})
	if err != nil {
		return nil, err
	}
	cartMap := make(map[int64]*cartV1.CartInfoReply, len(cartList.Results))
	for _, cart := range cartList.Results {
		cartMap[cart.Id] = cart
	}
	for _, item := range order.CartItem {
		cart := cartMap[item.CartId]
		if cart == nil || cart.SkuId != item.SkuId {
			return nil, kerrors.New(400, "CART_ITEM_INVALID", "购物车条目与订单不一致")
		}
		if item.SkuNum <= 0 {
			return nil, kerrors.New(400, "SKU_NUM_INVALID", "商品数量必须大于 0")
		}
	}

	// 跨服务查询 SKU，使用服务端价格计算金额
	skuResp, err := oc.goodsRPC.SkuList(ctx, &goodsV1.SkuListRequest{Id: order.CartItem.GetSkuId()})
	if err != nil {
		return nil, err
	}
	skuMap := make(map[int64]*goodsV1.SkuInfo, len(skuResp.List))
	for _, sku := range skuResp.List {
		skuMap[sku.Id] = sku
	}

	var (
		goodsAmount int64
		orderGoods  []*domain.OrderGoods
	)
	for _, item := range order.CartItem {
		sku := skuMap[item.SkuId]
		if sku == nil || !sku.OnSale {
			return nil, kerrors.New(400, "SKU_NOT_FOUND", "SKU 不存在或已下架")
		}
		price := sku.Price
		if sku.PromotionPrice > 0 {
			price = sku.PromotionPrice
		}
		cart := cartMap[item.CartId]
		if cart != nil && cart.GoodsPrice > 0 && cart.GoodsPrice != price {
			return nil, kerrors.New(400, "PRICE_CHANGED", "商品价格已发生变化，请刷新购物车后再下单")
		}
		if item.SkuPrice > 0 && item.SkuPrice != price {
			return nil, kerrors.New(400, "PRICE_CHANGED", "商品价格已发生变化，请刷新购物车后再下单")
		}
		amount := price * int64(item.SkuNum)
		goodsAmount += amount
		orderGoods = append(orderGoods, &domain.OrderGoods{
			UserId:     order.UserId,
			SkuId:      sku.Id,
			SkuName:    sku.SkuName,
			SkuPrice:   price,
			Num:        item.SkuNum,
			TotalPrice: amount,
		})
	}

	// 跨服务查询收货地址
	address, err := oc.userRPC.GetAddress(ctx, &userV1.AddressReq{
		Id:  order.AddressId,
		Uid: order.UserId,
	})
	if err != nil {
		return nil, err
	}

	orderSn := generateOrderSn()
	od := &domain.Order{
		User:          order.UserId,
		OrderSn:       orderSn,
		GoodsAmount:   goodsAmount,
		OrderAmount:   goodsAmount,
		ExpressAmount: 0,
		OrderStatus:   domain.OrderStatusInventoryPending,
		Address:       address.Address,
		SignerName:    address.Name,
		SingerMobile:  address.Mobile,
	}
	orderAddress := &domain.OrderAddress{
		User:            order.UserId,
		OrderSn:         orderSn,
		RecipientName:   address.Name,
		RecipientMobile: address.Mobile,
		Province:        address.Province,
		City:            address.City,
		Districts:       address.Districts,
		Address:         address.Address,
		PostCode:        address.PostCode,
	}
	for _, item := range orderGoods {
		item.OrderSn = orderSn
	}

	eventID := generateEventID()
	outbox := &domain.OutboxEvent{
		EventID:     eventID,
		EventType:   "order.created",
		OrderSn:     orderSn,
		Payload:     buildOrderEventPayload(eventID, "order.created", orderSn, order.UserId, orderGoods),
		Traceparent: traceparentFromCtx(ctx),
	}
	if err := oc.repo.Create(ctx, od, orderAddress, orderGoods, outbox); err != nil {
		return nil, err
	}
	// 下单成功后删除已购买的购物车条目（尽力而为，不阻塞订单）
	for _, item := range order.CartItem {
		if _, err := oc.cartRPC.DeleteCart(ctx, &cartV1.DeleteCartRequest{Id: item.CartId, UserId: order.UserId}); err != nil {
			oc.log.Error("delete cart after order failed", slog.Int64("cart_id", item.CartId), slog.Any("err", err))
		}
	}
	fmt.Printf("order created: %s amount=%d\n", od.OrderSn, od.OrderAmount)
	return od, nil
}

func generateOrderSn() string {
	return fmt.Sprintf("%s%06d", time.Now().Format("20060102150405"), rand.Intn(1000000))
}

func generateEventID() string {
	return fmt.Sprintf("evt-%d-%d", time.Now().UnixNano(), rand.Intn(1000000))
}

// traceparentFromCtx captures the W3C traceparent of the current span so the
// async outbox relay can resume the trace when it publishes later.
func traceparentFromCtx(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

func buildOrderEventPayload(eventID, eventType, orderSn string, userId int64, items []*domain.OrderGoods) []byte {
	type skuItem struct {
		SkuID int64 `json:"sku_id"`
		Num   int32 `json:"num"`
	}
	type payload struct {
		Skus []skuItem `json:"skus"`
	}
	type event struct {
		EventID   string  `json:"event_id"`
		EventType string  `json:"event_type"`
		OrderSn   string  `json:"order_sn"`
		UserID    int64   `json:"user_id"`
		Payload   payload `json:"payload"`
	}
	e := event{
		EventID:   eventID,
		EventType: eventType,
		OrderSn:   orderSn,
		UserID:    userId,
		Payload:   payload{},
	}
	for _, item := range items {
		e.Payload.Skus = append(e.Payload.Skus, skuItem{SkuID: item.SkuId, Num: item.Num})
	}
	b, _ := json.Marshal(e)
	return b
}

func (oc *OrderUsecase) CancelOrder(ctx context.Context, userId int64, orderSn string) error {
	items, err := oc.repo.ListItemsByOrderSn(ctx, orderSn)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return kerrors.New(400, "ORDER_NOT_FOUND", "订单不存在")
	}
	changed, err := oc.repo.UpdateStatusIf(ctx, orderSn, domain.OrderStatusInventoryPending, domain.OrderStatusCancelled)
	if err != nil && kerrors.FromError(err).Reason == "ORDER_STATUS_INVALID" {
		changed, err = oc.repo.UpdateStatusIf(ctx, orderSn, domain.OrderStatusPendingPayment, domain.OrderStatusCancelled)
	}
	if err != nil {
		return err
	}
	if !changed {
		return nil // 已经是取消状态，幂等
	}
	eventID := generateEventID()
	outbox := &domain.OutboxEvent{
		EventID:     eventID,
		EventType:   "order.cancelled",
		OrderSn:     orderSn,
		Payload:     buildOrderEventPayload(eventID, "order.cancelled", orderSn, userId, items),
		Traceparent: traceparentFromCtx(ctx),
	}
	return oc.repo.CreateOutbox(ctx, outbox)
}

func (oc *OrderUsecase) MarkInventoryLocked(ctx context.Context, orderSn string) error {
	_, err := oc.repo.UpdateStatusIf(ctx, orderSn, domain.OrderStatusInventoryPending, domain.OrderStatusPendingPayment)
	return err
}

func (oc *OrderUsecase) MarkInventoryFailed(ctx context.Context, orderSn string) error {
	_, err := oc.repo.UpdateStatusIf(ctx, orderSn, domain.OrderStatusInventoryPending, domain.OrderStatusCancelled)
	return err
}

func (oc *OrderUsecase) MarkPaid(ctx context.Context, orderSn string) error {
	items, err := oc.repo.ListItemsByOrderSn(ctx, orderSn)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return kerrors.New(400, "ORDER_NOT_FOUND", "订单不存在")
	}
	changed, err := oc.repo.UpdateStatusIf(ctx, orderSn, domain.OrderStatusPendingPayment, domain.OrderStatusPaid)
	if err != nil {
		return err
	}
	if !changed {
		return nil // 已经是已支付状态，幂等
	}
	eventID := generateEventID()
	outbox := &domain.OutboxEvent{
		EventID:     eventID,
		EventType:   "order.paid",
		OrderSn:     orderSn,
		Payload:     buildOrderEventPayload(eventID, "order.paid", orderSn, items[0].UserId, items),
		Traceparent: traceparentFromCtx(ctx),
	}
	return oc.repo.CreateOutbox(ctx, outbox)
}

func (oc *OrderUsecase) GetOrder(ctx context.Context, userId int64, orderSn string) (*domain.Order, error) {
	return oc.repo.GetDetail(ctx, userId, orderSn)
}

func (oc *OrderUsecase) ListOrders(ctx context.Context, userId int64, page, pageSize int) ([]*domain.Order, int64, error) {
	return oc.repo.ListByUser(ctx, userId, page, pageSize)
}

func (oc *OrderUsecase) AdminListOrders(ctx context.Context, page, pageSize, status int) ([]*domain.Order, int64, error) {
	return oc.repo.AdminList(ctx, page, pageSize, status)
}

func (oc *OrderUsecase) AdminGetOrder(ctx context.Context, orderSn string) (*domain.Order, error) {
	return oc.repo.GetByOrderSn(ctx, orderSn)
}

func (oc *OrderUsecase) ShipOrder(ctx context.Context, orderSn, post string) error {
	changed, err := oc.repo.Ship(ctx, orderSn, post)
	if err != nil {
		return err
	}
	if !changed {
		return kerrors.New(400, "ORDER_STATUS_INVALID", "只有已支付订单可以发货")
	}
	return nil
}

func (oc *OrderUsecase) RefundOrder(ctx context.Context, orderSn string) error {
	changed, err := oc.repo.Refund(ctx, orderSn)
	if err != nil {
		return err
	}
	if !changed {
		return kerrors.New(400, "ORDER_STATUS_INVALID", "当前订单状态不支持退款")
	}
	return nil
}

func (oc *OrderUsecase) DashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	return oc.repo.DashboardStats(ctx)
}

func (oc *OrderUsecase) relayLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := oc.RelayOutbox(context.Background()); err != nil {
			oc.log.Error("outbox relay error", slog.Any("err", err))
		}
	}
}

func (oc *OrderUsecase) RelayOutbox(ctx context.Context) error {
	events, err := oc.repo.ListPendingOutbox(ctx, 50)
	if err != nil {
		return err
	}
	for _, evt := range events {
		pubCtx := ctx
		if evt.Traceparent != "" {
			pubCtx = otel.GetTextMapPropagator().Extract(pubCtx, propagation.MapCarrier{"traceparent": evt.Traceparent})
		}
		if err := oc.publisher.Publish(pubCtx, "order.exchange", evt.EventType, evt.Payload); err != nil {
			oc.log.Error("publish outbox event failed", slog.Int64("id", evt.ID), slog.String("event_type", evt.EventType), slog.Any("err", err))
			if err := oc.repo.IncrementOutboxRetry(ctx, evt.ID); err != nil {
				oc.log.Error("increment outbox retry failed", slog.Int64("id", evt.ID), slog.Any("err", err))
			}
			if evt.RetryCount+1 >= 5 {
				if err := oc.repo.MarkOutboxFailed(ctx, evt.ID); err != nil {
					oc.log.Error("mark outbox failed failed", slog.Int64("id", evt.ID), slog.Any("err", err))
				}
			}
			continue
		}
		if err := oc.repo.MarkOutboxPublished(ctx, evt.ID); err != nil {
			oc.log.Error("mark outbox published failed", slog.Int64("id", evt.ID), slog.Any("err", err))
		}
	}
	return nil
}

func (oc *OrderUsecase) timeoutLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := oc.CancelExpiredOrders(context.Background()); err != nil {
			oc.log.Error("cancel expired orders error", slog.Any("err", err))
		}
	}
}

func (oc *OrderUsecase) CancelExpiredOrders(ctx context.Context) error {
	orders, err := oc.repo.ListPendingTimeout(ctx, 15)
	if err != nil {
		return err
	}
	for _, od := range orders {
		if err := oc.CancelOrder(ctx, od.User, od.OrderSn); err != nil {
			oc.log.Error("cancel expired order failed", slog.String("order_sn", od.OrderSn), slog.Any("err", err))
		}
	}
	return nil
}
