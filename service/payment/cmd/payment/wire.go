//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"payment/internal/biz"
	"payment/internal/conf"
	"payment/internal/data"
	"payment/internal/server"
	"payment/internal/service"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
)

func wireApp(*conf.Server, *conf.Data, *conf.Registry, *conf.Service, *conf.Mq, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
