package usecases

import (
	"context"
	"core/application/ports"
	"fmt"
	"strings"
)

const (
	maxBroadcastPageSize          = 10_000
	maxBroadcastDescriptionLength = 2_000
	maxBroadcastIdentifierLength  = 256
	maxBroadcastLikes             = 1_000
)

type BroadcastService struct {
	gateway ports.BroadcastGateway
}

func NewBroadcastService(gateway ports.BroadcastGateway) *BroadcastService {
	return &BroadcastService{gateway: gateway}
}

func (s *BroadcastService) FetchTrending(ctx context.Context, provider ports.BroadcastProvider, query ports.BroadcastTrendingQuery) ([]byte, error) {
	if s == nil || s.gateway == nil {
		return nil, ports.ErrBroadcastIntegrationDisabled
	}
	if provider != ports.BroadcastProviderHornet && provider != ports.BroadcastProviderGrowlr {
		return nil, ports.ErrBroadcastProviderUnsupported
	}
	if query.PageSize <= 0 {
		query.PageSize = 100
	}
	if query.PageSize > maxBroadcastPageSize {
		return nil, fmt.Errorf("%w: page size exceeds %d", ports.ErrBroadcastInvalidInput, maxBroadcastPageSize)
	}
	if strings.TrimSpace(query.Gender) == "" {
		query.Gender = "all"
	}
	if strings.TrimSpace(query.Score) == "" {
		query.Score = "0"
	}
	return s.gateway.FetchTrending(ctx, provider, query)
}

func (s *BroadcastService) CreateBroadcast(ctx context.Context, principal ports.BroadcastPrincipal, description string) ([]byte, error) {
	if s == nil || s.gateway == nil {
		return nil, ports.ErrBroadcastIntegrationDisabled
	}
	if !principal.Valid() {
		return nil, ports.ErrBroadcastUnauthorized
	}
	description = strings.TrimSpace(description)
	if len(description) > maxBroadcastDescriptionLength {
		return nil, fmt.Errorf("%w: description is too long", ports.ErrBroadcastInvalidInput)
	}
	return s.gateway.CreateBroadcast(ctx, description)
}

func (s *BroadcastService) ViewBroadcast(ctx context.Context, principal ports.BroadcastPrincipal, broadcastID string) ([]byte, error) {
	if s == nil || s.gateway == nil {
		return nil, ports.ErrBroadcastIntegrationDisabled
	}
	if !principal.Valid() {
		return nil, ports.ErrBroadcastUnauthorized
	}
	broadcastID = strings.TrimSpace(broadcastID)
	if broadcastID == "" || len(broadcastID) > maxBroadcastIdentifierLength {
		return nil, fmt.Errorf("%w: broadcast ID is required", ports.ErrBroadcastInvalidInput)
	}
	return s.gateway.ViewBroadcast(ctx, ports.BroadcastViewInput{BroadcastID: broadcastID, Source: "trending"})
}

func (s *BroadcastService) RequestGuestBroadcast(ctx context.Context, principal ports.BroadcastPrincipal, broadcastID, streamClientID string) ([]byte, error) {
	if s == nil || s.gateway == nil {
		return nil, ports.ErrBroadcastIntegrationDisabled
	}
	if !principal.Valid() {
		return nil, ports.ErrBroadcastUnauthorized
	}
	input := ports.BroadcastGuestRequest{
		BroadcastID:    strings.TrimSpace(broadcastID),
		StreamClientID: strings.TrimSpace(streamClientID),
	}
	if input.BroadcastID == "" || input.StreamClientID == "" ||
		len(input.BroadcastID) > maxBroadcastIdentifierLength || len(input.StreamClientID) > maxBroadcastIdentifierLength {
		return nil, fmt.Errorf("%w: broadcast ID and stream client ID are required", ports.ErrBroadcastInvalidInput)
	}
	return s.gateway.RequestGuestBroadcast(ctx, input)
}

func (s *BroadcastService) LikeBroadcast(ctx context.Context, principal ports.BroadcastPrincipal, broadcastID, viewerID string, numLikes int) ([]byte, error) {
	if s == nil || s.gateway == nil {
		return nil, ports.ErrBroadcastIntegrationDisabled
	}
	if !principal.Valid() {
		return nil, ports.ErrBroadcastUnauthorized
	}
	input := ports.BroadcastLikeInput{
		BroadcastID: strings.TrimSpace(broadcastID),
		ViewerID:    strings.TrimSpace(viewerID),
		NumLikes:    numLikes,
	}
	if input.BroadcastID == "" || input.ViewerID == "" || input.NumLikes <= 0 || input.NumLikes > maxBroadcastLikes ||
		len(input.BroadcastID) > maxBroadcastIdentifierLength || len(input.ViewerID) > maxBroadcastIdentifierLength {
		return nil, fmt.Errorf("%w: broadcast ID, viewer ID and a positive like count are required", ports.ErrBroadcastInvalidInput)
	}
	return s.gateway.LikeBroadcast(ctx, input)
}
