package application

import (
	"core/helpers"
	"core/routes"

	socketio "github.com/vchitai/go-socket.io/v4"
	"gorm.io/gorm"
)

type App struct {
	DB            *gorm.DB
	Router        *routes.Router
	SnowFlakeNode *helpers.Node
	SocketServer  *socketio.Server
}
