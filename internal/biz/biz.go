package biz

import "github.com/google/wire"

// ProviderSet is biz providers. Every mall usecase is a plain constructor
// over ports; the concrete implementations come from data and mq.
var ProviderSet = wire.NewSet(
	NewTodoUsecase,
	NewUserUsecase,
	NewAddressUsecase,
	NewGoodsUsecase,
	NewCartUsecase,
	NewOrderUsecase,
	NewInventoryUsecase,
	NewPaymentUsecase,
	NewPermissionUsecase,
)
