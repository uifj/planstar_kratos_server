//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package main

import (
	"log/slog"

	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	"user/internal/biz"
	"user/internal/conf"
	"user/internal/data"
	"user/internal/server"
	"user/internal/service"
)

// initApp init kratos application.
func initApp(*conf.Server, *conf.Data, *conf.Registry, *slog.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
