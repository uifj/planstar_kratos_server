package mq

import "github.com/google/wire"

// ProviderSet wires the async adapter. The inbound event-handler ports
// (OrderEventHandler / InventoryEventHandler / GoodsEventHandler /
// OrderTimeoutHandler) are supplied by internal/service, so this package
// never imports a concrete usecase.
var ProviderSet = wire.NewSet(
	NewConfig,
	NewState,
	NewPublisher,
	NewServer,
)
