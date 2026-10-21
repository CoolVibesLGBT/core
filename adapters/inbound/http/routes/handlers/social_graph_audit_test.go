package handlers

import (
	"bytes"
	"context"
	"core/application/ports"
	"core/application/types"
	"core/application/usecases"
	domainuser "core/domain/user"
	"core/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func TestFollowReturnsForbiddenForBlockAndNotFoundForUnavailableUser(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	target := &models.User{ID: uuid.New(), PublicID: 20}
	for _, test := range []struct {
		err    error
		status int
	}{
		{domainuser.ErrInteractionBlocked, http.StatusForbidden},
		{domainuser.ErrInteractionUnavailable, http.StatusNotFound},
	} {
		repo := &handlerEngagementRepo{reciprocalErr: test.err}
		service := usecases.NewUserService(&handlerUserRepo{byPublicID: map[int64]*models.User{10: actor, 20: target}},
			&handlerPostRepo{}, &handlerMediaRepo{}, repo, &handlerNotificationRepo{})
		response := performMultipartHandlerRequest(t, HandleFollow(service), actor, map[string]string{"followee_id": "20"}, nil)
		if response.StatusCode != test.status {
			t.Fatalf("follow with %v = %d; want %d", test.err, response.StatusCode, test.status)
		}
		_ = response.Body.Close()
	}
}

func TestPrivateSocialListsReturnForbiddenBeforeRepositoryRead(t *testing.T) {
	owner := &models.User{ID: uuid.New(), PublicID: 20}
	visitor := &models.User{ID: uuid.New(), PublicID: 10}
	for _, kind := range []string{"like_given", "dislike_given", "matched", "blocking", "blocked_by"} {
		repo := &handlerEngagementRepo{}
		service := usecases.NewUserService(&handlerUserRepo{byPublicID: map[int64]*models.User{20: owner}},
			&handlerPostRepo{}, &handlerMediaRepo{}, repo, &handlerNotificationRepo{})
		response := performMultipartHandlerRequest(t, HandleFetchUserEngagements(service), visitor,
			map[string]string{"user_id": "20", "engagement_type": kind}, nil)
		if response.StatusCode != http.StatusForbidden || repo.getCalls != 0 {
			t.Fatalf("private list %s returned %d with %d repository reads", kind, response.StatusCode, repo.getCalls)
		}
		_ = response.Body.Close()
	}
}

type matchAuditRepository struct {
	ports.MatchesRepository
	reaction domainuser.MatchReaction
	limit    int
}

func (r *matchAuditRepository) RecordView(_ context.Context, _, _ uuid.UUID, reaction domainuser.MatchReaction) (bool, error) {
	r.reaction = reaction
	return false, nil
}

func (r *matchAuditRepository) GetLikesAfter(_ context.Context, _ uuid.UUID, _ *types.MatchListCursor, limit int) (types.MatchListPage, error) {
	r.limit = limit
	return types.MatchListPage{Users: []types.PublicUserSummary{}}, nil
}

func TestMatchActionsAcceptJSONIncludingLikePagination(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	repo := &matchAuditRepository{}
	service := usecases.NewMatchService(&handlerUserRepo{}, repo)
	for _, test := range []struct {
		handler fiber.Handler
		body    string
	}{
		{HandleRecordView(service), `{"public_id":"20","reaction":"like"}`},
		{HandleGetLikesAfter(service), `{"limit":3}`},
	} {
		app := fiber.New()
		app.Post("/", func(c fiber.Ctx) error {
			c.Locals("authenticatedUser", actor)
			return test.handler(c)
		})
		request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(test.body))
		request.Header.Set("Content-Type", "application/json")
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("JSON match action returned %d", response.StatusCode)
		}
		_ = response.Body.Close()
	}
	if repo.reaction != domainuser.MatchReactionLike || repo.limit != 3 {
		t.Fatalf("JSON input not used: reaction=%s limit=%d", repo.reaction, repo.limit)
	}
}
