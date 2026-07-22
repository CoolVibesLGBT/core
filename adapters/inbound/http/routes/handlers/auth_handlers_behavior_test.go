package handlers

import (
	"context"
	"core/application/ports"
	"core/application/types"
	usecases "core/application/usecases"
	"core/constants"
	"core/models"
	"core/models/media"
	modelutils "core/models/utils"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

type authHandlerUserRepo struct {
	ports.UserRepository
	created     *models.User
	byID        map[uuid.UUID]*models.User
	byUsername  *models.User
	getByIDErr  error
	requestedID uuid.UUID
}

func (r *authHandlerUserRepo) ExistsByNameOrMail(input string) (bool, error) {
	return false, nil
}

func (r *authHandlerUserRepo) ExistsByUsername(username string) (bool, error) {
	return false, nil
}

func (r *authHandlerUserRepo) ExistsByEmail(email string) (bool, error) {
	return false, nil
}

func (r *authHandlerUserRepo) Create(user *models.User) error {
	r.created = user
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*models.User)
	}
	r.byID[user.ID] = user
	return nil
}

func (r *authHandlerUserRepo) GetByID(userID uuid.UUID) (*models.User, error) {
	r.requestedID = userID
	if r.getByIDErr != nil {
		return nil, r.getByIDErr
	}
	if user, ok := r.byID[userID]; ok {
		return user, nil
	}
	if r.created != nil && r.created.ID == userID {
		return r.created, nil
	}
	return nil, errors.New("user not found")
}

func (r *authHandlerUserRepo) GetByUserNameOrEmailOrUsername(input string) (*models.User, error) {
	if r.byUsername != nil {
		return r.byUsername, nil
	}
	return nil, errors.New("user not found")
}

func (r *authHandlerUserRepo) GetUserUUIDByPublicID(publicID int64) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (r *authHandlerUserRepo) GetPreferences() (*models.PreferencesData, error) {
	return &models.PreferencesData{}, nil
}

func (r *authHandlerUserRepo) GetUserByPublicIdWithoutRelations(filters types.Filter) (*models.User, error) {
	return &models.User{ID: uuid.New(), PublicID: filters.UserID}, nil
}

type authHandlerCaptcha struct {
	token string
}

func (v *authHandlerCaptcha) VerifyCaptcha(ctx context.Context, response string) (bool, error) {
	v.token = response
	return true, nil
}

type authHandlerHasher struct {
	compareOK bool
}

func (h *authHandlerHasher) HashPassword(raw string) (string, error) {
	return "hashed:" + raw, nil
}

func (h *authHandlerHasher) ComparePassword(hashed string, raw string) (bool, error) {
	return h.compareOK, nil
}

type authHandlerTokenIssuer struct {
	token string
}

func (i authHandlerTokenIssuer) GenerateUserToken(userID uuid.UUID, publicID int64) (string, error) {
	return i.token, nil
}

type authHandlerPublicIDGenerator struct {
	next int64
}

func (g authHandlerPublicIDGenerator) GeneratePublicID() int64 {
	return g.next
}

func TestHandleRegisterParsesMultipartAndReturnsToken(t *testing.T) {
	userRepo := &authHandlerUserRepo{}
	captcha := &authHandlerCaptcha{}
	service := usecases.NewUserService(
		userRepo,
		&handlerPostRepo{},
		&handlerMediaRepo{},
		&handlerEngagementRepo{},
		&handlerNotificationRepo{},
		usecases.WithCaptchaVerifier(captcha),
		usecases.WithPasswordHasher(&authHandlerHasher{}),
		usecases.WithTokenIssuer(authHandlerTokenIssuer{token: "register-token"}),
		usecases.WithPublicIDGenerator(authHandlerPublicIDGenerator{next: 777}),
	)

	resp := performMultipartHandlerRequest(t, HandleRegister(service), nil, map[string]string{
		"name":           "alice",
		"nickname":       "Alice",
		"password":       "secret-123",
		"domain":         string(models.CoolVibes),
		"email":          "alice@example.com",
		"recaptchaToken": "captcha-token",
	}, nil)

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if userRepo.created == nil || userRepo.created.PublicID != 777 || userRepo.created.Password != "hashed:secret-123" {
		t.Fatalf("expected created registered user, got %#v", userRepo.created)
	}
	if captcha.token != "captcha-token" {
		t.Fatalf("expected recaptcha token to pass through, got %q", captcha.token)
	}

	var body struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID       string `json:"id"`
				PublicID string `json:"public_id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Token != "register-token" || body.Data.User.ID != "777" || body.Data.User.PublicID != "777" {
		t.Fatalf("expected token/public id in response, got %#v", body.Data)
	}
}

func TestHandleLoginParsesCredentialsAndReturnsToken(t *testing.T) {
	userID := uuid.New()
	userRepo := &authHandlerUserRepo{
		byUsername: &models.User{
			ID: userID, PublicID: 888, UserName: "alice", DisplayName: "Alice",
			Email: "alice-private@example.test", Password: "stored-hash",
			Balance: decimal.RequireFromString("12345.67"), PreferencesFlags: "private-preference-bits",
			UserRole: constants.UserRoleAdmin, BroadcastInfo: datatypes.JSON(`{"token":"broadcast-secret"}`),
			Subscriptions: datatypes.JSON(`{"endpoint":"subscription-secret"}`),
		},
	}
	service := usecases.NewUserService(
		userRepo,
		&handlerPostRepo{},
		&handlerMediaRepo{},
		&handlerEngagementRepo{},
		&handlerNotificationRepo{},
		usecases.WithPasswordHasher(&authHandlerHasher{compareOK: true}),
		usecases.WithTokenIssuer(authHandlerTokenIssuer{token: "login-token"}),
	)

	resp := performMultipartHandlerRequest(t, HandleLogin(service), nil, map[string]string{
		"nickname": "alice",
		"password": "secret",
	}, nil)

	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	encoded, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Data struct {
			Token string `json:"token"`
			User  struct {
				ID       string `json:"id"`
				PublicID string `json:"public_id"`
				UserRole string `json:"user_role"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Token != "login-token" || body.Data.User.ID != "888" || body.Data.User.PublicID != "888" || body.Data.User.UserRole != string(constants.UserRoleAdmin) {
		t.Fatalf("expected safe login token/user projection, got %#v", body.Data)
	}
	assertAuthJSONDoesNotLeak(t, encoded, userID.String(), "alice-private@example.test", "stored-hash", "12345.67", "private-preference-bits", "broadcast-secret", "subscription-secret")
	if strings.Contains(string(encoded), `"user_role"`) == false {
		t.Fatalf("authenticated UI role should remain allowlisted: %s", encoded)
	}
}

func TestHandleAuthCheckReloadsAndReturnsFullProfile(t *testing.T) {
	userID := uuid.New()
	coverID := uuid.New()
	coverFileID := uuid.New()
	engagementID := uuid.New()
	fullUser := &models.User{
		ID: userID, PublicID: 999, UserName: "full-profile", DisplayName: "Full Profile",
		Email: "auth-check-private@example.test", Password: "auth-check-password",
		Balance: decimal.RequireFromString("7654.32"), PreferencesFlags: "auth-check-preferences",
		UserRole: constants.UserRoleAdmin, SocketID: stringPointerForAuthTest("socket-secret"),
		BroadcastInfo: datatypes.JSON(`{"private":"broadcast-secret"}`),
		Subscriptions: datatypes.JSON(`{"private":"subscription-secret"}`),
		Cover: &media.Media{ID: coverID, PublicID: 777001, FileID: coverFileID, OwnerID: userID, UserID: userID, File: modelutils.FileMetadata{
			ID: coverFileID, URL: "https://cdn.example.test/safe-cover.jpg", StoragePath: "/private/storage/cover.jpg",
		}},
		Engagements: &models.Engagement{ID: engagementID},
	}
	userRepo := &authHandlerUserRepo{byID: map[uuid.UUID]*models.User{userID: fullUser}}
	service := usecases.NewUserService(
		userRepo,
		&handlerPostRepo{},
		&handlerMediaRepo{},
		&handlerEngagementRepo{},
		&handlerNotificationRepo{},
	)

	// Middleware now provides only this narrow projection. The endpoint must
	// explicitly reload the full profile rather than echoing it.
	sessionUser := &models.User{ID: userID, PublicID: 999, UserName: "session-only"}
	resp := performMultipartHandlerRequest(t, HandleAuthCheck(service), sessionUser, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if userRepo.requestedID != userID {
		t.Fatalf("GetUserByID() id = %s, want %s", userRepo.requestedID, userID)
	}

	encoded, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Data struct {
			ID       string `json:"id"`
			PublicID string `json:"public_id"`
			UserName string `json:"username"`
			UserRole string `json:"user_role"`
			Cover    *struct {
				File struct {
					URL string `json:"url"`
				} `json:"file"`
			} `json:"cover"`
		} `json:"data"`
	}
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.ID != "999" || body.Data.PublicID != "999" || body.Data.UserName != fullUser.UserName ||
		body.Data.UserRole != string(constants.UserRoleAdmin) || body.Data.Cover == nil || body.Data.Cover.File.URL != "https://cdn.example.test/safe-cover.jpg" {
		t.Fatalf("auth check did not return safe profile: %#v", body.Data)
	}
	assertAuthJSONDoesNotLeak(t, encoded,
		userID.String(), coverID.String(), coverFileID.String(), engagementID.String(), "/private/storage/cover.jpg",
		"auth-check-private@example.test", "auth-check-password", "7654.32", "auth-check-preferences",
		"socket-secret", "broadcast-secret", "subscription-secret",
	)
}

func assertAuthJSONDoesNotLeak(t *testing.T, encoded []byte, forbiddenValues ...string) {
	t.Helper()
	payload := string(encoded)
	for _, forbidden := range append(forbiddenValues,
		`"email"`, `"password"`, `"balance"`, `"preferences_flags"`, `"socket_id"`,
		`"broadcast_info"`, `"subscriptions"`, `"storage_path"`, `"file_id"`, `"owner_id"`,
	) {
		if forbidden != "" && strings.Contains(payload, forbidden) {
			t.Fatalf("authenticated user response leaked %q: %s", forbidden, payload)
		}
	}
}

func stringPointerForAuthTest(value string) *string { return &value }

func TestHandleAuthCheckRejectsStaleSessionWhenFullProfileLoadFails(t *testing.T) {
	userID := uuid.New()
	userRepo := &authHandlerUserRepo{getByIDErr: errors.New("database unavailable")}
	service := usecases.NewUserService(
		userRepo,
		&handlerPostRepo{},
		&handlerMediaRepo{},
		&handlerEngagementRepo{},
		&handlerNotificationRepo{},
	)

	resp := performMultipartHandlerRequest(t, HandleAuthCheck(service), &models.User{ID: userID}, nil, nil)
	if resp.StatusCode != 401 {
		t.Fatalf("expected status 401, got %d", resp.StatusCode)
	}
	if userRepo.requestedID != userID {
		t.Fatalf("GetUserByID() id = %s, want %s", userRepo.requestedID, userID)
	}
}

func TestHandleUserInfoUsesSameSafeAuthProjection(t *testing.T) {
	internalID := uuid.New()
	user := &models.User{
		ID: internalID, PublicID: 1234567890123, UserName: "session-user", DisplayName: "Session User",
		Email: "userinfo-private@example.test", Password: "userinfo-password",
		Balance: decimal.RequireFromString("555.44"), PreferencesFlags: "userinfo-preferences",
		BroadcastInfo: datatypes.JSON(`{"token":"userinfo-broadcast"}`),
		Subscriptions: datatypes.JSON(`{"endpoint":"userinfo-subscription"}`),
	}
	repo := &authHandlerUserRepo{byID: map[uuid.UUID]*models.User{internalID: user}}
	service := usecases.NewUserService(repo, &handlerPostRepo{}, &handlerMediaRepo{}, &handlerEngagementRepo{}, &handlerNotificationRepo{})

	resp := performMultipartHandlerRequest(t, HandleUserInfo(service), &models.User{ID: internalID, PublicID: user.PublicID}, nil, nil)
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	encoded, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Data struct {
			User struct {
				ID       string `json:"id"`
				PublicID string `json:"public_id"`
				UserName string `json:"username"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(encoded, &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.User.ID != "1234567890123" || body.Data.User.PublicID != "1234567890123" || body.Data.User.UserName != "session-user" {
		t.Fatalf("unexpected auth user-info projection: %#v", body.Data.User)
	}
	assertAuthJSONDoesNotLeak(t, encoded, internalID.String(), "userinfo-private@example.test", "userinfo-password", "555.44", "userinfo-preferences", "userinfo-broadcast", "userinfo-subscription")
}

var _ ports.UserRepository = (*authHandlerUserRepo)(nil)
