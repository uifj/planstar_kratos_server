package data

import (
	"context"
	"database/sql"
	serrors "errors"
	"log/slog"
	"strings"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"

	"order/internal/biz"
	"order/internal/domain"
)

// orderColumns lists the columns read for an order row. `user` is a MySQL
// reserved word, so it is always back-quoted.
const orderColumns = "id, `user`, order_sn, order_amount, goods_amount, order_status, express_amount, delivery_at, refund_time, post, created_at, updated_at"

type orderRepo struct {
	data *Data
	log  *slog.Logger
}

// NewOrderRepo .
func NewOrderRepo(data *Data, logger *slog.Logger) biz.OrderRepo {
	return &orderRepo{
		data: data,
		log:  logger,
	}
}

// scanOrder reads one order row into a domain.Order. delivery_at and
// refund_time are nullable, so they are scanned via sql.NullTime.
func scanOrder(row interface{ Scan(dest ...any) error }) (*domain.Order, error) {
	var (
		o          domain.Order
		deliveryAt sql.NullTime
		refundTime sql.NullTime
	)
	if err := row.Scan(&o.ID, &o.User, &o.OrderSn, &o.OrderAmount, &o.GoodsAmount,
		&o.OrderStatus, &o.ExpressAmount, &deliveryAt, &refundTime, &o.Post,
		&o.CreatedAt, &o.UpdatedAt); err != nil {
		return nil, err
	}
	if deliveryAt.Valid {
		o.DeliveryAt = deliveryAt.Time
	}
	if refundTime.Valid {
		o.RefundTime = refundTime.Time
	}
	return &o, nil
}

// loadAddress enriches dom with the recipient info stored in order_address.
func (o *orderRepo) loadAddress(ctx context.Context, dom *domain.Order) {
	var addr struct {
		address, name, mobile string
	}
	err := o.data.DB(ctx).QueryRowContext(ctx,
		"SELECT address, recipient_name, recipient_mobile FROM order_address WHERE order_sn = ? LIMIT 1",
		dom.OrderSn,
	).Scan(&addr.address, &addr.name, &addr.mobile)
	if err == nil {
		dom.Address = addr.address
		dom.SignerName = addr.name
		dom.SingerMobile = addr.mobile
	}
}

func (o *orderRepo) Create(ctx context.Context, order *domain.Order, address *domain.OrderAddress, items []*domain.OrderGoods, outbox *domain.OutboxEvent) error {
	return o.data.ExecTx(ctx, func(ctx context.Context) error {
		res, err := o.data.DB(ctx).ExecContext(ctx,
			"INSERT INTO orders (`user`, order_sn, order_amount, goods_amount, order_status, express_amount, post, created_at, updated_at) "+
				"VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
			order.User, order.OrderSn, order.OrderAmount, order.GoodsAmount,
			order.OrderStatus, order.ExpressAmount, order.Post,
		)
		if err != nil {
			return err
		}
		id, _ := res.LastInsertId()
		order.ID = id
		now := time.Now()
		order.CreatedAt = now
		order.UpdatedAt = now

		if address != nil {
			if _, err := o.data.DB(ctx).ExecContext(ctx,
				"INSERT INTO order_address (`user`, order_sn, recipient_name, recipient_mobile, province, city, districts, address, post_code, created_at, updated_at) "+
					"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
				address.User, address.OrderSn, address.RecipientName, address.RecipientMobile,
				address.Province, address.City, address.Districts, address.Address, address.PostCode,
			); err != nil {
				return err
			}
		}

		for _, item := range items {
			if _, err := o.data.DB(ctx).ExecContext(ctx,
				"INSERT INTO order_goods (order_sn, user_id, sku_id, sku_name, sku_price, num, total_price, created_at, updated_at) "+
					"VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())",
				item.OrderSn, item.UserId, item.SkuId, item.SkuName, item.SkuPrice, item.Num, item.TotalPrice,
			); err != nil {
				return err
			}
		}

		if outbox != nil {
			ores, err := o.data.DB(ctx).ExecContext(ctx,
				"INSERT INTO order_event_outbox (event_id, event_type, order_sn, payload, status, retry_count, created_at) "+
					"VALUES (?, ?, ?, ?, 0, 0, NOW())",
				outbox.EventID, outbox.EventType, outbox.OrderSn, string(outbox.Payload),
			)
			if err != nil {
				return err
			}
			outbox.ID, _ = ores.LastInsertId()
		}
		return nil
	})
}

func (o *orderRepo) ListPendingOutbox(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	rows, err := o.data.DB(ctx).QueryContext(ctx,
		"SELECT id, event_id, event_type, order_sn, payload, status, retry_count FROM order_event_outbox WHERE status = 0 ORDER BY id ASC LIMIT ?",
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.OutboxEvent
	for rows.Next() {
		var ev domain.OutboxEvent
		var payload string
		if err := rows.Scan(&ev.ID, &ev.EventID, &ev.EventType, &ev.OrderSn, &payload, &ev.Status, &ev.RetryCount); err != nil {
			return nil, err
		}
		ev.Payload = []byte(payload)
		res = append(res, &ev)
	}
	return res, rows.Err()
}

func (o *orderRepo) MarkOutboxPublished(ctx context.Context, id int64) error {
	_, err := o.data.DB(ctx).ExecContext(ctx,
		"UPDATE order_event_outbox SET status = 1, published_at = NOW() WHERE id = ?", id)
	return err
}

func (o *orderRepo) CreateOutbox(ctx context.Context, outbox *domain.OutboxEvent) error {
	res, err := o.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO order_event_outbox (event_id, event_type, order_sn, payload, status, retry_count, created_at) "+
			"VALUES (?, ?, ?, ?, 0, 0, NOW())",
		outbox.EventID, outbox.EventType, outbox.OrderSn, string(outbox.Payload),
	)
	if err != nil {
		return err
	}
	outbox.ID, _ = res.LastInsertId()
	return nil
}

func (o *orderRepo) UpdateStatusIf(ctx context.Context, orderSn string, from, to int) (bool, error) {
	res, err := o.data.DB(ctx).ExecContext(ctx,
		"UPDATE orders SET order_status = ?, updated_at = NOW() WHERE order_sn = ? AND order_status = ?",
		to, orderSn, from)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		var current int
		if err := o.data.DB(ctx).QueryRowContext(ctx,
			"SELECT order_status FROM orders WHERE order_sn = ?", orderSn,
		).Scan(&current); err != nil {
			return false, err
		}
		if current == to {
			return false, nil // 幂等：已经是目标状态
		}
		return false, kerrors.New(400, "ORDER_STATUS_INVALID", "订单状态不允许此操作")
	}
	return true, nil
}

func (o *orderRepo) ListItemsByOrderSn(ctx context.Context, orderSn string) ([]*domain.OrderGoods, error) {
	rows, err := o.data.DB(ctx).QueryContext(ctx,
		"SELECT id, order_sn, user_id, sku_id, sku_name, sku_price, num, total_price FROM order_goods WHERE order_sn = ?",
		orderSn)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.OrderGoods
	for rows.Next() {
		var item domain.OrderGoods
		if err := rows.Scan(&item.ID, &item.OrderSn, &item.UserId, &item.SkuId, &item.SkuName, &item.SkuPrice, &item.Num, &item.TotalPrice); err != nil {
			return nil, err
		}
		res = append(res, &item)
	}
	return res, rows.Err()
}

func (o *orderRepo) ListPendingTimeout(ctx context.Context, minutes int) ([]*domain.Order, error) {
	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	rows, err := o.data.DB(ctx).QueryContext(ctx,
		"SELECT "+orderColumns+" FROM orders WHERE order_status = ? AND created_at < ?",
		domain.OrderStatusPendingPayment, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []*domain.Order
	for rows.Next() {
		dom, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		res = append(res, dom)
	}
	return res, rows.Err()
}

func (o *orderRepo) GetDetail(ctx context.Context, userId int64, orderSn string) (*domain.Order, error) {
	row := o.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+orderColumns+" FROM orders WHERE order_sn = ? AND `user` = ?", orderSn, userId)
	dom, err := scanOrder(row)
	if serrors.Is(err, sql.ErrNoRows) {
		return nil, kerrors.New(400, "ORDER_NOT_FOUND", "订单不存在")
	}
	if err != nil {
		return nil, err
	}
	o.loadAddress(ctx, dom)
	items, err := o.ListItemsByOrderSn(ctx, orderSn)
	if err != nil {
		return nil, err
	}
	dom.Items = items
	return dom, nil
}

func (o *orderRepo) ListByUser(ctx context.Context, userId int64, page, pageSize int) ([]*domain.Order, int64, error) {
	var total int64
	if err := o.data.DB(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM orders WHERE `user` = ?", userId,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	rows, err := o.data.DB(ctx).QueryContext(ctx,
		"SELECT "+orderColumns+" FROM orders WHERE `user` = ? ORDER BY id DESC LIMIT ? OFFSET ?",
		userId, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	res := make([]*domain.Order, 0, pageSize)
	for rows.Next() {
		dom, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		res = append(res, dom)
	}
	return res, total, rows.Err()
}

func (o *orderRepo) AdminList(ctx context.Context, page, pageSize, status int) ([]*domain.Order, int64, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if status > 0 {
		where = append(where, "order_status = ?")
		args = append(args, status)
	}
	cond := " WHERE " + strings.Join(where, " AND ")

	var total int64
	if err := o.data.DB(ctx).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM orders"+cond, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	rows, err := o.data.DB(ctx).QueryContext(ctx,
		"SELECT "+orderColumns+" FROM orders"+cond+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var res []*domain.Order
	for rows.Next() {
		dom, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		o.loadAddress(ctx, dom)
		if items, err := o.ListItemsByOrderSn(ctx, dom.OrderSn); err == nil {
			dom.Items = items
		}
		res = append(res, dom)
	}
	return res, total, rows.Err()
}

func (o *orderRepo) GetByOrderSn(ctx context.Context, orderSn string) (*domain.Order, error) {
	row := o.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+orderColumns+" FROM orders WHERE order_sn = ?", orderSn)
	dom, err := scanOrder(row)
	if serrors.Is(err, sql.ErrNoRows) {
		return nil, kerrors.New(400, "ORDER_NOT_FOUND", "订单不存在")
	}
	if err != nil {
		return nil, err
	}
	o.loadAddress(ctx, dom)
	items, err := o.ListItemsByOrderSn(ctx, orderSn)
	if err != nil {
		return nil, err
	}
	dom.Items = items
	return dom, nil
}

func (o *orderRepo) Ship(ctx context.Context, orderSn, post string) (bool, error) {
	res, err := o.data.DB(ctx).ExecContext(ctx,
		"UPDATE orders SET order_status = 3, post = ?, delivery_at = NOW(), updated_at = NOW() WHERE order_sn = ? AND order_status = 2",
		post, orderSn)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (o *orderRepo) Refund(ctx context.Context, orderSn string) (bool, error) {
	res, err := o.data.DB(ctx).ExecContext(ctx,
		"UPDATE orders SET order_status = 7, refund_time = NOW(), updated_at = NOW() WHERE order_sn = ? AND order_status IN (2,3,4)",
		orderSn)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected > 0, nil
}

func (o *orderRepo) DashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	stats := &domain.DashboardStats{}
	db := o.data.DB(ctx)

	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM orders").Scan(&stats.TotalOrders); err != nil {
		return nil, err
	}
	if err := db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(order_amount),0) FROM orders WHERE order_status IN (2,3,4,6)").Scan(&stats.TotalSales); err != nil {
		return nil, err
	}
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM orders WHERE DATE(created_at) = CURDATE()").Scan(&stats.TodayOrders); err != nil {
		return nil, err
	}
	if err := db.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(order_amount),0) FROM orders WHERE DATE(created_at) = CURDATE() AND order_status IN (2,3,4,6)").Scan(&stats.TodaySales); err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx,
		"SELECT order_status, COUNT(*) FROM orders GROUP BY order_status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sc domain.StatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			return nil, err
		}
		stats.StatusCounts = append(stats.StatusCounts, &sc)
	}

	daily, err := db.QueryContext(ctx, `
		SELECT DATE_FORMAT(created_at, '%Y-%m-%d') AS d,
		       COUNT(*) AS order_count,
		       COALESCE(SUM(CASE WHEN order_status IN (2,3,4,6) THEN order_amount ELSE 0 END), 0) AS amount
		FROM orders
		WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL 29 DAY)
		GROUP BY d
		ORDER BY d`)
	if err != nil {
		return nil, err
	}
	defer daily.Close()
	for daily.Next() {
		var ds domain.DailySales
		if err := daily.Scan(&ds.Date, &ds.OrderCount, &ds.Amount); err != nil {
			return nil, err
		}
		stats.Last30Days = append(stats.Last30Days, &ds)
	}

	top, err := db.QueryContext(ctx, `
		SELECT og.sku_id, og.sku_name, SUM(og.num) AS num, SUM(og.total_price) AS amount
		FROM order_goods og
		JOIN orders o ON o.order_sn = og.order_sn
		WHERE o.order_status IN (2,3,4,6)
		GROUP BY og.sku_id, og.sku_name
		ORDER BY num DESC
		LIMIT 5`)
	if err != nil {
		return nil, err
	}
	defer top.Close()
	for top.Next() {
		var tg domain.TopGoods
		if err := top.Scan(&tg.SkuID, &tg.SkuName, &tg.Num, &tg.Amount); err != nil {
			return nil, err
		}
		stats.TopGoods = append(stats.TopGoods, &tg)
	}

	return stats, nil
}

func (o *orderRepo) IncrementOutboxRetry(ctx context.Context, id int64) error {
	_, err := o.data.DB(ctx).ExecContext(ctx,
		"UPDATE order_event_outbox SET retry_count = retry_count + 1 WHERE id = ?", id)
	return err
}

func (o *orderRepo) MarkOutboxFailed(ctx context.Context, id int64) error {
	_, err := o.data.DB(ctx).ExecContext(ctx,
		"UPDATE order_event_outbox SET status = 2 WHERE id = ?", id)
	return err
}
