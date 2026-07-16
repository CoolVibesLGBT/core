package mcpserver

import (
	"core/application/mcpserver/tools"
	usecases "core/application/usecases"
	"core/mcp"
	"errors"
)

func NewServer(
	aiService *usecases.AIService,
	newsService *usecases.NewsService,
	placeService *usecases.PlaceService,
) (*mcp.MCPServer, error) {
	server := mcp.NewMCPServer()
	server.SetInstructions("Initialize the session, send notifications/initialized, inspect tools/list, then call tools/call with one of the registered tool names.")

	if err := errors.Join(
		tools.RegisterAI(server, aiService),
		tools.RegisterNews(server, newsService),
		tools.RegisterPlace(server, placeService),
	); err != nil {
		return nil, err
	}

	return server, nil
}
