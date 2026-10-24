package handlers

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"github.com/gofiber/fiber/v3"
	"net/http/httptest"
	"strings"
	"testing"
)

type groupListStub struct {
	query ports.PublicGroupQuery
	calls int
}

func (s *groupListStub) ListPublicGroups(_ context.Context, q ports.PublicGroupQuery) ([]types.ChatGroupPreview, *string, error) {
	s.query = q
	s.calls++
	return []types.ChatGroupPreview{}, nil, nil
}
func TestPublicGroupListAndGlobalSearch(t *testing.T) {
	for _, tc := range []struct {
		path, body string
		status     int
		calls      int
	}{
		{"/groups", "query=community&limit=5", 200, 1},
		{"/groups", "cursor=bad", 400, 0},
		{"/groups", "limit=0", 400, 0},
		{"/search", "query=community&scope=groups&limit=5", 200, 1},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			stub := &groupListStub{}
			app := fiber.New()
			app.Post("/groups", HandleListPublicGroups(stub))
			app.Post("/search", HandleGlobalSearch(nil, nil, stub))
			req := httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := resp.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			if resp.StatusCode != tc.status || stub.calls != tc.calls {
				t.Fatalf("status=%d calls=%d", resp.StatusCode, stub.calls)
			}
			if tc.calls > 0 && (stub.query.Search != "community" || stub.query.Limit != 5) {
				t.Fatalf("query=%+v", stub.query)
			}
		})
	}
}
