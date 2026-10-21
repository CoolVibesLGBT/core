package handlers

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type rejectedPostInteractionRepository struct {
	ports.PostRepository
	err error
}

func (r *rejectedPostInteractionRepository) Like(types.Filter) error         { return r.err }
func (r *rejectedPostInteractionRepository) Dislike(types.Filter) error      { return r.err }
func (r *rejectedPostInteractionRepository) Banana(types.Filter) error       { return r.err }
func (r *rejectedPostInteractionRepository) Bookmark(types.Filter) error     { return r.err }
func (r *rejectedPostInteractionRepository) View(types.Filter) (bool, error) { return false, r.err }

func TestPostInteractionsReportBlocksMissingPostsAndTimeouts(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"blocked", domainuser.ErrInteractionBlocked, http.StatusForbidden},
		{"unavailable", domainuser.ErrInteractionUnavailable, http.StatusNotFound},
		{"missing", constants.ErrPostPublicIDNotFound.Format(42), http.StatusNotFound},
		{"timeout", context.DeadlineExceeded, http.StatusGatewayTimeout},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := usecases.NewPostService(nil, &rejectedPostInteractionRepository{err: test.err}, nil)
			for _, route := range []struct {
				name    string
				handler fiber.Handler
			}{
				{"like", HandlePostLike(service)}, {"dislike", HandlePostDislike(service)},
				{"banana", HandlePostBanana(service)}, {"bookmark", HandlePostBookmark(service)}, {"view", HandlePostView(service)},
			} {
				t.Run(route.name, func(t *testing.T) {
					response := performMultipartHandlerRequest(t, route.handler, actor, map[string]string{"post_id": "42"}, nil)
					defer func() {
						if err := response.Body.Close(); err != nil {
							t.Errorf("close response body: %v", err)
						}
					}()
					if response.StatusCode != test.status {
						t.Fatalf("status=%d want %d", response.StatusCode, test.status)
					}
				})
			}
		})
	}
}
