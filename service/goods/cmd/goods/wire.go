//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"github.com/google/wire"
	"goods/internal/biz"
	"goods/internal/conf"
	"goods/internal/data"
	"goods/internal/server"
	"goods/internal/service"

	"github.com/go-kratos/kratos/v3"
)

// initApp init kratos application.
func initApp(*conf.Server, *conf.Data, *conf.Registry, *conf.Service, *conf.Mq, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
