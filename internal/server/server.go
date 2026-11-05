package server

import (
	"github.com/google/wire"
)

// ProviderSet is server providers. NewMiddlewares composes every documented
// Kratos middleware; the two server constructors only apply the composed
// chains and register the generated services.
var ProviderSet = wire.NewSet(NewMiddlewares, NewGRPCServer, NewHTTPServer)
