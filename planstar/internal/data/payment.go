package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"planstar/internal/biz"
)

// payOrderColumns 是支付单查询列（金额为元，时间字段由 DATETIME 解析）。
const payOrderColumns = `pay_no, order_number, amount, channel, status,
	COALESCE(third_trade_no,''), pay_params, expire_at, paid_at`

// PaymentRepo 实现 biz.PaymentRepo，落 planstar_payment.pay_order / order_refund。
type PaymentRepo struct {
	d *Data
}

func NewPaymentRepo(d *Data) *PaymentRepo { return &PaymentRepo{d: d} }

func (r *PaymentRepo) CreatePayment(ctx context.Context, userID int64, in *biz.PayInput, now time.Time) (*biz.PaymentOrder, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var (
		orderID     int64
		orderNumber string
		status      int
		payAmount   float64
	)
	err = tx.QueryRowContext(ctx,
		`SELECT order_id, order_number, COALESCE(status,0), COALESCE(pay_amount,0)
		   FROM planstar_order.order_info
		  WHERE order_number = ? AND user_id = ? LIMIT 1 FOR UPDATE`,
		in.OrderNumber, userID).Scan(&orderID, &orderNumber, &status, &payAmount)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// 已关闭 / 已退款的订单不再允许发起支付。
	if status == 6 || status == 7 {
		return nil, biz.ErrConflict
	}

	existing, err := paymentByOrderChannel(ctx, tx, userID, orderNumber, in.Channel)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// 已过期（读取时可能已落为 expired）或超过有效期：复用同一支付单并重新计时。
		if existing.Status == biz.PaymentStatusExpired ||
			(existing.Status == biz.PaymentStatusPending && isExpired(existing, now)) {
			expireAt := now.Add(biz.DefaultPaymentTTL)
			if _, err := tx.ExecContext(ctx,
				`UPDATE pay_order SET status = 'pending', expire_at = ?, paid_at = NULL, gmt_modified = NOW()
				  WHERE pay_no = ?`, expireAt, existing.PayNo); err != nil {
				return nil, err
			}
			existing.Status = biz.PaymentStatusPending
			existing.ExpireAt = expireAt.UnixMilli()
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return existing, nil
	}

	expireAt := now.Add(biz.DefaultPaymentTTL)
	amount := biz.Fen(payAmount)
	payNo := newPayNo(time.Now())
	for attempt := 0; attempt < 3; attempt++ {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO pay_order
			   (pay_no, order_number, order_id, user_id, channel, amount, status, third_trade_no, pay_params, expire_at, gmt_create, gmt_modified)
			 VALUES (?, ?, ?, ?, ?, ?, ?, '', '{}', ?, NOW(), NOW())`,
			payNo, orderNumber, orderID, userID, in.Channel, fenToYuan(amount),
			biz.PaymentStatusPending, expireAt)
		if err == nil {
			break
		}
		if !isDuplicateKey(err) {
			return nil, err
		}
		// 并发同订单同渠道：回滚后返回先落库的支付单。
		if dup, qerr := paymentByOrderChannel(ctx, tx, userID, orderNumber, in.Channel); qerr == nil && dup != nil {
			_ = tx.Rollback()
			return dup, nil
		}
		payNo = newPayNo(time.Now())
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &biz.PaymentOrder{
		PayNo: payNo, OrderNumber: orderNumber, Amount: amount, Channel: in.Channel,
		Status: biz.PaymentStatusPending, PayParams: map[string]any{},
		ExpireAt: expireAt.UnixMilli(),
	}, nil
}

func (r *PaymentRepo) PaymentDetail(ctx context.Context, userID int64, payNo, orderNumber string) (*biz.PaymentOrder, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	var row *sql.Row
	if payNo != "" {
		row = db.QueryRowContext(ctx,
			`SELECT `+payOrderColumns+` FROM pay_order WHERE pay_no = ? AND user_id = ? LIMIT 1`,
			payNo, userID)
	} else {
		row = db.QueryRowContext(ctx,
			`SELECT `+payOrderColumns+` FROM pay_order WHERE order_number = ? AND user_id = ?
			  ORDER BY id DESC LIMIT 1`, orderNumber, userID)
	}
	p, err := scanPaymentRow(row)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	// 读取时把已过期仍为 pending 的支付单落为 expired，保持库内状态与返回一致。
	if p.Status == biz.PaymentStatusPending && isExpired(p, time.Now()) {
		_, _ = db.ExecContext(ctx,
			`UPDATE pay_order SET status = 'expired', gmt_modified = NOW()
			  WHERE pay_no = ? AND status = 'pending' AND expire_at <= NOW()`, p.PayNo)
		p.Status = biz.PaymentStatusExpired
	}
	return p, nil
}

func (r *PaymentRepo) CreateRefund(ctx context.Context, userID, orderID int64, in *biz.RefundInput, now time.Time) (*biz.Refund, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var (
		orderNumber string
		status      int
		payAmount   float64
	)
	err = tx.QueryRowContext(ctx,
		`SELECT order_number, COALESCE(status,0),
		        COALESCE(NULLIF(pay_amount,0), total_amount, 0)
		   FROM planstar_order.order_info
		  WHERE order_id = ? AND user_id = ? LIMIT 1 FOR UPDATE`,
		orderID, userID).Scan(&orderNumber, &status, &payAmount)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// 幂等：同订单已存在退款单时直接返回（即使订单状态随后变为已退款）。
	existing, err := refundByOrder(ctx, tx, userID, orderID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return existing, nil
	}

	// OrderStatus: 1..5 可退款；0 未知、6 已关闭、7 已退款均不可退。
	if status < 1 || status > 5 {
		return nil, biz.ErrRefundNotAllowed
	}
	amount := biz.Fen(payAmount)
	if amount <= 0 {
		return nil, biz.ErrRefundNotAllowed
	}

	refund := &biz.Refund{
		RefundNo:            newRefundNo(now),
		OrderID:             orderID,
		RefundableAmount:    amount,
		Status:              "applied",
		ExpectedArrivalDays: biz.DefaultRefundArrivalDays,
	}
	for attempt := 0; attempt < 3; attempt++ {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO order_refund
			   (refund_no, order_id, order_number, user_id, reason_code, reason_text,
			    refundable_amount, status, expected_arrival_days, gmt_create, gmt_modified)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`,
			refund.RefundNo, orderID, orderNumber, userID, in.ReasonCode, in.ReasonText,
			fenToYuan(amount), refund.Status, refund.ExpectedArrivalDays)
		if err == nil {
			break
		}
		if !isDuplicateKey(err) {
			return nil, err
		}
		// 并发重复提交：回滚后返回先落库的退款单。
		if dup, qerr := refundByOrder(ctx, tx, userID, orderID); qerr == nil && dup != nil {
			_ = tx.Rollback()
			return dup, nil
		}
		refund.RefundNo = newRefundNo(time.Now())
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return refund, nil
}

func (r *PaymentRepo) RefundByOrder(ctx context.Context, userID, orderID int64) (*biz.Refund, error) {
	db, err := r.d.DB("payment")
	if err != nil {
		return nil, err
	}
	refund, err := refundByOrder(ctx, db, userID, orderID)
	if err != nil {
		return nil, err
	}
	if refund == nil {
		return nil, biz.ErrNotFound
	}
	return refund, nil
}

// paymentByOrderChannel 按（用户 + 订单号 + 渠道）查支付单；不存在返回 (nil, nil)。
func paymentByOrderChannel(ctx context.Context, q rowQuerier, userID int64, orderNumber, channel string) (*biz.PaymentOrder, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+payOrderColumns+` FROM pay_order
		  WHERE order_number = ? AND user_id = ? AND channel = ? LIMIT 1`,
		orderNumber, userID, channel)
	p, err := scanPaymentRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

// refundByOrder 按（用户 + 订单）查退款单；不存在返回 (nil, nil)。
func refundByOrder(ctx context.Context, q rowQuerier, userID, orderID int64) (*biz.Refund, error) {
	refund := &biz.Refund{}
	var amount float64
	err := q.QueryRowContext(ctx,
		`SELECT refund_no, order_id, refundable_amount, status, expected_arrival_days
		   FROM order_refund WHERE order_id = ? AND user_id = ? LIMIT 1`,
		orderID, userID).
		Scan(&refund.RefundNo, &refund.OrderID, &amount, &refund.Status, &refund.ExpectedArrivalDays)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	refund.RefundableAmount = biz.Fen(amount)
	return refund, nil
}

// scanPaymentRow 读取一行支付单。
func scanPaymentRow(row *sql.Row) (*biz.PaymentOrder, error) {
	p := &biz.PaymentOrder{}
	var (
		amount float64
		third  string
		params []byte
		expire time.Time
		paid   sql.NullTime
	)
	if err := row.Scan(&p.PayNo, &p.OrderNumber, &amount, &p.Channel, &p.Status,
		&third, &params, &expire, &paid); err != nil {
		return nil, err
	}
	p.Amount = biz.Fen(amount)
	p.ThirdTradeNo = third
	p.ExpireAt = expire.UnixMilli()
	if paid.Valid {
		p.PaidAt = paid.Time.UnixMilli()
	}
	p.PayParams = map[string]any{}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &p.PayParams)
	}
	return p, nil
}

// isExpired 判断 pending 支付单是否已过有效期。
func isExpired(p *biz.PaymentOrder, now time.Time) bool {
	return p.ExpireAt > 0 && now.UnixMilli() >= p.ExpireAt
}

// newPayNo 生成 PY + 毫秒时间戳 + 6 位随机数的支付单号（配合唯一索引保证唯一）。
func newPayNo(now time.Time) string {
	return fmt.Sprintf("PY%d%06d", now.UnixMilli(), rand.Intn(1000000))
}

// newRefundNo 生成 RF + 毫秒时间戳 + 6 位随机数的退款单号。
func newRefundNo(now time.Time) string {
	return fmt.Sprintf("RF%d%06d", now.UnixMilli(), rand.Intn(1000000))
}
