package server

import "github.com/google/wire"

// ProviderSet is server providers. Consul-based service registration has been
// dropped: the target stack runs the modules as plain gRPC services.
var ProviderSet = wire.NewSet(NewGRPCServer)
