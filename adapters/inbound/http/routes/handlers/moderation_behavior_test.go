package handlers

import (
	"bytes"
	"context"
	"core/application/ports"
	"core/application/usecases"
	"core/constants"
	domainmoderation "core/domain/moderation"
	"core/models"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type moderationHandlerRepository struct {
	fetchFilter ports.ModerationReportFilter
	resolve     ports.ModerationResolveInput
}

func (r *moderationHandlerRepository) FetchReports(_ context.Context, filter ports.ModerationReportFilter) (ports.ModerationReportPage, error) {
	r.fetchFilter = filter
	return ports.ModerationReportPage{Limit: filter.Limit}, nil
}

func (r *moderationHandlerRepository) ResolveReport(_ context.Context, input ports.ModerationResolveInput) (*ports.ModerationReportView, error) {
	r.resolve = input
	return &ports.ModerationReportView{
		ID:     input.ReportID,
		Status: input.Status,
	}, nil
}

func (r *moderationHandlerRepository) SetPostPublished(_ context.Context, postPublicID int64, published bool, _ uuid.UUID, _ string) (ports.ModerationPostView, error) {
	return ports.ModerationPostView{
		"public_id": strconv.FormatInt(postPublicID, 10),
		"published": published,
	}, nil
}

func TestModerationFetchAcceptsJSONAndDefaultsPending(t *testing.T) {
	repo := &moderationHandlerRepository{}
	service := usecases.NewModerationService(repo)
	moderator := &models.User{ID: uuid.New(), UserRole: constants.UserRoleModerator}

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", moderator)
		return HandleModerationFetchReports(service)(c)
	})
	body, _ := json.Marshal(map[string]any{"content_type": "user", "user_id": 88, "limit": 7})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.fetchFilter.Status != domainmoderation.StatusPending || repo.fetchFilter.ContentableType != domainmoderation.TargetUser || repo.fetchFilter.UserPublicID != 88 || repo.fetchFilter.Limit != 7 {
		t.Fatalf("filter = %#v", repo.fetchFilter)
	}
}

func TestModerationResolveAcceptsJSONBoolean(t *testing.T) {
	repo := &moderationHandlerRepository{}
	service := usecases.NewModerationService(repo)
	moderator := &models.User{ID: uuid.New(), UserRole: constants.UserRoleAdmin}
	reportID := uuid.New()

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", moderator)
		return HandleModerationResolveReport(service)(c)
	})
	body, _ := json.Marshal(map[string]any{
		"report_id":    reportID.String(),
		"status":       domainmoderation.StatusActioned,
		"publish_post": false,
		"resolution":   "hidden",
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if repo.resolve.ReportID != reportID || repo.resolve.Status != domainmoderation.StatusActioned || repo.resolve.PublishPost == nil || *repo.resolve.PublishPost || repo.resolve.ReviewedByID != moderator.ID {
		t.Fatalf("resolve input = %#v", repo.resolve)
	}
}
