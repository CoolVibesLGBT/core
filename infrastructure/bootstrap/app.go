package bootstrap

import (
	"core/helpers"
	"core/infrastructure/ai"
	"core/mcp"
	"core/routes"

	"gorm.io/gorm"
)

type App struct {
	DB            *gorm.DB
	Router        *routes.Router
	MCPServer     *mcp.MCPServer
	SnowFlakeNode *helpers.Node
	AIRegistry    *ai.Registry
}
