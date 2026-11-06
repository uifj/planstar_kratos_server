package server

import (
	"github.com/google/wire"
)

// ProviderSet is server providers. Consul-based service registration has been
// removed: the module exposes its gRPC listener directly.
var ProviderSet = wire.NewSet(NewGRPCServer)
