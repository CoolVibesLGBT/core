package usecases

import (
	"context"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestRegisterUserCreatesUserAndReturnsIssuedToken(t *testing.T) {
	userRepo := &fakeUserRepository{byID: map[uuid.UUID]*models.User{}}
	captcha := &fakeCaptchaVerifier{valid: true}
	events := &fakeEventPublisher{}
	tokenIssuer := &fakeTokenIssuer{token: "issued-token"}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithCaptchaVerifier(captcha),
		WithPasswordHasher(&fakePasswordHasher{}),
		WithTokenIssuer(tokenIssuer),
		WithPublicIDGenerator(&fakePublicIDGenerator{next: 4242}),
		WithEventPublisher(events),
	)

	user, token, err := service.RegisterUser(context.Background(), RegisterInput{
		Name:           "  Alice Wonderland  ",
		Nickname:       "AliceHandle",
		Password:       "secret-123",
		Domain:         string(models.CoolVibes),
		Email:          "Alice@Example.COM",
		RecaptchaToken: "captcha-token",
	})
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}

	if token != "issued-token" {
		t.Fatalf("expected issued token, got %q", token)
	}
	if tokenIssuer.userID != userRepo.created.ID || tokenIssuer.publicID != 4242 {
		t.Fatalf("token subject changed: internal=%s public=%d", tokenIssuer.userID, tokenIssuer.publicID)
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
	if userRepo.created.UserName != "alicehandle" {
		t.Fatalf("UserName = %q, want normalized nickname %q", userRepo.created.UserName, "alicehandle")
	}
	if userRepo.created.DisplayName != "Alice Wonderland" {
		t.Fatalf("DisplayName = %q, want trimmed name %q", userRepo.created.DisplayName, "Alice Wonderland")
	}
	if userRepo.checkedUsername != "alicehandle" {
		t.Fatalf("username uniqueness input = %q, want %q", userRepo.checkedUsername, "alicehandle")
	}
	if userRepo.checkedEmail != "alice@example.com" {
		t.Fatalf("email uniqueness input = %q, want %q", userRepo.checkedEmail, "alice@example.com")
	}
	if userRepo.checkedNameOrMail != "" {
		t.Fatalf("combined identity lookup should not be used, got %q", userRepo.checkedNameOrMail)
	}
	if captcha.token != "captcha-token" {
		t.Fatalf("expected recaptcha token to be used, got %q", captcha.token)
	}
	if len(events.events) != 1 {
		t.Fatalf("expected registration event, got %d", len(events.events))
	}
}

func TestRegisterUserRejectsUsernameAndEmailCollisionsSeparately(t *testing.T) {
	tests := []struct {
		name             string
		usernameExists   bool
		emailExists      bool
		wantErr          error
		wantCheckedEmail string
	}{
		{
			name:           "username",
			usernameExists: true,
			wantErr:        ErrUsernameAlreadyExists,
		},
		{
			name:             "email",
			emailExists:      true,
			wantErr:          ErrEmailAlreadyExists,
			wantCheckedEmail: "alice@example.com",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userRepo := &fakeUserRepository{
				usernameExists: test.usernameExists,
				emailExists:    test.emailExists,
			}
			hasher := &fakePasswordHasher{}
			service := NewUserService(
				userRepo,
				&fakePostRepository{},
				&fakeMediaRepository{},
				&fakeEngagementRepository{},
				&fakeNotificationRepository{},
				WithCaptchaVerifier(&fakeCaptchaVerifier{valid: true}),
				WithPasswordHasher(hasher),
			)

			_, _, err := service.RegisterUser(context.Background(), RegisterInput{
				Name: "Alice", Nickname: "AliceHandle", Password: "secret-123",
				Domain: string(models.CoolVibes), Email: "Alice@Example.COM",
			})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("RegisterUser() error = %v, want %v", err, test.wantErr)
			}
			if userRepo.checkedUsername != "alicehandle" {
				t.Fatalf("username uniqueness input = %q, want %q", userRepo.checkedUsername, "alicehandle")
			}
			if userRepo.checkedEmail != test.wantCheckedEmail {
				t.Fatalf("email uniqueness input = %q, want %q", userRepo.checkedEmail, test.wantCheckedEmail)
			}
			if userRepo.checkedNameOrMail != "" {
				t.Fatalf("combined identity lookup should not be used, got %q", userRepo.checkedNameOrMail)
			}
			if hasher.hashedRaw != "" {
				t.Fatalf("password should not be hashed after an identity collision, got %q", hasher.hashedRaw)
			}
			if userRepo.created != nil {
				t.Fatal("user should not be persisted after an identity collision")
			}
		})
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
		byUsername: &models.User{ID: userID, PublicID: 55, UserName: "alice", Password: "stored-hash", UserRole: constants.UserRoleUser},
	}
	tokenIssuer := &fakeTokenIssuer{token: "login-token"}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithPasswordHasher(&fakePasswordHasher{compareOK: true}),
		WithTokenIssuer(tokenIssuer),
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
	if tokenIssuer.userID != userID || tokenIssuer.publicID != 55 {
		t.Fatalf("token subject changed: internal=%s public=%d", tokenIssuer.userID, tokenIssuer.publicID)
	}
}

func TestLoginUserRejectsBotsWithoutClaimingTheirPassword(t *testing.T) {
	for _, storedPassword := range []string{"", "stored-bot-hash"} {
		t.Run(storedPassword, func(t *testing.T) {
			userRepo := &fakeUserRepository{
				byUsername: &models.User{ID: uuid.New(), PublicID: 55, UserName: "broadcast-bot", Password: storedPassword, UserRole: constants.UserRoleUser, IsBot: true},
			}
			hasher := &fakePasswordHasher{compareOK: true}
			service := NewUserService(
				userRepo,
				&fakePostRepository{},
				&fakeMediaRepository{},
				&fakeEngagementRepository{},
				&fakeNotificationRepository{},
				WithPasswordHasher(hasher),
				WithTokenIssuer(&fakeTokenIssuer{token: "must-not-be-issued"}),
			)

			user, token, err := service.LoginUser(context.Background(), LoginInput{UserName: "broadcast-bot", Password: "attacker-selected"})
			if !errors.Is(err, ErrInvalidCredentials) || user != nil || token != "" {
				t.Fatalf("LoginUser(bot) = user %#v, token %q, error %v", user, token, err)
			}
			if userRepo.updated != nil || hasher.hashedRaw != "" || hasher.comparedRaw != "" {
				t.Fatalf("bot login mutated or verified credentials: updated=%#v hasher=%#v", userRepo.updated, hasher)
			}
		})
	}
}

func TestLoginUserRejectsSuspendedAccountRolesBeforePasswordVerification(t *testing.T) {
	for _, role := range []constants.UserRole{
		constants.UserRoleBanned,
		constants.UserRoleDeleted,
		constants.UserRolePending,
	} {
		t.Run(string(role), func(t *testing.T) {
			userRepo := &fakeUserRepository{
				byUsername: &models.User{
					ID: uuid.New(), PublicID: 55, UserName: "suspended-user",
					Password: "stored-hash", UserRole: role,
				},
			}
			hasher := &fakePasswordHasher{compareOK: true}
			issuer := &fakeTokenIssuer{token: "must-not-be-issued"}
			service := NewUserService(
				userRepo,
				&fakePostRepository{},
				&fakeMediaRepository{},
				&fakeEngagementRepository{},
				&fakeNotificationRepository{},
				WithPasswordHasher(hasher),
				WithTokenIssuer(issuer),
			)

			user, token, err := service.LoginUser(context.Background(), LoginInput{
				UserName: "suspended-user",
				Password: "correct-password",
			})
			if !errors.Is(err, ErrInvalidCredentials) || user != nil || token != "" {
				t.Fatalf("LoginUser(%s) = user %#v, token %q, error %v", role, user, token, err)
			}
			if hasher.comparedRaw != "" || issuer.userID != uuid.Nil {
				t.Fatalf("suspended account reached password/token path: hasher=%#v issuer=%#v", hasher, issuer)
			}
		})
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
	engagementRepo := &fakeEngagementRepository{}
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
	if len(engagementRepo.reciprocalCalls) != 1 {
		t.Fatalf("expected one atomic reciprocal mutation, got %d", len(engagementRepo.reciprocalCalls))
	}
	call := engagementRepo.reciprocalCalls[0]
	if call.actorID != followerID || call.targetID != followeeID {
		t.Fatalf("unexpected reciprocal mutation IDs: %#v", call)
	}
	pair, err := call.intent.EngagementPair()
	if err != nil || pair.Given != domainuser.EngagementFollowing || pair.Received != domainuser.EngagementFollower {
		t.Fatalf("unexpected follow pair: %+v, %v", pair, err)
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
