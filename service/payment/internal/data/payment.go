package data

import (
	"context"
	"database/sql"
	serrors "errors"
	"log/slog"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"

	"payment/internal/domain"
)

const paymentColumns = "id, payment_no, order_sn, user_id, amount, channel, status, trade_no, paid_at"

type paymentRepo struct {
	data *Data
	log  *slog.Logger
}

// scanPayment reads one payment row. paid_at is nullable.
func scanPayment(row interface{ Scan(dest ...any) error }) (*domain.Payment, error) {
	var (
		p      domain.Payment
		paidAt sql.NullTime
	)
	if err := row.Scan(&p.ID, &p.PaymentNo, &p.OrderSn, &p.UserID, &p.Amount,
		&p.Channel, &p.Status, &p.TradeNo, &paidAt); err != nil {
		return nil, err
	}
	if paidAt.Valid {
		p.PaidAt = &paidAt.Time
	}
	return &p, nil
}

func (r *paymentRepo) Create(ctx context.Context, p *domain.Payment) (*domain.Payment, error) {
	res, err := r.data.DB(ctx).ExecContext(ctx,
		"INSERT INTO payments (payment_no, order_sn, user_id, amount, channel, status, created_at) "+
			"VALUES (?, ?, ?, ?, ?, 1, NOW())",
		p.PaymentNo, p.OrderSn, p.UserID, p.Amount, p.Channel,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	p.ID = id
	p.Status = 1
	return p, nil
}

func (r *paymentRepo) GetByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error) {
	row := r.data.DB(ctx).QueryRowContext(ctx,
		"SELECT "+paymentColumns+" FROM payments WHERE payment_no = ?", paymentNo)
	p, err := scanPayment(row)
	if serrors.Is(err, sql.ErrNoRows) {
		return nil, kerrors.New(400, "PAYMENT_NOT_FOUND", "支付单不存在")
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *paymentRepo) MarkPaid(ctx context.Context, paymentNo, tradeNo string) (bool, error) {
	res, err := r.data.DB(ctx).ExecContext(ctx,
		"UPDATE payments SET status = 2, trade_no = ?, paid_at = ? WHERE payment_no = ? AND status = 1",
		tradeNo, time.Now(), paymentNo)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		var status int
		err := r.data.DB(ctx).QueryRowContext(ctx,
			"SELECT status FROM payments WHERE payment_no = ?", paymentNo,
		).Scan(&status)
		if serrors.Is(err, sql.ErrNoRows) {
			return false, kerrors.New(400, "PAYMENT_NOT_FOUND", "支付单不存在")
		}
		if err != nil {
			return false, err
		}
		if status == 2 {
			return false, nil // 幂等：已支付
		}
		return false, kerrors.New(400, "PAYMENT_NOT_FOUND", "支付单不存在")
	}
	return true, nil
}
