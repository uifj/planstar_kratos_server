//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"github.com/google/wire"
	"order/internal/biz"
	"order/internal/conf"
	"order/internal/data"
	"order/internal/server"
	"order/internal/service"

	"github.com/go-kratos/kratos/v3"
)

// wireApp init kratos application.
func wireApp(*conf.Server, *conf.Data, *conf.Registry, *conf.Auth, *conf.Service, *conf.Mq, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
