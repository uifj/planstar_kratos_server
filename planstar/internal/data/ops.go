package data

import (
	"context"
	"database/sql"

	"planstar/internal/biz"
)

// OpsRepo 实现 biz.OpsRepo，运营写操作落各业务库（community/shop/payment/order）。
type OpsRepo struct {
	d *Data
}

func NewOpsRepo(d *Data) *OpsRepo { return &OpsRepo{d: d} }

func (r *OpsRepo) SetFeedStatus(ctx context.Context, id int64, status int32) error {
	db, err := r.d.DB("community")
	if err != nil {
		return err
	}
	return setOpsStatus(ctx, db,
		`SELECT 1 FROM feed WHERE feed_id = ?`,
		`UPDATE feed SET status = ?, gmt_modified = NOW() WHERE feed_id = ?`,
		status, id)
}

func (r *OpsRepo) SetGoodsStatus(ctx context.Context, id int64, status int32) error {
	db, err := r.d.DB("shop")
	if err != nil {
		return err
	}
	return setOpsStatus(ctx, db,
		`SELECT 1 FROM goods WHERE goods_id = ?`,
		`UPDATE goods SET status = ?, gmt_modified = NOW() WHERE goods_id = ?`,
		status, id)
}

func (r *OpsRepo) SetGoodsSKUStatus(ctx context.Context, id int64, status int32) error {
	db, err := r.d.DB("shop")
	if err != nil {
		return err
	}
	return setOpsStatus(ctx, db,
		`SELECT 1 FROM goods_sku WHERE sku_id = ?`,
		`UPDATE goods_sku SET status = ?, gmt_modified = NOW() WHERE sku_id = ?`,
		status, id)
}

// setOpsStatus 先确认行存在再更新状态。不用 RowsAffected 判存在：MySQL 默认返回
// 「被改变的行数」，把状态设成原值时为 0，会被误判成不存在。
func setOpsStatus(ctx context.Context, db *sql.DB, existsQuery, updateQuery string, status int32, id int64) error {
	var one int
	err := db.QueryRowContext(ctx, existsQuery, id).Scan(&one)
	if err == sql.ErrNoRows {
		return biz.ErrNotFound
	}
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, updateQuery, status, id)
	return err
}

// ReviewRefund 单事务审核退款单：approve 置 success 并把订单置为已退款(7)、回写已退金额；
// reject 仅置 rejected，订单保持不变。退款与订单分属 payment/order 两库，但同一 MySQL
// 实例，故可在一个事务内用跨库限定名更新（与 Redeem 的跨库写法一致）。
func (r *OpsRepo) ReviewRefund(ctx context.Context, id int64, approve bool) (*biz.RefundReview, error) {
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
		refundNo string
		orderID  int64
		amount   float64
		status   string
	)
	err = tx.QueryRowContext(ctx,
		`SELECT refund_no, order_id, refundable_amount, status
		   FROM order_refund WHERE id = ? FOR UPDATE`, id).
		Scan(&refundNo, &orderID, &amount, &status)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if status == "success" || status == "rejected" {
		return nil, biz.ErrRefundNotAllowed
	}

	var orderStatus int32
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM planstar_order.order_info WHERE order_id = ? FOR UPDATE`, orderID).
		Scan(&orderStatus)
	if err == sql.ErrNoRows {
		return nil, biz.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	out := &biz.RefundReview{
		RefundID:     id,
		RefundNo:     refundNo,
		OrderID:      orderID,
		RefundAmount: biz.Fen(amount),
		OrderStatus:  orderStatus,
	}
	if approve {
		if _, err := tx.ExecContext(ctx,
			`UPDATE order_refund SET status = 'success', gmt_modified = NOW() WHERE id = ?`, id); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE planstar_order.order_info
			    SET status = 7, refund_amount = ?, gmt_modified = NOW()
			  WHERE order_id = ?`, amount, orderID); err != nil {
			return nil, err
		}
		out.Status = "success"
		out.OrderStatus = 7
	} else {
		if _, err := tx.ExecContext(ctx,
			`UPDATE order_refund SET status = 'rejected', gmt_modified = NOW() WHERE id = ?`, id); err != nil {
			return nil, err
		}
		out.Status = "rejected"
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}