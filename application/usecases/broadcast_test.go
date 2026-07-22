package usecases

import (
	"context"
	"core/application/ports"
	"errors"
	"strings"
	"testing"
)

type broadcastGatewayFake struct {
	called   string
	provider ports.BroadcastProvider
	query    ports.BroadcastTrendingQuery
}

func (g *broadcastGatewayFake) FetchTrending(_ context.Context, provider ports.BroadcastProvider, query ports.BroadcastTrendingQuery) ([]byte, error) {
	g.called, g.provider, g.query = "fetch", provider, query
	return []byte(`{"ok":true}`), nil
}

func (g *broadcastGatewayFake) CreateBroadcast(context.Context, string) ([]byte, error) {
	g.called = "create"
	return []byte(`{}`), nil
}

func (g *broadcastGatewayFake) ViewBroadcast(context.Context, ports.BroadcastViewInput) ([]byte, error) {
	g.called = "view"
	return []byte(`{}`), nil
}

func (g *broadcastGatewayFake) RequestGuestBroadcast(context.Context, ports.BroadcastGuestRequest) ([]byte, error) {
	g.called = "guest"
	return []byte(`{}`), nil
}

func (g *broadcastGatewayFake) LikeBroadcast(context.Context, ports.BroadcastLikeInput) ([]byte, error) {
	g.called = "like"
	return []byte(`{}`), nil
}

func TestBroadcastServiceRequiresPrincipalForMutations(t *testing.T) {
	gateway := &broadcastGatewayFake{}
	service := NewBroadcastService(gateway)

	_, err := service.CreateBroadcast(context.Background(), ports.BroadcastPrincipal{}, "description")
	if !errors.Is(err, ports.ErrBroadcastUnauthorized) {
		t.Fatalf("CreateBroadcast() error = %v, want unauthorized", err)
	}
	if gateway.called != "" {
		t.Fatalf("gateway called without principal: %s", gateway.called)
	}
}

func TestBroadcastServiceAppliesTrendingDefaults(t *testing.T) {
	gateway := &broadcastGatewayFake{}
	service := NewBroadcastService(gateway)

	if _, err := service.FetchTrending(context.Background(), ports.BroadcastProviderHornet, ports.BroadcastTrendingQuery{}); err != nil {
		t.Fatalf("FetchTrending() error = %v", err)
	}
	if gateway.query.PageSize != 100 || gateway.query.Gender != "all" || gateway.query.Score != "0" {
		t.Fatalf("query defaults = %#v", gateway.query)
	}
}

func TestBroadcastServiceRejectsOversizedAndAbusiveInputs(t *testing.T) {
	service := NewBroadcastService(&broadcastGatewayFake{})
	principal := ports.BroadcastPrincipal{UserID: "user-id"}

	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "description",
			call: func() error {
				_, err := service.CreateBroadcast(context.Background(), principal, strings.Repeat("x", maxBroadcastDescriptionLength+1))
				return err
			},
		},
		{
			name: "broadcast id",
			call: func() error {
				_, err := service.ViewBroadcast(context.Background(), principal, strings.Repeat("x", maxBroadcastIdentifierLength+1))
				return err
			},
		},
		{
			name: "likes",
			call: func() error {
				_, err := service.LikeBroadcast(context.Background(), principal, "broadcast", "viewer", maxBroadcastLikes+1)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(); !errors.Is(err, ports.ErrBroadcastInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}
