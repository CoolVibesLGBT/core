package mcpserver

import (
	"core/application/mcpserver/tools"
	"core/mcp"
	services "core/services/user"
)

func NewServer(
	aiService *services.AIService,
	newsService *services.NewsService,
	placeService *services.PlaceService,
) *mcp.MCPServer {
	server := mcp.NewMCPServer()
	server.SetInstructions("Initialize the session, send notifications/initialized, inspect tools/list, then call tools/call with one of the registered tool names.")

	tools.RegisterAI(server, aiService)
	tools.RegisterNews(server, newsService)
	tools.RegisterPlace(server, placeService)

	return server
}
