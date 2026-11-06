package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent"
	"github.com/go-kratos/kratos-layout/internal/data/ent/payment"
)

func paymentToBiz(po *ent.Payment) *biz.Payment {
	if po == nil {
		return nil
	}
	return &biz.Payment{
		ID: idStr(po.ID), PaymentNo: po.PaymentNo, OrderSn: po.OrderSn,
		UserID: po.UserID, Amount: po.Amount, Channel: po.Channel,
		Status: po.Status,
	}
}

type paymentRepo struct{ data *Data }

func NewPaymentRepo(d *Data) biz.PaymentRepo { return &paymentRepo{data: d} }

func (r *paymentRepo) Create(ctx context.Context, p *biz.Payment) (*biz.Payment, error) {
	po, err := r.data.db.Payment.Create().
		SetID(parseID(p.ID)).SetPaymentNo(p.PaymentNo).SetOrderSn(p.OrderSn).
		SetUserID(p.UserID).SetAmount(p.Amount).SetChannel(p.Channel).
		SetStatus(p.Status).Save(ctx)
	if err != nil {
		if isConstraint(err) {
			return nil, biz.ErrPaymentStatusConflict
		}
		return nil, err
	}
	return paymentToBiz(po), nil
}

func (r *paymentRepo) FindByNo(ctx context.Context, paymentNo string) (*biz.Payment, error) {
	po, err := r.data.db.Payment.Query().
		Where(payment.PaymentNoEQ(paymentNo)).Only(ctx)
	if err != nil {
		if isNotFound(err) {
			return nil, biz.ErrPaymentNotFound
		}
		return nil, err
	}
	return paymentToBiz(po), nil
}

// MarkSuccessWithEvent flips a pending payment and writes the payment.success
// outbox row in one database transaction, so the event can never be lost.
func (r *paymentRepo) MarkSuccessWithEvent(ctx context.Context, paymentNo string, outbox *biz.OutboxEvent) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	n, err := tx.Payment.Update().
		Where(payment.PaymentNoEQ(paymentNo), payment.StatusEQ(biz.PaymentStatusPending)).
		SetStatus(biz.PaymentStatusSuccess).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if n == 0 {
		_ = tx.Rollback()
		return biz.ErrPaymentStatusConflict
	}
	if outbox != nil {
		if _, err := tx.OrderEventOutbox.Create().
			SetID(parseID(outbox.ID)).SetEventID(outbox.EventID).
			SetEventType(outbox.EventType).SetOrderSn(outbox.OrderSn).
			SetPayload(outbox.Payload).SetStatus(outbox.Status).
			SetRetryCount(outbox.RetryCount).Save(ctx); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
