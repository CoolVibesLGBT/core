//go:build wireinject
// +build wireinject

// go run github.com/google/wire/cmd/wire ./application
package application

import (
	"core/helpers"
	"core/mcp"
	"core/repositories"
	"core/routes"
	"core/services/db"
	"core/services/socket"
	services "core/services/user"

	wire "github.com/google/wire"
)

func InitializeApp() (*App, error) {
	wire.Build(
		db.ProviderSet,
		helpers.ProviderSet,
		repositories.ProviderSet,
		services.ProviderSet,
		socket.ProviderSet,
		mcp.NewMCPServer,
		routes.ProviderSet,
		wire.Struct(new(App), "DB", "Router", "SnowFlakeNode"),
	)
	return nil, nil
}
