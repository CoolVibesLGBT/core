package handlers

import (
	"context"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/models"
	"core/types"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type authHandlerUserRepo struct {
	ports.UserRepository
	created    *models.User
	byID       map[uuid.UUID]*models.User
	byUsername *models.User
}

func (r *authHandlerUserRepo) ExistsByNameOrMail(input string) (bool, error) {
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

func (r *authHandlerUserRepo) UpdateLocation(ctx context.Context, user *models.User, ip string) error {
	return nil
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
				PublicID string `json:"public_id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Token != "register-token" || body.Data.User.PublicID != "777" {
		t.Fatalf("expected token/public id in response, got %#v", body.Data)
	}
}

func TestHandleLoginParsesCredentialsAndReturnsToken(t *testing.T) {
	userID := uuid.New()
	userRepo := &authHandlerUserRepo{
		byUsername: &models.User{ID: userID, PublicID: 888, UserName: "alice", Password: "stored-hash"},
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
	var body struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Token != "login-token" {
		t.Fatalf("expected login token, got %q", body.Data.Token)
	}
}

var _ ports.UserRepository = (*authHandlerUserRepo)(nil)
