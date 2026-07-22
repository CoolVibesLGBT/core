package usecases

import (
	"context"
	"core/application/ports"
	"core/application/types"
	domainuser "core/domain/user"
	"core/models"
	"core/models/notifications"
	"core/models/post"
	modelutils "core/models/utils"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToggleLikeWritesExpectedEngagementPairsAndPublishesEvent(t *testing.T) {
	likerID := uuid.New()
	likeeID := uuid.New()
	userRepo := &fakeUserRepository{
		byPublicID: map[int64]*models.User{
			1: {ID: likerID, PublicID: 1, UserName: "liker"},
			2: {ID: likeeID, PublicID: 2, UserName: "likee"},
		},
	}
	engagementRepo := &fakeEngagementRepository{}
	events := &fakeEventPublisher{}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{}, WithEventPublisher(events))

	isLike, enabled, err := service.ToggleLike(context.Background(), models.User{ID: likerID, PublicID: 1}, 1, 2, true)
	if err != nil {
		t.Fatalf("ToggleLike() error = %v", err)
	}
	if !isLike || !enabled {
		t.Fatalf("expected like enabled, got isLike=%v enabled=%v", isLike, enabled)
	}
	if len(engagementRepo.reciprocalCalls) != 1 {
		t.Fatalf("expected one atomic like mutation, got %d", len(engagementRepo.reciprocalCalls))
	}
	pair, err := engagementRepo.reciprocalCalls[0].intent.EngagementPair()
	if err != nil || pair.Given != domainuser.EngagementLikeGiven || pair.Received != domainuser.EngagementLikeReceived {
		t.Fatalf("expected atomic like given/received pair, got %+v, %v", pair, err)
	}
	if len(events.events) != 1 {
		t.Fatalf("expected interaction event, got %d", len(events.events))
	}
}

func TestToggleDislikeWritesExpectedEngagementPairs(t *testing.T) {
	likerID := uuid.New()
	likeeID := uuid.New()
	userRepo := &fakeUserRepository{
		byPublicID: map[int64]*models.User{
			1: {ID: likerID, PublicID: 1},
			2: {ID: likeeID, PublicID: 2},
		},
	}
	engagementRepo := &fakeEngagementRepository{}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{}, WithEventPublisher(&fakeEventPublisher{}))

	isLike, enabled, err := service.ToggleLike(context.Background(), models.User{ID: likerID, PublicID: 1}, 1, 2, false)
	if err != nil {
		t.Fatalf("ToggleLike(false) error = %v", err)
	}
	if isLike || !enabled {
		t.Fatalf("expected dislike enabled, got isLike=%v enabled=%v", isLike, enabled)
	}
	if len(engagementRepo.reciprocalCalls) != 1 {
		t.Fatalf("expected one atomic dislike mutation, got %d", len(engagementRepo.reciprocalCalls))
	}
	pair, err := engagementRepo.reciprocalCalls[0].intent.EngagementPair()
	if err != nil || pair.Given != domainuser.EngagementDislikeGiven || pair.Received != domainuser.EngagementDislikeReceived {
		t.Fatalf("expected atomic dislike given/received pair, got %+v, %v", pair, err)
	}
}

func TestUserInteractionRejectsActorDifferentFromPrincipalBeforePersistence(t *testing.T) {
	engagementRepo := &fakeEngagementRepository{}
	service := NewUserService(
		&fakeUserRepository{},
		&fakePostRepository{},
		&fakeMediaRepository{},
		engagementRepo,
		&fakeNotificationRepository{},
	)

	_, _, err := service.ToggleLike(context.Background(), models.User{ID: uuid.New(), PublicID: 99}, 1, 2, true)
	if !errors.Is(err, domainuser.ErrActorMismatch) {
		t.Fatalf("ToggleLike(actor mismatch) error = %v; want ErrActorMismatch", err)
	}
	if len(engagementRepo.reciprocalCalls) != 0 {
		t.Fatalf("actor mismatch reached persistence: %#v", engagementRepo.reciprocalCalls)
	}
}

func TestToggleBlockAndSubscribeUseExistingStateForReturnedStatus(t *testing.T) {
	userA := uuid.New()
	userB := uuid.New()
	userRepo := &fakeUserRepository{
		byPublicID: map[int64]*models.User{
			1: {ID: userA, PublicID: 1},
			2: {ID: userB, PublicID: 2},
		},
	}
	engagementRepo := &fakeEngagementRepository{}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{}, WithEventPublisher(&fakeEventPublisher{}))

	blocked, err := service.ToggleBlock(context.Background(), models.User{ID: userA, PublicID: 1}, 1, 2)
	if err != nil {
		t.Fatalf("ToggleBlock() error = %v", err)
	}
	if !blocked {
		t.Fatalf("expected block to become enabled")
	}
	if len(engagementRepo.reciprocalCalls) != 1 {
		t.Fatalf("expected one atomic block mutation, got %#v", engagementRepo.reciprocalCalls)
	}
	pair, err := engagementRepo.reciprocalCalls[0].intent.EngagementPair()
	if err != nil || pair.Given != domainuser.EngagementBlocking || pair.Received != domainuser.EngagementBlockedBy {
		t.Fatalf("expected blocking/blocked_by pair, got %+v, %v", pair, err)
	}

	engagementRepo.reciprocalStates[domainuser.EngagementSubscribing] = true
	subscribed, err := service.ToggleSubscribe(context.Background(), models.User{ID: userA, PublicID: 1}, 1, 2)
	if err != nil {
		t.Fatalf("ToggleSubscribe() error = %v", err)
	}
	if subscribed {
		t.Fatalf("expected existing subscribe to become disabled")
	}
	if len(engagementRepo.reciprocalCalls) != 2 {
		t.Fatalf("expected one atomic subscribe mutation after block, got %#v", engagementRepo.reciprocalCalls)
	}
	pair, err = engagementRepo.reciprocalCalls[1].intent.EngagementPair()
	if err != nil || pair.Given != domainuser.EngagementSubscribing || pair.Received != domainuser.EngagementSubscribedBy {
		t.Fatalf("expected atomic subscribing/subscribed_by pair, got %+v, %v", pair, err)
	}
}

func TestFollowAndUnfollowSetStateIdempotently(t *testing.T) {
	followerID := uuid.New()
	followeeID := uuid.New()
	userRepo := &fakeUserRepository{byPublicID: map[int64]*models.User{
		1: {ID: followerID, PublicID: 1, UserName: "follower"},
		2: {ID: followeeID, PublicID: 2, UserName: "followee"},
	}}
	engagementRepo := &fakeEngagementRepository{}
	notifications := &fakeNotificationRepository{}
	events := &fakeEventPublisher{}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, notifications, WithEventPublisher(events))

	for attempt := 0; attempt < 2; attempt++ {
		enabled, err := service.Follow(context.Background(), 1, 2)
		if err != nil || !enabled {
			t.Fatalf("Follow() attempt %d = %v, %v; want true, nil", attempt+1, enabled, err)
		}
	}
	if len(engagementRepo.reciprocalCalls) != 2 {
		t.Fatalf("Follow() must make one atomic port call per command, got %d", len(engagementRepo.reciprocalCalls))
	}
	if len(events.events) != 1 {
		t.Fatalf("repeated Follow() published %d events; want 1", len(events.events))
	}
	if len(notifications.sent) != 2 {
		t.Fatalf("repeated Follow() sent %d notifications; want 2 from the first transition", len(notifications.sent))
	}
	assertInteractionEvent(t, events.events[0], domainuser.InteractionFollow, true)

	for attempt := 0; attempt < 2; attempt++ {
		enabled, err := service.Unfollow(context.Background(), 1, 2)
		if err != nil || enabled {
			t.Fatalf("Unfollow() attempt %d = %v, %v; want false, nil", attempt+1, enabled, err)
		}
	}
	if len(engagementRepo.reciprocalCalls) != 4 {
		t.Fatalf("Follow/Unfollow commands made %d port calls; want 4", len(engagementRepo.reciprocalCalls))
	}
	if len(events.events) != 2 {
		t.Fatalf("repeated Unfollow() left %d events; want 2 total", len(events.events))
	}
	if len(notifications.sent) != 4 {
		t.Fatalf("repeated Unfollow() left %d notifications; want 4 total", len(notifications.sent))
	}
	assertInteractionEvent(t, events.events[1], domainuser.InteractionFollow, false)
}

func TestBlockAndUnblockSetStateIdempotently(t *testing.T) {
	blockerID := uuid.New()
	blockedID := uuid.New()
	userRepo := &fakeUserRepository{byPublicID: map[int64]*models.User{
		1: {ID: blockerID, PublicID: 1},
		2: {ID: blockedID, PublicID: 2},
	}}
	engagementRepo := &fakeEngagementRepository{}
	events := &fakeEventPublisher{}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{}, WithEventPublisher(events))
	authUser := models.User{ID: blockerID, PublicID: 1}

	for attempt := 0; attempt < 2; attempt++ {
		enabled, err := service.Block(context.Background(), authUser, 1, 2)
		if err != nil || !enabled {
			t.Fatalf("Block() attempt %d = %v, %v; want true, nil", attempt+1, enabled, err)
		}
	}
	if len(events.events) != 1 {
		t.Fatalf("repeated Block() published %d events; want 1", len(events.events))
	}
	assertInteractionEvent(t, events.events[0], domainuser.InteractionBlock, true)

	for attempt := 0; attempt < 2; attempt++ {
		enabled, err := service.Unblock(context.Background(), authUser, 1, 2)
		if err != nil || enabled {
			t.Fatalf("Unblock() attempt %d = %v, %v; want false, nil", attempt+1, enabled, err)
		}
	}
	if len(engagementRepo.reciprocalCalls) != 4 {
		t.Fatalf("Block/Unblock commands made %d port calls; want 4", len(engagementRepo.reciprocalCalls))
	}
	if len(events.events) != 2 {
		t.Fatalf("repeated Unblock() left %d events; want 2 total", len(events.events))
	}
	assertInteractionEvent(t, events.events[1], domainuser.InteractionBlock, false)
}

type privatePhotoBlockRevokerFake struct {
	calls  int
	first  uuid.UUID
	second uuid.UUID
	err    error
}

func (r *privatePhotoBlockRevokerFake) RevokePrivatePhotoAccessBetween(_ context.Context, firstUserID, secondUserID uuid.UUID, _ time.Time) error {
	r.calls++
	r.first = firstUserID
	r.second = secondUserID
	return r.err
}

func TestBlockRevokesPrivatePhotoGrantsAndRetriesIdempotently(t *testing.T) {
	blockerID := uuid.New()
	blockedID := uuid.New()
	userRepo := &fakeUserRepository{byPublicID: map[int64]*models.User{
		1: {ID: blockerID, PublicID: 1},
		2: {ID: blockedID, PublicID: 2},
	}}
	engagementRepo := &fakeEngagementRepository{}
	revoker := &privatePhotoBlockRevokerFake{}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		engagementRepo,
		&fakeNotificationRepository{},
		WithPrivatePhotoBlockRevoker(revoker),
	)
	authUser := models.User{ID: blockerID, PublicID: 1}

	for attempt := 0; attempt < 2; attempt++ {
		enabled, err := service.Block(context.Background(), authUser, 1, 2)
		if err != nil || !enabled {
			t.Fatalf("Block() attempt %d = %v, %v; want true, nil", attempt+1, enabled, err)
		}
	}
	if revoker.calls != 2 {
		t.Fatalf("idempotent block called grant revoker %d times; want 2 so failed revocations can retry", revoker.calls)
	}
	if revoker.first != blockerID || revoker.second != blockedID {
		t.Fatalf("grant revoker pair = %s/%s", revoker.first, revoker.second)
	}

	if enabled, err := service.Unblock(context.Background(), authUser, 1, 2); err != nil || enabled {
		t.Fatalf("Unblock() = %v, %v; want false, nil", enabled, err)
	}
	if revoker.calls != 2 {
		t.Fatalf("unblock unexpectedly changed grant revoker calls to %d", revoker.calls)
	}
}

func TestReciprocalInteractionFailureDoesNotPublishEvent(t *testing.T) {
	actorID := uuid.New()
	targetID := uuid.New()
	userRepo := &fakeUserRepository{byPublicID: map[int64]*models.User{
		1: {ID: actorID, PublicID: 1},
		2: {ID: targetID, PublicID: 2},
	}}
	commitErr := errors.New("reciprocal transaction failed")
	engagementRepo := &fakeEngagementRepository{reciprocalErr: commitErr}
	events := &fakeEventPublisher{}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{}, WithEventPublisher(events))

	if _, err := service.Follow(context.Background(), 1, 2); !errors.Is(err, commitErr) {
		t.Fatalf("Follow() error = %v; want %v", err, commitErr)
	}
	if len(events.events) != 0 {
		t.Fatalf("failed reciprocal mutation published %d events; want 0", len(events.events))
	}
}

func assertInteractionEvent(t *testing.T, event interface{ Name() string }, interaction domainuser.InteractionKind, enabled bool) {
	t.Helper()
	toggled, ok := event.(domainuser.InteractionToggledEvent)
	if !ok {
		t.Fatalf("event type = %T; want domainuser.InteractionToggledEvent", event)
	}
	if toggled.Interaction != interaction || toggled.Enabled != enabled {
		t.Fatalf("event = %+v; want interaction=%s enabled=%v", toggled, interaction, enabled)
	}
}

func TestViewProfileRecordsOnlyForeignProfiles(t *testing.T) {
	viewer := models.User{ID: uuid.New(), PublicID: 10}
	target := models.User{ID: uuid.New(), PublicID: 20}
	userRepo := &fakeUserRepository{byPublicID: map[int64]*models.User{
		viewer.PublicID: &viewer,
		target.PublicID: &target,
	}}
	engagementRepo := &fakeEngagementRepository{recordViewResult: true}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{})

	counted, err := service.ViewProfile(context.Background(), &viewer, target.PublicID)
	if err != nil || !counted {
		t.Fatalf("ViewProfile(foreign) = %v, %v; want true, nil", counted, err)
	}
	if len(engagementRepo.recordedViews) != 1 {
		t.Fatalf("expected one recorded view, got %#v", engagementRepo.recordedViews)
	}
	record := engagementRepo.recordedViews[0]
	if record.engagerID != viewer.ID || record.engageeID != target.ID || record.contentableID != target.ID ||
		record.contentableType != models.EngagementContentableTypeUser || record.kind != models.EngagementKindViewReceived {
		t.Fatalf("unexpected profile view record: %#v", record)
	}

	counted, err = service.ViewProfile(context.Background(), &viewer, viewer.PublicID)
	if err != nil || counted {
		t.Fatalf("ViewProfile(self) = %v, %v; want false, nil", counted, err)
	}
	if len(engagementRepo.recordedViews) != 1 {
		t.Fatalf("self view must not reach repository, got %#v", engagementRepo.recordedViews)
	}
}

func TestFetchProfileViewEngagementsRequiresProfileOwner(t *testing.T) {
	owner := models.User{ID: uuid.New(), PublicID: 10}
	visitor := models.User{ID: uuid.New(), PublicID: 20}
	service := NewUserService(&fakeUserRepository{}, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})

	for _, kind := range []models.EngagementKind{models.EngagementKindViewGiven, models.EngagementKindViewReceived} {
		_, _, err := service.FetchUserEngagements(context.Background(), &visitor, owner.ID, models.EngagementContentableTypeUser, kind, nil, 10)
		if !errors.Is(err, ErrPrivateViewEngagements) {
			t.Fatalf("FetchUserEngagements(%s, visitor) error = %v; want ErrPrivateViewEngagements", kind, err)
		}

		if _, _, err := service.FetchUserEngagements(context.Background(), &owner, owner.ID, models.EngagementContentableTypeUser, kind, nil, 10); err != nil {
			t.Fatalf("FetchUserEngagements(%s, owner) error = %v", kind, err)
		}
	}
}

func TestUpdateUserProfileNormalizesFieldsAndUpsertsLocation(t *testing.T) {
	userID := uuid.New()
	userRepo := &fakeUserRepository{
		byID: map[uuid.UUID]*models.User{},
		byUUID: map[uuid.UUID]*models.User{
			userID: {
				ID:              userID,
				PublicID:        10,
				UserName:        "old",
				DisplayName:     "Old",
				DefaultLanguage: "en",
				Bio:             modelutils.MakeLocalizedString("en", "old bio"),
			},
		},
	}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})

	updated, err := service.UpdateUserProfile(context.Background(), models.User{ID: userID, PublicID: 10}, UpdateUserProfileInput{
		UserName:            "  Jane.Doe  ",
		DisplayName:         "  Jane Doe  ",
		Bio:                 "new bio",
		PrivacyLevel:        "private",
		LocationLatitude:    "41.0082",
		LocationLongitude:   "28.9784",
		LocationCity:        "Istanbul",
		LocationCountryCode: "TR",
	})
	if err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}
	if updated == nil || updated.UserName != "jane.doe" || updated.DisplayName != "Jane Doe" {
		t.Fatalf("expected normalized updated user, got %#v", updated)
	}
	if userRepo.updated == nil || userRepo.updated.Bio.GetLocalizedString("en") != "new bio" {
		t.Fatalf("expected updated localized bio, got %#v", userRepo.updated)
	}
	if userRepo.upsertLocation == nil || userRepo.upsertLocation.ContentableID != userID || *userRepo.upsertLocation.City != "Istanbul" {
		t.Fatalf("expected location upsert for user, got %#v", userRepo.upsertLocation)
	}
}

func TestUpdateUserProfileUsesReloadedPasswordWithLightweightSessionUser(t *testing.T) {
	userID := uuid.New()
	persisted := &models.User{
		ID:              userID,
		PublicID:        10,
		UserName:        "alice",
		DisplayName:     "Alice",
		Password:        "persisted-password-hash",
		DefaultLanguage: "en",
		Bio:             modelutils.MakeLocalizedString("en", "bio"),
	}
	userRepo := &fakeUserRepository{
		byID:   map[uuid.UUID]*models.User{},
		byUUID: map[uuid.UUID]*models.User{userID: persisted},
	}
	hasher := &fakePasswordHasher{compareOK: true}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithPasswordHasher(hasher),
	)

	// Session users intentionally have no Password field.
	_, err := service.UpdateUserProfile(context.Background(), models.User{ID: userID, PublicID: 10}, UpdateUserProfileInput{
		CurrentPassword: "plain-current-password",
	})
	if err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}
	if hasher.comparedHash != persisted.Password || hasher.comparedRaw != "plain-current-password" {
		t.Fatalf("password comparison = (%q, %q), want persisted hash and submitted password", hasher.comparedHash, hasher.comparedRaw)
	}
}

func TestUpdateUserProfilePersistsEmailWebsiteAndNewPassword(t *testing.T) {
	userID := uuid.New()
	persisted := &models.User{
		ID:              userID,
		PublicID:        10,
		UserName:        "alice",
		DisplayName:     "Alice",
		Email:           "old@example.com",
		Password:        "old-password-hash",
		DefaultLanguage: "en",
	}
	userRepo := &fakeUserRepository{byUUID: map[uuid.UUID]*models.User{userID: persisted}}
	hasher := &fakePasswordHasher{compareOK: true}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
		WithPasswordHasher(hasher),
	)

	updated, err := service.UpdateUserProfile(context.Background(), models.User{ID: userID, PublicID: 10}, UpdateUserProfileInput{
		CurrentPassword:         "current-secret",
		NewPassword:             "new-secret",
		NewPasswordConfirmation: "new-secret",
		Email:                   "  New@Example.COM ",
		Website:                 "example.com/about",
	})
	if err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}
	if hasher.comparedHash != "old-password-hash" || hasher.comparedRaw != "current-secret" || hasher.hashedRaw != "new-secret" {
		t.Fatalf("unexpected password operations: %#v", hasher)
	}
	if userRepo.profileUpdate == nil || userRepo.profileUpdate.Email == nil || *userRepo.profileUpdate.Email != "new@example.com" ||
		userRepo.profileUpdate.Website == nil || *userRepo.profileUpdate.Website != "https://example.com/about" ||
		userRepo.profileUpdate.PasswordHash == nil || *userRepo.profileUpdate.PasswordHash != "hashed:new-secret" {
		t.Fatalf("unexpected atomic profile command: %#v", userRepo.profileUpdate)
	}
	if updated.Email != "new@example.com" || updated.Website != "https://example.com/about" || updated.Password != "hashed:new-secret" {
		t.Fatalf("updated user is missing persisted fields: %#v", updated)
	}
	payload, err := json.Marshal(updated)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "hashed:new-secret") || strings.Contains(string(payload), "new@example.com") {
		t.Fatalf("profile response exposed credentials: %s", payload)
	}
}

func TestUpdateUserProfileValidatesEveryFieldBeforePersistence(t *testing.T) {
	userID := uuid.New()
	userRepo := &fakeUserRepository{byUUID: map[uuid.UUID]*models.User{
		userID: {ID: userID, PublicID: 10, UserName: "old", DefaultLanguage: "en"},
	}}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})

	_, err := service.UpdateUserProfile(context.Background(), models.User{ID: userID}, UpdateUserProfileInput{
		UserName:         "would-have-been-written",
		LocationLatitude: "41.0",
	})
	if !errors.Is(err, ErrLocationCoordinates) {
		t.Fatalf("UpdateUserProfile(incomplete location) error = %v, want %v", err, ErrLocationCoordinates)
	}
	if userRepo.profileUpdate != nil || userRepo.updated != nil {
		t.Fatalf("invalid input reached persistence: command=%#v user=%#v", userRepo.profileUpdate, userRepo.updated)
	}
}

func TestUpdateUserProfileDoesNotMutateLoadedUserWhenAtomicPersistenceFails(t *testing.T) {
	userID := uuid.New()
	persisted := &models.User{
		ID: userID, PublicID: 10, UserName: "old", DisplayName: "Old", Email: "old@example.com", DefaultLanguage: "en",
	}
	persistenceErr := errors.New("forced location write failure")
	userRepo := &fakeUserRepository{
		byUUID:           map[uuid.UUID]*models.User{userID: persisted},
		profileUpdateErr: persistenceErr,
	}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})

	_, err := service.UpdateUserProfile(context.Background(), models.User{ID: userID}, UpdateUserProfileInput{
		UserName:          "new-name",
		Email:             "new@example.com",
		LocationLatitude:  "41.0082",
		LocationLongitude: "28.9784",
	})
	if !errors.Is(err, persistenceErr) {
		t.Fatalf("UpdateUserProfile() error = %v, want %v", err, persistenceErr)
	}
	if persisted.UserName != "old" || persisted.Email != "old@example.com" || userRepo.upsertLocation != nil {
		t.Fatalf("failed atomic command mutated in-memory state: user=%#v location=%#v", persisted, userRepo.upsertLocation)
	}
}

func TestUpdateUserProfileRejectsLegacyOrIncompletePasswordFields(t *testing.T) {
	userID := uuid.New()
	tests := []struct {
		name    string
		input   UpdateUserProfileInput
		wantErr error
	}{
		{name: "legacy password", input: UpdateUserProfileInput{Password: "secret"}, wantErr: ErrLegacyPasswordField},
		{name: "missing current", input: UpdateUserProfileInput{NewPassword: "new", NewPasswordConfirmation: "new"}, wantErr: domainuser.ErrCurrentPasswordRequired},
		{name: "mismatched confirmation", input: UpdateUserProfileInput{CurrentPassword: "current", NewPassword: "new", NewPasswordConfirmation: "different"}, wantErr: domainuser.ErrPasswordConfirmationMismatch},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userRepo := &fakeUserRepository{byUUID: map[uuid.UUID]*models.User{userID: {ID: userID}}}
			service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, &fakeEngagementRepository{}, &fakeNotificationRepository{})
			if _, err := service.UpdateUserProfile(context.Background(), models.User{ID: userID}, test.input); !errors.Is(err, test.wantErr) {
				t.Fatalf("UpdateUserProfile() error = %v, want %v", err, test.wantErr)
			}
			if userRepo.profileUpdate != nil {
				t.Fatalf("invalid password fields reached persistence: %#v", userRepo.profileUpdate)
			}
		})
	}
}

func TestProfileMediaUpdatesReloadPersistedUserBeforeSave(t *testing.T) {
	userID := uuid.New()
	persisted := &models.User{
		ID:       userID,
		PublicID: 10,
		UserName: "alice",
		Email:    "alice@example.com",
		Password: "persisted-password-hash",
	}
	userRepo := &fakeUserRepository{byUUID: map[uuid.UUID]*models.User{userID: persisted}}
	service := NewUserService(
		userRepo,
		&fakePostRepository{},
		&fakeMediaRepository{},
		&fakeEngagementRepository{},
		&fakeNotificationRepository{},
	)
	sessionUser := &models.User{ID: userID, PublicID: 10}

	if _, err := service.UpdateAvatar(context.Background(), nil, sessionUser); err != nil {
		t.Fatalf("UpdateAvatar() error = %v", err)
	}
	if userRepo.updated != persisted || userRepo.updated.AvatarID == nil ||
		userRepo.updated.Password != "persisted-password-hash" || userRepo.updated.Email != "alice@example.com" {
		t.Fatalf("avatar update saved a partial session user: %#v", userRepo.updated)
	}

	userRepo.updated = nil
	if _, err := service.UpdateCover(context.Background(), nil, sessionUser); err != nil {
		t.Fatalf("UpdateCover() error = %v", err)
	}
	if userRepo.updated != persisted || userRepo.updated.CoverID == nil ||
		userRepo.updated.Password != "persisted-password-hash" || userRepo.updated.Email != "alice@example.com" {
		t.Fatalf("cover update saved a partial session user: %#v", userRepo.updated)
	}
}

func TestUserServiceSimpleRepositoryDelegates(t *testing.T) {
	authUser := models.User{ID: uuid.New(), PublicID: 10}
	nextCursor := int64(100)
	notificationCursor := time.Now().UTC()
	distance := 20.5
	userRepo := &fakeUserRepository{
		usersStartingWith:   []models.User{{ID: uuid.New(), UserName: "ali"}},
		fetchNearbyUsers:    []types.NearbyUser{{PublicID: 11}},
		fetchNearbyDistance: &distance,
		fetchLiveUsers:      []*models.User{{ID: uuid.New(), PublicID: 12}},
		notifications:       []*notifications.Notification{{ID: uuid.New()}},
		nextNotification:    &notificationCursor,
	}
	postRepo := &fakePostRepository{userMediasCursor: &nextCursor}
	engagementRepo := &fakeEngagementRepository{}
	service := NewUserService(userRepo, postRepo, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{})

	if err := service.UpsertUserPreference(context.Background(), authUser.ID, "pref", true); err != nil {
		t.Fatalf("UpsertUserPreference() error = %v", err)
	}
	if userRepo.upsertPreference == nil || userRepo.upsertPreference.userID != authUser.ID || userRepo.upsertPreference.preferenceItemID != "pref" || !userRepo.upsertPreference.enabled {
		t.Fatalf("expected preference upsert call, got %#v", userRepo.upsertPreference)
	}

	users, err := service.GetUsersStartingWith("a", 5)
	if err != nil || len(users) != 1 {
		t.Fatalf("GetUsersStartingWith() = %#v, %v", users, err)
	}
	nearby, gotDistance, err := service.FetchNearbyUsers(types.Filter{Limit: 6})
	if err != nil || len(nearby) != 1 || gotDistance == nil || *gotDistance != distance || userRepo.fetchNearbyFilter.Limit != 6 {
		t.Fatalf("FetchNearbyUsers() = %#v, %v, filter=%#v, err=%v", nearby, gotDistance, userRepo.fetchNearbyFilter, err)
	}
	live, err := service.FetchLiveUsers(types.Filter{Limit: 7})
	if err != nil || len(live) != 1 || userRepo.fetchLiveFilter.Limit != 7 {
		t.Fatalf("FetchLiveUsers() = %#v, filter=%#v, err=%v", live, userRepo.fetchLiveFilter, err)
	}
	if err := service.DeleteUser(types.Filter{AuthUser: &types.Actor{ID: authUser.ID, PublicID: authUser.PublicID, Role: string(authUser.UserRole)}}); err != nil || userRepo.deletedFilter.AuthUser == nil {
		t.Fatalf("DeleteUser() err=%v filter=%#v", err, userRepo.deletedFilter)
	}

	notifications, next, err := service.FetchUserNotifications(context.Background(), &authUser, nil, 3)
	if err != nil || len(notifications) != 1 || next == nil || !next.Equal(notificationCursor) {
		t.Fatalf("FetchUserNotifications() = %#v, %v, %v", notifications, next, err)
	}

	engagements, _, err := service.FetchUserEngagements(context.Background(), &authUser, uuid.New(), models.EngagementContentableTypeUser, models.EngagementKindFollowing, nil, 2)
	if err != nil || engagements != nil {
		t.Fatalf("FetchUserEngagements() = %#v, %v", engagements, err)
	}

	checkin, err := service.CheckIn(context.Background(), ports.FormData{}, &authUser, post.PostKindCheckIn)
	if err != nil || checkin.PostKind != post.PostKindCheckIn {
		t.Fatalf("CheckIn() = %#v, %v", checkin, err)
	}
	checkins, err := service.FetchCheckIns(types.Filter{PostKind: post.PostKindCheckIn})
	if err != nil || len(checkins.Posts) != 1 || checkins.Posts[0].PostKind != string(post.PostKindCheckIn) {
		t.Fatalf("FetchCheckIns() = %#v, %v", checkins, err)
	}
}
