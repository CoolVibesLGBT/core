package usecases

import (
	"context"
	"core/constants"
	"core/models"
	"testing"

	"github.com/google/uuid"
)

func TestRegisterUserCreatesUserAndReturnsIssuedToken(t *testing.T) {
	userRepo := &fakeUserRepository{byID: map[uuid.UUID]*models.User{}}
	captcha := &fakeCaptchaVerifier{valid: true}
	events := &fakeEventPublisher{}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithCaptchaVerifier(captcha),
		WithPasswordHasher(&fakePasswordHasher{}),
		WithTokenIssuer(&fakeTokenIssuer{token: "issued-token"}),
		WithPublicIDGenerator(&fakePublicIDGenerator{next: 4242}),
		WithEventPublisher(events),
	)

	user, token, err := service.RegisterUser(context.Background(), RegisterInput{
		Name:           "alice",
		Nickname:       "Alice",
		Password:       "secret-123",
		Domain:         string(models.CoolVibes),
		Email:          "alice@example.com",
		RecaptchaToken: "captcha-token",
	})
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	if token != "issued-token" {
		t.Fatalf("expected issued token, got %q", token)
	}
	if user == nil || user.PublicID != 4242 {
		t.Fatalf("expected created user public id 4242, got %#v", user)
	}
	if userRepo.created == nil {
		t.Fatalf("expected user to be persisted")
	}
	if userRepo.created.Password != "hashed:secret-123" {
		t.Fatalf("expected hashed password, got %q", userRepo.created.Password)
	}
	if captcha.token != "captcha-token" {
		t.Fatalf("expected recaptcha token to be used, got %q", captcha.token)
	}
	if len(events.events) != 1 {
		t.Fatalf("expected registration event, got %d", len(events.events))
	}
}

func TestRegisterUserRejectsInvalidCaptchaBeforeCreate(t *testing.T) {
	userRepo := &fakeUserRepository{}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithCaptchaVerifier(&fakeCaptchaVerifier{valid: false}),
	)

	_, _, err := service.RegisterUser(context.Background(), RegisterInput{
		Name: "alice", Nickname: "Alice", Password: "secret-123", Domain: string(models.CoolVibes), Email: "alice@example.com",
	})
	if err == nil {
		t.Fatalf("expected invalid captcha error")
	}
	if userRepo.created != nil {
		t.Fatalf("user should not be created when captcha fails")
	}
}

func TestRegisterUserAppliesNamedReferralCode(t *testing.T) {
	referrerID := uuid.New()
	userRepo := &fakeUserRepository{
		byID:         map[uuid.UUID]*models.User{},
		byNameOrMail: &models.User{ID: referrerID, PublicID: 99, UserName: "dilber"},
	}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithCaptchaVerifier(&fakeCaptchaVerifier{valid: true}),
		WithPasswordHasher(&fakePasswordHasher{}),
		WithTokenIssuer(&fakeTokenIssuer{token: "issued-token"}),
		WithPublicIDGenerator(&fakePublicIDGenerator{next: 4243}),
	)

	user, _, err := service.RegisterUser(context.Background(), RegisterInput{
		Name:           "bob",
		Nickname:       "Bob",
		Password:       "secret-123",
		Domain:         string(models.CoolVibes),
		Email:          "bob@example.com",
		RecaptchaToken: "captcha-token",
		Referral:       "dilber",
	})
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	if userRepo.referralReferrerID != referrerID {
		t.Fatalf("expected referrer %s, got %s", referrerID, userRepo.referralReferrerID)
	}
	if userRepo.referralReferredID != user.ID {
		t.Fatalf("expected referred user %s, got %s", user.ID, userRepo.referralReferredID)
	}
	if !userRepo.referralReward.Equal(constants.DEFAULT_REFERRAL_REWARD) {
		t.Fatalf("expected referral reward %s, got %s", constants.DEFAULT_REFERRAL_REWARD, userRepo.referralReward)
	}
}

func TestLoginUserVerifiesPasswordAndIssuesToken(t *testing.T) {
	userID := uuid.New()
	userRepo := &fakeUserRepository{
		byUsername: &models.User{ID: userID, PublicID: 55, UserName: "alice", Password: "stored-hash"},
	}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithPasswordHasher(&fakePasswordHasher{compareOK: true}),
		WithTokenIssuer(&fakeTokenIssuer{token: "login-token"}),
	)

	user, token, err := service.LoginUser(context.Background(), LoginInput{UserName: "alice", Password: "secret"})
	if err != nil {
		t.Fatalf("LoginUser() error = %v", err)
	}
	if user.ID != userID {
		t.Fatalf("expected logged in user %s, got %s", userID, user.ID)
	}
	if token != "login-token" {
		t.Fatalf("expected login token, got %q", token)
	}
}

func TestToggleFollowWritesBothSidesAndPublishesEvent(t *testing.T) {
	followerID := uuid.New()
	followeeID := uuid.New()
	userRepo := &fakeUserRepository{
		byPublicID: map[int64]*models.User{
			1: {ID: followerID, PublicID: 1, UserName: "follower"},
			2: {ID: followeeID, PublicID: 2, UserName: "followee"},
		},
	}
	engagementRepo := &fakeEngagementRepository{has: map[models.EngagementKind]bool{
		models.EngagementKindFollowing: true,
	}}
	events := &fakeEventPublisher{}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		engagementRepo,
		&fakeNotificationRepository{},
		WithEventPublisher(events),
	)

	status, err := service.ToggleFollow(context.Background(), 1, 2)
	if err != nil {
		t.Fatalf("ToggleFollow() error = %v", err)
	}
	if !status {
		t.Fatalf("expected follow status true")
	}
	if len(engagementRepo.toggles) != 2 {
		t.Fatalf("expected two engagement toggles, got %d", len(engagementRepo.toggles))
	}
	if engagementRepo.toggles[0].kind != models.EngagementKindFollowing {
		t.Fatalf("expected following toggle, got %q", engagementRepo.toggles[0].kind)
	}
	if engagementRepo.toggles[1].kind != models.EngagementKindFollower {
		t.Fatalf("expected follower toggle, got %q", engagementRepo.toggles[1].kind)
	}
	if len(events.events) != 1 {
		t.Fatalf("expected follow event, got %d", len(events.events))
	}
}

func TestToggleFollowRejectsSelfInteraction(t *testing.T) {
	service := NewUserService(
		&fakeUserRepository{},
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
	)

	_, err := service.ToggleFollow(context.Background(), 1, 1)
	if err == nil {
		t.Fatalf("expected self follow to fail")
	}
	if err.Error() == constants.ErrUnknown.String() {
		t.Fatalf("expected domain self interaction error, got %v", err)
	}
}
