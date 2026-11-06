//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"inventory/internal/biz"
	"inventory/internal/conf"
	"inventory/internal/data"
	"inventory/internal/server"
	"inventory/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

func initApp(*conf.Server, *conf.Data, *conf.Mq, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
