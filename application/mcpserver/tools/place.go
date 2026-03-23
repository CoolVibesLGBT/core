package tools

import (
	"context"
	"core/application/mcpserver/internal/shared"
	"core/constants"
	"core/mcp"

	services "core/services/user"
)

func RegisterPlace(server *mcp.MCPServer, placeService *services.PlaceService) {
	server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        constants.CMD_PLACE_FETCH,
			Title:       "Fetch Places",
			Description: "Fetch places using the same filter contract as the HTTP API.",
			InputSchema: shared.BaseFilterSchema(),
			Annotations: shared.ReadOnlyToolAnnotations("Fetch Places"),
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			filters, err := shared.BuildFilter(ctx, req.Arguments)
			if err != nil {
				return nil, err
			}

			places, cursorInfo, err := placeService.GetNearByPlaces(filters)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"places": places,
				"cursor": cursorInfo,
			}, nil
		},
	))

	server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        constants.CMD_PLACE_CATEGORIES,
			Title:       "List Place Categories",
			Description: "List place categories.",
			InputSchema: shared.BaseFilterSchema(),
			Annotations: shared.ReadOnlyToolAnnotations("List Place Categories"),
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			filters, err := shared.BuildFilter(ctx, req.Arguments)
			if err != nil {
				return nil, err
			}

			return placeService.GetPlacesCategories(filters)
		},
	))
}
