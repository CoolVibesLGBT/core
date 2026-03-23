//go:build wireinject
// +build wireinject

// go run github.com/google/wire/cmd/wire ./application
package application

import (
	"core/ai"
	"core/application/mcpserver"
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
		ai.ProviderSet,
		db.ProviderSet,
		helpers.ProviderSet,
		repositories.ProviderSet,
		services.ProviderSet,
		socket.ProviderSet,
		mcpserver.NewServer,
		routes.ProviderSet,
		wire.Struct(new(App), "DB", "Router", "MCPServer", "SnowFlakeNode", "AIRegistry"),
	)
	return nil, nil
}

func InitializeMCPOnly() (*mcp.MCPServer, error) {
	wire.Build(
		ai.ProviderSet,
		db.ProviderSet,
		helpers.ProviderSet,
		repositories.ProviderSet,
		services.ProviderSet,
		routes.GeoIPDBProvider,
		mcpserver.NewServer,
	)
	return nil, nil
}
