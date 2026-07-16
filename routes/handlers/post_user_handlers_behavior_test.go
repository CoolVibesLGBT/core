package handlers

import (
	"bytes"
	"context"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/models"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"core/models/taxonomy"
	"core/models/utils"
	"core/types"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type handlerPostRepo struct {
	ports.PostRepository
	createContentableType string
	createFileCount       int
	createdID             uuid.UUID
	likeFilter            types.Filter
	storiesFilter         types.Filter
	eventRSVPPostID       int64
	eventRSVPUserID       uuid.UUID
	eventRSVPStatus       *postpayloads.EventAttendanceStatus
}

func (r *handlerPostRepo) CreateContentablePost(ctx context.Context, form ports.FormData, author *models.User, contentableType string, contentableID *uuid.UUID) (*post.Post, error) {
	r.createContentableType = contentableType
	r.createFileCount = len(form.Files)
	r.createdID = uuid.New()
	return &post.Post{ID: r.createdID, PublicID: 1001, PostKind: post.PostKind(contentableType), AuthorID: author.ID}, nil
}

func (r *handlerPostRepo) GetPostByID(id uuid.UUID) (*post.Post, error) {
	return &post.Post{ID: id, PublicID: 1001, PostKind: post.PostKind(r.createContentableType)}, nil
}

func (r *handlerPostRepo) Like(filters types.Filter) error {
	r.likeFilter = filters
	return nil
}

func (r *handlerPostRepo) SetEventRSVP(ctx context.Context, postPublicID int64, userID uuid.UUID, status *postpayloads.EventAttendanceStatus) (*postpayloads.EventRSVPResult, error) {
	r.eventRSVPPostID = postPublicID
	r.eventRSVPUserID = userID
	r.eventRSVPStatus = status
	return &postpayloads.EventRSVPResult{
		Status: status,
		Counts: postpayloads.EventAttendanceCounts{Going: 1},
	}, nil
}

func (r *handlerPostRepo) GetPostsByKind(filters types.Filter) (types.PostsResult, error) {
	r.storiesFilter = filters
	cursor, _ := types.NewPublicIDCursor(1001)
	return types.PostsResult{
		Posts:  []post.Post{{ID: uuid.New(), PublicID: 1001, PostKind: filters.PostKind}},
		Cursor: cursor,
	}, nil
}

type handlerUserRepo struct {
	ports.UserRepository
	byPublicID    map[int64]*models.User
	deletedFilter types.Filter
}

func (r *handlerUserRepo) GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (r *handlerUserRepo) GetUserByPublicIdWithoutRelations(filters types.Filter) (*models.User, error) {
	if user, ok := r.byPublicID[filters.UserID]; ok {
		return user, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *handlerUserRepo) UpdateLocation(ctx context.Context, user *models.User, ip string) error {
	return nil
}

func (r *handlerUserRepo) DeleteUser(filters types.Filter) error {
	r.deletedFilter = filters
	return nil
}

func (r *handlerUserRepo) GetPreferences() (*models.PreferencesData, error) {
	return &models.PreferencesData{}, nil
}

type handlerMediaRepo struct {
	ports.MediaRepository
}

func (r *handlerMediaRepo) AddMedia(ownerID uuid.UUID, ownerType media.OwnerType, userID uuid.UUID, role media.MediaRole, file ports.UploadedFile) (*media.Media, error) {
	return &media.Media{ID: uuid.New(), OwnerID: ownerID, OwnerType: ownerType, UserID: userID, Role: role}, nil
}

type handlerEngagementRepo struct {
	ports.EngagementRepository
	toggles      []models.EngagementKind
	recordedView []models.EngagementKind
	getCalls     int
}

func (r *handlerEngagementRepo) ToggleEngagement(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) (bool, error) {
	r.toggles = append(r.toggles, kind)
	return true, nil
}

func (r *handlerEngagementRepo) HasUserEngaged(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind) (bool, error) {
	return true, nil
}

func (r *handlerEngagementRepo) RecordViewOnce(ctx context.Context, engagerID uuid.UUID, engageeID uuid.UUID, kind models.EngagementKind, contentableID uuid.UUID, contentableType models.EngagementContentableType) (bool, error) {
	r.recordedView = append(r.recordedView, kind)
	return true, nil
}

func (r *handlerEngagementRepo) GetEngagements(ctx context.Context, contentableType models.EngagementContentableType, contentableID uuid.UUID, engagementKind models.EngagementKind, cursor *time.Time, limit int) ([]models.EngagementDetail, *time.Time, error) {
	r.getCalls++
	return nil, nil, nil
}

type handlerNotificationRepo struct {
	ports.NotificationRepository
}

func (r *handlerNotificationRepo) SendNotificationToUser(sender models.User, receiver models.User, notificationType string, notificationTitle string, notificationMessage string, payload notifications.NotificationPayload) error {
	return nil
}

func (r *handlerNotificationRepo) FetchAndMarkShownNotifications(userID uuid.UUID, limit int) ([]notifications.Notification, error) {
	return nil, nil
}

func TestHandleCreateUsesRequestedPostKind(t *testing.T) {
	postRepo := &handlerPostRepo{}
	service := usecases.NewPostService(&handlerUserRepo{}, postRepo, &handlerMediaRepo{})
	authUser := &models.User{ID: uuid.New(), PublicID: 10, DefaultLanguage: "en", Domain: models.CoolVibes}

	resp := performMultipartHandlerRequest(t, HandleCreate(service), authUser, map[string]string{
		"kind":    string(post.PostKindVideo),
		"content": "hello",
	}, nil)

	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}
	if postRepo.createContentableType != string(post.PostKindVideo) {
		t.Fatalf("expected video kind, got %q", postRepo.createContentableType)
	}
}

func TestHandlePostLikeParsesFilterAndAuthUser(t *testing.T) {
	postRepo := &handlerPostRepo{}
	service := usecases.NewPostService(&handlerUserRepo{}, postRepo, &handlerMediaRepo{})
	authUser := &models.User{ID: uuid.New(), PublicID: 10}

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", authUser)
		return HandlePostLike(service)(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("post_id=123"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if postRepo.likeFilter.PostID != 123 {
		t.Fatalf("expected parsed post id 123, got %#v", postRepo.likeFilter)
	}
	if postRepo.likeFilter.AuthUser == nil || postRepo.likeFilter.AuthUser.ID != authUser.ID {
		t.Fatalf("expected auth user in filter, got %#v", postRepo.likeFilter.AuthUser)
	}
}

func TestHandlePostEventRSVPParsesCanonicalStatusAndAuthUser(t *testing.T) {
	postRepo := &handlerPostRepo{}
	service := usecases.NewPostService(&handlerUserRepo{}, postRepo, &handlerMediaRepo{})
	authUser := &models.User{ID: uuid.New(), PublicID: 10}

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", authUser)
		return HandlePostEventRSVP(service)(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("post_id=123&status=going"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if postRepo.eventRSVPPostID != 123 || postRepo.eventRSVPUserID != authUser.ID || postRepo.eventRSVPStatus == nil || *postRepo.eventRSVPStatus != postpayloads.EventAttendanceGoing {
		t.Fatalf("unexpected RSVP repository args: %#v", postRepo)
	}
}

func TestHandlePostEventRSVPRejectsNonCanonicalStatus(t *testing.T) {
	postRepo := &handlerPostRepo{}
	service := usecases.NewPostService(&handlerUserRepo{}, postRepo, &handlerMediaRepo{})
	authUser := &models.User{ID: uuid.New(), PublicID: 10}

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", authUser)
		return HandlePostEventRSVP(service)(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("post_id=123&status=interested"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	if postRepo.eventRSVPPostID != 0 {
		t.Fatal("invalid status reached repository")
	}
}

func TestHandleFetchUserViewEngagementsRejectsNonOwner(t *testing.T) {
	owner := &models.User{ID: uuid.New(), PublicID: 20}
	visitor := &models.User{ID: uuid.New(), PublicID: 10}
	engagementRepo := &handlerEngagementRepo{}
	service := usecases.NewUserService(
		&handlerUserRepo{byPublicID: map[int64]*models.User{owner.PublicID: owner}},
		&handlerPostRepo{},
		&handlerMediaRepo{},
		engagementRepo,
		&handlerNotificationRepo{},
	)

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", visitor)
		return HandleFetchUserEngagements(service)(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("user_id=20&engagement_type=view_received"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected status 403, got %d", resp.StatusCode)
	}
	if engagementRepo.getCalls != 0 {
		t.Fatalf("private view list reached repository %d times", engagementRepo.getCalls)
	}
}

func TestHandleUserViewProfileReturnsCounted(t *testing.T) {
	viewer := &models.User{ID: uuid.New(), PublicID: 10}
	target := &models.User{ID: uuid.New(), PublicID: 20}
	engagementRepo := &handlerEngagementRepo{}
	service := usecases.NewUserService(
		&handlerUserRepo{byPublicID: map[int64]*models.User{target.PublicID: target}},
		&handlerPostRepo{},
		&handlerMediaRepo{},
		engagementRepo,
		&handlerNotificationRepo{},
	)

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", viewer)
		return HandleUserViewProfile(service)(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("public_id=20"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	var body struct {
		Data struct {
			Counted bool `json:"counted"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !body.Data.Counted || len(engagementRepo.recordedView) != 1 || engagementRepo.recordedView[0] != models.EngagementKindViewReceived {
		t.Fatalf("unexpected profile view result: body=%#v records=%#v", body, engagementRepo.recordedView)
	}
}

func TestHandleUserLikeTogglesUserLike(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	targetUser := &models.User{ID: uuid.New(), PublicID: 20}
	userRepo := &handlerUserRepo{byPublicID: map[int64]*models.User{
		authUser.PublicID:   authUser,
		targetUser.PublicID: targetUser,
	}}
	engagementRepo := &handlerEngagementRepo{}
	service := usecases.NewUserService(userRepo, &handlerPostRepo{}, &handlerMediaRepo{}, engagementRepo, &handlerNotificationRepo{})

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", authUser)
		return HandleUserLike(service)(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("likee_id=20"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if len(engagementRepo.toggles) != 2 {
		t.Fatalf("expected two engagement toggles, got %d", len(engagementRepo.toggles))
	}
	if engagementRepo.toggles[0] != models.EngagementKindLikeGiven || engagementRepo.toggles[1] != models.EngagementKindLikeReceived {
		t.Fatalf("expected like given/received toggles, got %#v", engagementRepo.toggles)
	}
}

func TestHandleUploadStoryCreatesStoryPostWithFile(t *testing.T) {
	postRepo := &handlerPostRepo{}
	service := usecases.NewUserService(&handlerUserRepo{}, postRepo, &handlerMediaRepo{}, &handlerEngagementRepo{}, &handlerNotificationRepo{})
	authUser := &models.User{ID: uuid.New(), PublicID: 10, DefaultLanguage: "en", Domain: models.CoolVibes}

	resp := performMultipartHandlerRequest(t, HandleUploadStory(service), authUser, map[string]string{
		"caption": "story caption",
	}, map[string][]byte{
		"story": []byte("fake-image"),
	})

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if postRepo.createContentableType != string(post.PostKindStory) {
		t.Fatalf("expected story contentable type, got %q", postRepo.createContentableType)
	}
	if postRepo.createFileCount != 1 {
		t.Fatalf("expected one uploaded file, got %d", postRepo.createFileCount)
	}
}

func TestHandleFetchStoriesReturnsPostsAndCursor(t *testing.T) {
	postRepo := &handlerPostRepo{}
	service := usecases.NewUserService(&handlerUserRepo{}, postRepo, &handlerMediaRepo{}, &handlerEngagementRepo{}, &handlerNotificationRepo{})

	app := fiber.New()
	app.Post("/", HandleFetchStories(service))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("limit=5"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if postRepo.storiesFilter.PostKind != post.PostKindStory {
		t.Fatalf("expected story filter, got %q", postRepo.storiesFilter.PostKind)
	}

	var body struct {
		Data struct {
			Stories []struct {
				PostKind post.PostKind `json:"post_kind"`
			} `json:"stories"`
			Cursor *string `json:"cursor"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data.Stories) != 1 || body.Data.Stories[0].PostKind != post.PostKindStory {
		t.Fatalf("expected story response, got %#v", body.Data.Stories)
	}
	if body.Data.Cursor == nil {
		t.Fatalf("expected cursor in response")
	}
}

func TestHandleUserDeleteUsesAuthenticatedUserID(t *testing.T) {
	authUser := &models.User{ID: uuid.New(), PublicID: 10}
	submittedID := uuid.New()
	userRepo := &handlerUserRepo{}
	service := usecases.NewUserService(userRepo, &handlerPostRepo{}, &handlerMediaRepo{}, &handlerEngagementRepo{}, &handlerNotificationRepo{})

	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		c.Locals("authenticatedUser", authUser)
		return HandleUserDelete(service)(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("user_id="+submittedID.String()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if userRepo.deletedFilter.UserUUID != authUser.ID {
		t.Fatalf("expected delete user uuid %s, got %s", authUser.ID, userRepo.deletedFilter.UserUUID)
	}
	if userRepo.deletedFilter.AuthUser == nil || userRepo.deletedFilter.AuthUser.ID != authUser.ID {
		t.Fatalf("expected authenticated user in filter, got %#v", userRepo.deletedFilter.AuthUser)
	}
}

func performMultipartHandlerRequest(t *testing.T, handler fiber.Handler, authUser *models.User, fields map[string]string, files map[string][]byte) *http.Response {
	t.Helper()
	app := fiber.New()
	app.Post("/", func(c fiber.Ctx) error {
		if authUser != nil {
			c.Locals("authenticatedUser", authUser)
		}
		return handler(c)
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	for key, content := range files {
		part, err := writer.CreateFormFile(key, key+".jpg")
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	return resp
}

var _ ports.PostRepository = (*handlerPostRepo)(nil)
var _ ports.UserRepository = (*handlerUserRepo)(nil)
var _ ports.MediaRepository = (*handlerMediaRepo)(nil)
var _ ports.EngagementRepository = (*handlerEngagementRepo)(nil)
var _ ports.NotificationRepository = (*handlerNotificationRepo)(nil)

// Keep imported repository model packages referenced by embedded interfaces in older Go toolchains.
var _ = taxonomy.Pillar{}
var _ = decimal.Decimal{}
var _ = utils.Location{}
