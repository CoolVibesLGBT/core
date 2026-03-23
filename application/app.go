package application

import (
	"core/helpers"
	"core/mcp"
	"core/routes"

	socketio "github.com/vchitai/go-socket.io/v4"
	"google.golang.org/genai"
	"gorm.io/gorm"
)

type App struct {
	DB            *gorm.DB
	Router        *routes.Router
	MCPServer     *mcp.MCPServer
	SnowFlakeNode *helpers.Node
	SocketServer  *socketio.Server
	GenAIClient   *genai.Client
}
