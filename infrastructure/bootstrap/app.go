package bootstrap

import (
	"core/adapters/inbound/http/routes"
	usecases "core/application/usecases"
	"core/helpers"
	"core/infrastructure/ai"
	telegramService "core/infrastructure/bot/telegram"
	"core/infrastructure/db"
	"core/infrastructure/repositories"
	"core/mcp"
	broadcastworker "core/workers/broadcast"
	mediaworker "core/workers/media"
	newsworker "core/workers/news"
	"errors"

	"github.com/oschwald/maxminddb-golang"
	"gorm.io/gorm"
)

type App struct {
	DB            *gorm.DB
	Router        *routes.Router
	MCPServer     *mcp.MCPServer
	SnowFlakeNode *helpers.Node
	AIRegistry    *ai.Registry
	GEOIPDB       *maxminddb.Reader

	UserService            *usecases.UserService
	NewsService            *usecases.NewsService
	ChatService            *usecases.ChatService
	BroadcastService       *usecases.BroadcastService
	TelegramService        *telegramService.Service
	NotificationRepository *repositories.NotificationRepository

	MediaProcessorRepository    mediaworker.Repository
	MediaProcessingObserver     mediaworker.ProcessingObserver
	NewsWorkerDependencies      newsworker.Dependencies
	BroadcastWorkerDependencies broadcastworker.Dependencies
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	var geoErr error
	if a.GEOIPDB != nil {
		geoErr = a.GEOIPDB.Close()
	}
	return errors.Join(geoErr, db.Close(a.DB))
}
