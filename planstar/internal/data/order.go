package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"planstar/internal/biz"
)

// OrderRepo 实现 biz.OrderRepo，落 planstar_order.order_info。
type OrderRepo struct {
	d *Data
}

func NewOrderRepo(d *Data) *OrderRepo { return &OrderRepo{d: d} }

func (r *OrderRepo) Orders(ctx context.Context, userID int64, status, orderType, pageIndex, pageSize int) ([]*biz.TicketOrder, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	where := "user_id = ?"
	args := []any{userID}
	if status > 0 {
		where += " AND status = ?"
		args = append(args, status)
	}
	if orderType > 0 {
		where += " AND order_type = ?"
		args = append(args, orderType)
	}
	args = append(args, pageSize, (pageIndex-1)*pageSize)

	rows, err := db.QueryContext(ctx,
		`SELECT order_id, order_number, COALESCE(activity_id,0), COALESCE(activity_event_id,0),
		        COALESCE(order_type,0), COALESCE(status,0), COALESCE(title,''), COALESCE(poster,''),
		        COALESCE(pay_amount,0), gmt_create
		   FROM order_info WHERE `+where+`
		ORDER BY order_id DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*biz.TicketOrder, 0, pageSize)
	for rows.Next() {
		o := &biz.TicketOrder{}
		var payAmount float64
		var created time.Time
		if err := rows.Scan(&o.OrderID, &o.OrderNumber, &o.ShowID, &o.SessionID, &o.OrderType, &o.Status,
			&o.Title, &o.Poster, &payAmount, &created); err != nil {
			return nil, err
		}
		o.ID = o.OrderID
		o.BuyType = 1
		o.CreateTime = biz.FmtDT(created)
		o.PayAmount = biz.Fen(payAmount)
		items = append(items, o)
	}
	return items, rows.Err()
}

func (r *OrderRepo) OrderByNumber(ctx context.Context, userID int64, orderNumber string) (*biz.TicketOrder, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}
	o := &biz.TicketOrder{}
	var payAmount float64
	var created time.Time
	err = db.QueryRowContext(ctx,
		`SELECT order_id, order_number, COALESCE(activity_id,0), COALESCE(activity_event_id,0),
		        COALESCE(order_type,0), COALESCE(status,0), COALESCE(title,''), COALESCE(poster,''),
		        COALESCE(pay_amount,0), gmt_create
		   FROM order_info WHERE order_number = ? AND user_id = ? LIMIT 1`, orderNumber, userID).
		Scan(&o.OrderID, &o.OrderNumber, &o.ShowID, &o.SessionID, &o.OrderType, &o.Status,
			&o.Title, &o.Poster, &payAmount, &created)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	o.ID = o.OrderID
	o.BuyType = 1
	o.CreateTime = biz.FmtDT(created)
	o.PayAmount = biz.Fen(payAmount)
	return o, nil
}

func (r *OrderRepo) CreateOrder(ctx context.Context, userID int64, in *biz.OrderInput) (*biz.TicketOrder, error) {
	db, err := r.d.DB("order")
	if err != nil {
		return nil, err
	}

	var title, poster string
	_ = db.QueryRowContext(ctx,
		`SELECT COALESCE(name,''), COALESCE(poster, img, '') FROM planstar_catalog.activity WHERE activity_id = ?`,
		in.ActivityID).Scan(&title, &poster)

	var orderID int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(order_id),0)+1 FROM order_info`).Scan(&orderID); err != nil {
		return nil, err
	}
	now := time.Now()
	orderNumber := fmt.Sprintf("PN%s%04d", now.Format("20060102150405"), now.Nanosecond()/100000)

	_, err = db.ExecContext(ctx,
		`INSERT INTO order_info
		   (order_id, order_number, user_id, activity_id, activity_event_id, order_type, status,
		    title, poster, pay_amount, total_amount, name, phone, user_address_id, gmt_create, gmt_modified)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		orderID, orderNumber, userID, in.ActivityID, in.ActivityEventID, in.OrderType,
		title, poster, fenToYuan(in.PayAmount), fenToYuan(in.TotalAmount), in.Name, in.Phone, in.AddressID)
	if err != nil {
		return nil, err
	}

	return &biz.TicketOrder{
		OrderID: orderID, ID: orderID, OrderNumber: orderNumber,
		ShowID: in.ActivityID, SessionID: in.ActivityEventID,
		CreateTime: biz.FmtDT(now), OrderType: in.OrderType, BuyType: 1, Status: 1,
		Title: title, Poster: poster, PayAmount: in.PayAmount,
	}, nil
}

func fenToYuan(v int64) float64 { return float64(v) / 100 }
