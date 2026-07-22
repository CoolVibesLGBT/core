package handlers

import (
	"bytes"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/constants"
	"core/models"
	"core/models/media"
	modelutils "core/models/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

type publicProjectionHandlerRepository struct {
	ports.UserRepository
	user  *models.User
	users []models.User
}

func (r *publicProjectionHandlerRepository) GetByUserNameOrEmailOrUsername(_ string) (*models.User, error) {
	return r.user, nil
}

func (r *publicProjectionHandlerRepository) GetUsersStartingWith(_ string, _ int) ([]models.User, error) {
	return r.users, nil
}

func TestFetchPublicProfileDoesNotSerializePersistenceSecrets(t *testing.T) {
	repository := &publicProjectionHandlerRepository{user: sensitivePublicProjectionUser()}
	service := usecases.NewUserService(repository, nil, nil, nil, nil)
	app := fiber.New()
	app.Get("/", HandleFetchUserProfile(service))

	request := httptest.NewRequest(http.MethodGet, "/?username=public-user", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	user := responseUser(t, response)
	assertPublicUserJSON(t, user)
	if user["website"] != "https://example.test/profile" {
		t.Fatalf("profile response lost public website: %#v", user["website"])
	}
	if _, ok := user["cover"]; !ok {
		t.Fatalf("profile response lost public cover: %#v", user)
	}
	engagements, ok := user["engagements"].(map[string]any)
	if !ok {
		t.Fatalf("profile response lost public aggregate counts: %#v", user["engagements"])
	}
	counts, _ := engagements["counts"].(map[string]any)
	if counts["follower_count"] != float64(12) {
		t.Fatalf("follower_count = %#v, want 12", counts["follower_count"])
	}
	for _, forbidden := range []string{"report_count", "deposit_amount", "withdraw_amount"} {
		if _, exists := counts[forbidden]; exists {
			t.Fatalf("private engagement counter %q leaked: %#v", forbidden, counts)
		}
	}
}

func TestPublicUserLookupAndGlobalSearchReturnSafeSummaries(t *testing.T) {
	user := sensitivePublicProjectionUser()
	repository := &publicProjectionHandlerRepository{users: []models.User{*user}}
	service := usecases.NewUserService(repository, nil, nil, nil, nil)

	tests := []struct {
		name    string
		handler fiber.Handler
		body    string
	}{
		{name: "lookup", handler: HandleGetUsersStartingWith(service), body: "query=public"},
		{name: "global", handler: HandleGlobalSearch(service, nil), body: "query=public&scope=people&limit=5"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Post("/", test.handler)
			request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(test.body))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("app.Test() error = %v", err)
			}
			if response.StatusCode != fiber.StatusOK {
				t.Fatalf("expected status 200, got %d", response.StatusCode)
			}

			var envelope struct {
				Data map[string]any `json:"data"`
			}
			if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			users, ok := envelope.Data["users"].([]any)
			if !ok || len(users) != 1 {
				t.Fatalf("users = %#v, want one result", envelope.Data["users"])
			}
			result, ok := users[0].(map[string]any)
			if !ok {
				t.Fatalf("user result = %#v", users[0])
			}
			assertPublicUserJSON(t, result)
			for _, profileOnly := range []string{"cover", "engagements", "date_of_birth", "privacy_level"} {
				if _, exists := result[profileOnly]; exists {
					t.Fatalf("search summary contains profile-only key %q: %#v", profileOnly, result)
				}
			}
		})
	}
}

func TestPublicUserFallbackFiltersPrivateBannedPendingAndBotAccounts(t *testing.T) {
	visible := sensitivePublicProjectionUser()
	privateUser := *sensitivePublicProjectionUser()
	privateUser.PublicID++
	privateUser.PrivacyLevel = constants.PrivacyPrivate
	bannedUser := *sensitivePublicProjectionUser()
	bannedUser.PublicID += 2
	bannedUser.UserRole = constants.UserRoleBanned
	pendingUser := *sensitivePublicProjectionUser()
	pendingUser.PublicID += 3
	pendingUser.UserRole = constants.UserRolePending
	botUser := *sensitivePublicProjectionUser()
	botUser.PublicID += 4
	botUser.IsBot = true

	repository := &publicProjectionHandlerRepository{
		users: []models.User{*visible, privateUser, bannedUser, pendingUser, botUser},
	}
	service := usecases.NewUserService(repository, nil, nil, nil, nil)
	app := fiber.New()
	app.Post("/", HandleGetUsersStartingWith(service))
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("query=public"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("app.Test() error = %v", err)
	}

	var envelope struct {
		Data struct {
			Users []map[string]any `json:"users"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(envelope.Data.Users) != 1 {
		t.Fatalf("public search users = %#v, want only visible account", envelope.Data.Users)
	}

	repository.user = &privateUser
	profileApp := fiber.New()
	profileApp.Get("/", HandleFetchUserProfile(service))
	profileResponse, err := profileApp.Test(httptest.NewRequest(http.MethodGet, "/?username=private-user", nil))
	if err != nil {
		t.Fatalf("profile app.Test() error = %v", err)
	}
	if profileResponse.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("private public-profile status = %d, want 400", profileResponse.StatusCode)
	}
}

func sensitivePublicProjectionUser() *models.User {
	avatarURL := "https://cdn.example/avatar.jpg"
	coverURL := "https://cdn.example/cover.jpg"
	variants := &modelutils.FileVariants{Image: &modelutils.ImageVariants{
		Small: &modelutils.VariantInfo{URL: "https://cdn.example/avatar-small.jpg"},
	}}
	avatar := &media.Media{
		ID:       uuid.New(),
		PublicID: 701,
		OwnerID:  uuid.New(),
		File: modelutils.FileMetadata{
			ID:          uuid.New(),
			URL:         avatarURL,
			StoragePath: "private/internal/avatar.jpg",
			Name:        "original-secret-name.jpg",
			Variants:    variants,
		},
	}
	cover := &media.Media{
		ID:       uuid.New(),
		PublicID: 702,
		File: modelutils.FileMetadata{
			ID:          uuid.New(),
			URL:         coverURL,
			StoragePath: "private/internal/cover.jpg",
			Name:        "cover-secret-name.jpg",
			Variants:    variants,
		},
	}
	bio := modelutils.LocalizedString{"en": "Public bio"}
	city := "Istanbul"
	display := "Istanbul, Türkiye"
	return &models.User{
		ID:               uuid.New(),
		PublicID:         9_007_199_254_740_993,
		UserName:         "public-user",
		DisplayName:      "Public User",
		Website:          "https://example.test/profile",
		Email:            "private@example.test",
		Password:         "password-hash",
		Bio:              &bio,
		Balance:          decimal.NewFromInt(999),
		PrivacyLevel:     constants.PrivacyPublic,
		PreferencesFlags: "private-bits",
		UserRole:         constants.UserRoleAdmin,
		IsOnline:         true,
		CreatedAt:        time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
		Location:         &modelutils.Location{City: &city, Display: &display},
		AvatarID:         &avatar.ID,
		Avatar:           avatar,
		CoverID:          &cover.ID,
		Cover:            cover,
		BroadcastInfo:    datatypes.JSON(`{"private":"broadcast-token"}`),
		Engagements: &models.Engagement{Counts: datatypes.JSON(
			`{"follower_count":12,"following_count":3,"report_count":8,"deposit_amount":"500"}`,
		)},
	}
}

func responseUser(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	var envelope struct {
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	user, ok := envelope.Data["user"].(map[string]any)
	if !ok {
		t.Fatalf("user payload = %#v", envelope.Data["user"])
	}
	return user
}

func assertPublicUserJSON(t *testing.T, user map[string]any) {
	t.Helper()
	const publicID = "9007199254740993"
	if user["id"] != publicID || user["public_id"] != publicID {
		t.Fatalf("public IDs = (%#v, %#v), want %q", user["id"], user["public_id"], publicID)
	}
	for _, forbidden := range []string{
		"balance", "user_role", "preferences_flags", "broadcast_info", "subscriptions",
		"socket_id", "avatar_id", "cover_id", "wallet", "deleted_at", "updated_at",
	} {
		if _, exists := user[forbidden]; exists {
			t.Fatalf("private user key %q leaked: %#v", forbidden, user)
		}
	}

	avatar, ok := user["avatar"].(map[string]any)
	if !ok {
		t.Fatalf("avatar = %#v", user["avatar"])
	}
	if len(avatar) != 1 {
		t.Fatalf("avatar contains non-file metadata: %#v", avatar)
	}
	file, ok := avatar["file"].(map[string]any)
	if !ok || file["url"] != "https://cdn.example/avatar.jpg" {
		t.Fatalf("avatar file = %#v", avatar["file"])
	}
	for _, forbidden := range []string{"id", "storage_path", "name", "size", "created_at", "mime_type"} {
		if _, exists := file[forbidden]; exists {
			t.Fatalf("private avatar file key %q leaked: %#v", forbidden, file)
		}
	}
}
