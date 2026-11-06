package service

import "github.com/google/wire"

// ProviderSet is service providers. The event-handler adapters implement the
// inbound ports consumed by internal/mq; outbox/consumer repos and the AMQP
// state come from data and mq respectively.
var ProviderSet = wire.NewSet(
	NewTodoService,
	NewShopService,
	NewAdminService,
	NewOrderEvents,
	NewTimeoutHandler,
	NewInventoryEvents,
	NewGoodsEvents,
)
