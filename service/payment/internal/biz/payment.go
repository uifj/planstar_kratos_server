package biz

import (
	"context"
	"log/slog"

	"payment/internal/domain"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewPaymentUsecase)

type PaymentRepo interface {
	Create(ctx context.Context, p *domain.Payment) (*domain.Payment, error)
	GetByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error)
	MarkPaid(ctx context.Context, paymentNo, tradeNo string) (bool, error)
}

type PaymentUsecase struct {
	repo PaymentRepo
	log  *slog.Logger
}

func NewPaymentUsecase(repo PaymentRepo, logger *slog.Logger) *PaymentUsecase {
	return &PaymentUsecase{repo: repo, log: logger}
}

func (uc *PaymentUsecase) Create(ctx context.Context, p *domain.Payment) (*domain.Payment, error) {
	return uc.repo.Create(ctx, p)
}

func (uc *PaymentUsecase) GetByPaymentNo(ctx context.Context, paymentNo string) (*domain.Payment, error) {
	return uc.repo.GetByPaymentNo(ctx, paymentNo)
}

func (uc *PaymentUsecase) MarkPaid(ctx context.Context, paymentNo, tradeNo string) (bool, error) {
	return uc.repo.MarkPaid(ctx, paymentNo, tradeNo)
}
