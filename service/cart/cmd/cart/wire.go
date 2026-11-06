//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"cart/internal/biz"
	"cart/internal/conf"
	"cart/internal/data"
	"cart/internal/server"
	"cart/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

func initApp(*conf.Server, *conf.Data, *conf.Registry, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
