package tools

import (
	"context"
	"core/application/mcpserver/internal/shared"
	"core/constants"
	"core/mcp"
	"fmt"

	usecases "core/application/usecases"
	"github.com/google/uuid"
)

func RegisterNews(server *mcp.MCPServer, newsService *usecases.NewsService) error {
	if err := server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        constants.CMD_NEWS_FETCH,
			Title:       "Fetch News",
			Description: "Fetch news posts using the same filter contract as the HTTP API.",
			InputSchema: shared.BaseFilterSchema(),
			Annotations: shared.ReadOnlyToolAnnotations("Fetch News"),
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			filters, err := shared.BuildFilter(ctx, req.Arguments)
			if err != nil {
				return nil, err
			}

			postResult, err := newsService.GetNews(filters)
			if err != nil {
				return nil, err
			}

			return map[string]any{
				"news":   postResult.Posts,
				"cursor": postResult.Cursor,
			}, nil
		},
	)); err != nil {
		return err
	}

	if err := server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        constants.CMD_NEWS_GET,
			Title:       "Get News Item",
			Description: "Fetch a single news post by numeric post id or UUID.",
			InputSchema: shared.BaseFilterSchema("post_id"),
			Annotations: shared.ReadOnlyToolAnnotations("Get News Item"),
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			filters, err := shared.BuildFilter(ctx, req.Arguments)
			if err != nil {
				return nil, err
			}

			if filters.PostUUID == uuid.Nil && filters.PostID == 0 {
				return nil, fmt.Errorf("post_id is required")
			}

			return newsService.Get(filters)
		},
	)); err != nil {
		return err
	}

	return server.RegisterTool(mcp.NewTool(
		mcp.ToolDefinition{
			Name:        constants.CMD_NEWS_CATEGORIES,
			Title:       "List News Categories",
			Description: "List news categories.",
			InputSchema: shared.BaseFilterSchema(),
			Annotations: shared.ReadOnlyToolAnnotations("List News Categories"),
		},
		func(ctx context.Context, req mcp.CallToolRequest) (any, error) {
			filters, err := shared.BuildFilter(ctx, req.Arguments)
			if err != nil {
				return nil, err
			}

			return newsService.Categories(filters)
		},
	))
}
