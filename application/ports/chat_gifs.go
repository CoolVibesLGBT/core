package ports

import (
	"context"
	"core/application/types"
)

type ChatGIFPage struct {
	Documents  []types.ChatEmoticon `json:"documents"`
	NextOffset string               `json:"nextOffset"`
}
type ChatGIFSearch interface {
	SearchGIFs(context.Context, string, string) (*ChatGIFPage, error)
}
type ChatGIFMedia interface {
	PrepareGIF(context.Context, string) error
}
type ChatGIFProvider interface {
	ChatGIFSearch
	ChatGIFMedia
}
