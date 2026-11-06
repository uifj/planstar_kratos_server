//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	"shop/internal/biz"
	"shop/internal/conf"
	"shop/internal/data"
	"shop/internal/server"
	"shop/internal/service"
)

// initApp init shop application.
func initApp(*conf.Server, *conf.Data, *conf.Auth, *conf.Service, *conf.Registry, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
