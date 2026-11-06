package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent"
	"github.com/go-kratos/kratos-layout/internal/data/ent/order"
	"github.com/go-kratos/kratos-layout/internal/data/ent/orderaddress"
	"github.com/go-kratos/kratos-layout/internal/data/ent/ordergoods"
	"github.com/go-kratos/kratos-layout/internal/data/ent/predicate"
)

func orderToBiz(po *ent.Order) *biz.Order {
	if po == nil {
		return nil
	}
	return &biz.Order{
		ID: idStr(po.ID), UserID: po.UserID, OrderSn: po.OrderSn,
		Status: po.Status, Post: po.Post, Total: po.Total,
		Address: po.Address, Name: po.Name, Mobile: po.Mobile,
		AddTime: po.AddTime, PayTime: po.PayTime,
	}
}

func orderGoodsToBiz(po *ent.OrderGoods) *biz.OrderGoods {
	return &biz.OrderGoods{
		ID: idStr(po.ID), OrderID: po.OrderID, SkuID: po.SkuID,
		SkuName: po.SkuName, SkuPrice: po.SkuPrice, Num: po.Num,
		TotalPrice: po.TotalPrice,
	}
}

type orderRepo struct{ data *Data }

func NewOrderRepo(d *Data) biz.OrderRepo { return &orderRepo{data: d} }

func (r *orderRepo) CreateOrderTx(ctx context.Context, o *biz.Order, a *biz.OrderAddress, items []*biz.OrderGoods, outbox *biz.OutboxEvent) (*biz.Order, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	po, err := tx.Order.Create().
		SetID(parseID(o.ID)).SetUserID(o.UserID).SetOrderSn(o.OrderSn).
		SetStatus(o.Status).SetPost(o.Post).SetTotal(o.Total).
		SetAddress(o.Address).SetName(o.Name).SetMobile(o.Mobile).
		SetAddTime(o.AddTime).SetPayTime(o.PayTime).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	a.OrderID = idStr(po.ID)
	if _, err = tx.OrderAddress.Create().
		SetID(parseID(a.ID)).SetOrderID(a.OrderID).
		SetName(a.Name).SetMobile(a.Mobile).SetProvince(a.Province).
		SetCity(a.City).SetDistricts(a.Districts).SetAddress(a.Address).
		SetPostCode(a.PostCode).Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	for _, it := range items {
		it.OrderID = idStr(po.ID)
		if _, err = tx.OrderGoods.Create().
			SetID(parseID(it.ID)).SetOrderID(it.OrderID).SetSkuID(it.SkuID).
			SetSkuName(it.SkuName).SetSkuPrice(it.SkuPrice).SetNum(it.Num).
			SetTotalPrice(it.TotalPrice).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if outbox != nil {
		if _, err = tx.OrderEventOutbox.Create().
			SetID(parseID(outbox.ID)).SetEventID(outbox.EventID).
			SetEventType(outbox.EventType).SetOrderSn(outbox.OrderSn).
			SetPayload(outbox.Payload).SetStatus(outbox.Status).
			SetRetryCount(outbox.RetryCount).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.FindBySn(ctx, o.OrderSn)
}

func (r *orderRepo) hydrate(ctx context.Context, po *ent.Order) (*biz.Order, error) {
	o := orderToBiz(po)
	goods, err := r.OrderGoods(ctx, idStr(po.ID))
	if err == nil {
		o.Goods = goods
	}
	addr, err := r.OrderAddress(ctx, idStr(po.ID))
	if err == nil {
		o.Address2 = addr
	}
	return o, nil
}

func (r *orderRepo) FindBySn(ctx context.Context, orderSn string) (*biz.Order, error) {
	po, err := r.data.db.Order.Query().Where(order.OrderSnEQ(orderSn)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	return r.hydrate(ctx, po)
}

func (r *orderRepo) FindByID(ctx context.Context, id string) (*biz.Order, error) {
	po, err := r.data.db.Order.Query().Where(order.IDEQ(parseID(id))).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrOrderNotFound
		}
		return nil, err
	}
	return r.hydrate(ctx, po)
}

func (r *orderRepo) List(ctx context.Context, userID string, status, page, pageSize int) ([]*biz.Order, int32, error) {
	var ps []predicate.Order
	if userID != "" {
		ps = append(ps, order.UserIDEQ(userID))
	}
	if status > 0 {
		ps = append(ps, order.StatusEQ(int32(status)))
	}
	total, err := r.data.db.Order.Query().Where(ps...).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	pos, err := r.data.db.Order.Query().Where(ps...).
		Order(order.ByAddTime(), order.ByID()).
		Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*biz.Order, 0, len(pos))
	for _, po := range pos {
		o, err := r.hydrate(ctx, po)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, o)
	}
	return out, int32(total), nil
}

func (r *orderRepo) UpdateStatusIf(ctx context.Context, orderSn string, from, to int32) (bool, error) {
	n, err := r.data.db.Order.Update().
		Where(order.OrderSnEQ(orderSn), order.StatusEQ(from)).
		SetStatus(to).Save(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *orderRepo) SetPost(ctx context.Context, orderSn, post string) error {
	_, err := r.data.db.Order.Update().Where(order.OrderSnEQ(orderSn)).SetPost(post).Save(ctx)
	return err
}

func (r *orderRepo) ListTimedOut(ctx context.Context, before int64, limit int) ([]*biz.Order, error) {
	pos, err := r.data.db.Order.Query().
		Where(order.StatusEQ(biz.OrderStatusPending), order.AddTimeLTE(before)).
		Order(order.ByAddTime()).Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.Order, 0, len(pos))
	for _, po := range pos {
		out = append(out, orderToBiz(po))
	}
	return out, nil
}

func (r *orderRepo) OrderGoods(ctx context.Context, orderID string) ([]*biz.OrderGoods, error) {
	pos, err := r.data.db.OrderGoods.Query().Where(ordergoods.OrderIDEQ(orderID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.OrderGoods, 0, len(pos))
	for _, po := range pos {
		out = append(out, orderGoodsToBiz(po))
	}
	return out, nil
}

func (r *orderRepo) OrderAddress(ctx context.Context, orderID string) (*biz.OrderAddress, error) {
	po, err := r.data.db.OrderAddress.Query().Where(orderaddress.OrderIDEQ(orderID)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.OrderAddress{
		ID: idStr(po.ID), OrderID: po.OrderID, Name: po.Name,
		Mobile: po.Mobile, Province: po.Province, City: po.City,
		Districts: po.Districts, Address: po.Address, PostCode: po.PostCode,
	}, nil
}

// Dashboard is computed in Go from the order rows. For a teaching/demo mall
// the dataset is small; a production system would push these aggregations
// into SQL. It avoids leaking Ent builder types out of the data layer.
func (r *orderRepo) Dashboard(ctx context.Context) (*biz.DashboardStats, error) {
	pos, err := r.data.db.Order.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	stats := &biz.DashboardStats{}
	users, err := r.data.db.User.Query().Count(ctx)
	if err == nil {
		stats.TotalUsers = int64(users)
	}
	today := time.Now().Format("2006-01-02")
	byStatus := map[int32]int64{}
	byDate := map[string]*biz.DailySales{}
	top := map[string]*biz.TopGoods{}
	for _, po := range pos {
		stats.TotalOrders++
		stats.TotalSales += po.Total
		byStatus[po.Status]++
		d := time.Unix(po.AddTime, 0).Format("2006-01-02")
		if d == today {
			stats.TodayOrders++
			stats.TodaySales += po.Total
		}
		if _, ok := byDate[d]; !ok {
			byDate[d] = &biz.DailySales{Date: d}
		}
		byDate[d].OrderCount++
		byDate[d].Amount += po.Total
		goods, _ := r.OrderGoods(ctx, idStr(po.ID))
		for _, g := range goods {
			if _, ok := top[g.SkuID]; !ok {
				top[g.SkuID] = &biz.TopGoods{SkuID: g.SkuID, SkuName: g.SkuName}
			}
			top[g.SkuID].Num += int64(g.Num)
			top[g.SkuID].Amount += g.TotalPrice
		}
	}
	for s, c := range byStatus {
		stats.StatusCounts = append(stats.StatusCounts, biz.StatusCount{Status: s, Count: c})
	}
	for i := 29; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		if v, ok := byDate[d]; ok {
			stats.Last30Days = append(stats.Last30Days, *v)
		} else {
			stats.Last30Days = append(stats.Last30Days, biz.DailySales{Date: d})
		}
	}
	for _, v := range top {
		stats.TopGoods = append(stats.TopGoods, *v)
	}
	return stats, nil
}
