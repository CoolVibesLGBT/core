package tools

import (
	"context"
	"core/adapters/inbound/mcpserver/internal/shared"
	"core/constants"
	"core/mcp"

	usecases "core/application/usecases"
)

func RegisterPlace(server *mcp.MCPServer, placeService *usecases.PlaceService) error {
	if err := server.RegisterTool(mcp.NewTool(
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
	)); err != nil {
		return err
	}

	return server.RegisterTool(mcp.NewTool(
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
