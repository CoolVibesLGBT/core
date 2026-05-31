package bootstrap

import (
	"core/ai"
	"core/helpers"
	"core/mcp"
	"core/routes"

	socketio "github.com/vchitai/go-socket.io/v4"
	"gorm.io/gorm"
)

type App struct {
	DB            *gorm.DB
	Router        *routes.Router
	MCPServer     *mcp.MCPServer
	SnowFlakeNode *helpers.Node
	SocketServer  *socketio.Server
	AIRegistry    *ai.Registry
}
