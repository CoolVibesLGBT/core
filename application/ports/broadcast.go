package ports

import (
	"context"
	"errors"
)

type BroadcastProvider string

const (
	BroadcastProviderHornet BroadcastProvider = "hornet"
	BroadcastProviderGrowlr BroadcastProvider = "growlr"
)

var (
	ErrBroadcastIntegrationDisabled = errors.New("broadcast integration is disabled")
	ErrBroadcastProviderUnsupported = errors.New("broadcast provider is unsupported")
	ErrBroadcastInvalidInput        = errors.New("broadcast input is invalid")
	ErrBroadcastUpstream            = errors.New("broadcast upstream request failed")
	ErrBroadcastUnauthorized        = errors.New("broadcast operation requires authentication")
)

type BroadcastPrincipal struct {
	UserID string
}

func (p BroadcastPrincipal) Valid() bool {
	return p.UserID != ""
}

type BroadcastTrendingQuery struct {
	PageSize  int
	Gender    string
	Latitude  float64
	Longitude float64
	More      bool
	Score     string
}

type BroadcastViewInput struct {
	BroadcastID string
	Source      string
}

type BroadcastGuestRequest struct {
	BroadcastID    string
	StreamClientID string
}

type BroadcastLikeInput struct {
	BroadcastID string
	ViewerID    string
	NumLikes    int
}

// BroadcastGateway is the outbound boundary for external live providers.
// Implementations own HTTP, provider headers and credentials.
type BroadcastGateway interface {
	FetchTrending(context.Context, BroadcastProvider, BroadcastTrendingQuery) ([]byte, error)
	CreateBroadcast(context.Context, string) ([]byte, error)
	ViewBroadcast(context.Context, BroadcastViewInput) ([]byte, error)
	RequestGuestBroadcast(context.Context, BroadcastGuestRequest) ([]byte, error)
	LikeBroadcast(context.Context, BroadcastLikeInput) ([]byte, error)
}
