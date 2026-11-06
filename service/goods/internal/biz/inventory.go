package biz

import (
	"context"
	"log/slog"
	"goods/internal/domain"
)

type InventoryRepo interface {
	Create(context.Context, *domain.Inventory) (*domain.Inventory, error)
}

type InventoryUsecase struct {
	repo InventoryRepo
	log  *slog.Logger
}

func NewInventoryUsecase(repo InventoryRepo, logger *slog.Logger) *InventoryUsecase {
	return &InventoryUsecase{repo: repo, log: logger}
}
