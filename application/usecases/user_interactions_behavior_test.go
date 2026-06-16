package usecases

import (
	"context"
	"core/application/ports"
	"core/models"
	"core/models/notifications"
	"core/models/post"
	modelutils "core/models/utils"
	"core/types"
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
	if len(engagementRepo.toggles) != 2 {
		t.Fatalf("expected two engagement toggles, got %d", len(engagementRepo.toggles))
	}
	if engagementRepo.toggles[0].kind != models.EngagementKindLikeGiven || engagementRepo.toggles[1].kind != models.EngagementKindLikeReceived {
		t.Fatalf("expected like given/received toggles, got %#v", engagementRepo.toggles)
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
	if engagementRepo.toggles[0].kind != models.EngagementKindDislikeGiven || engagementRepo.toggles[1].kind != models.EngagementKindDisLikeReceived {
		t.Fatalf("expected dislike given/received toggles, got %#v", engagementRepo.toggles)
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
	engagementRepo := &fakeEngagementRepository{has: map[models.EngagementKind]bool{
		models.EngagementKindBlocking: false,
	}}
	service := NewUserService(userRepo, &fakePostRepository{}, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{}, WithEventPublisher(&fakeEventPublisher{}))

	blocked, err := service.ToggleBlock(context.Background(), models.User{ID: userA, PublicID: 1}, 1, 2)
	if err != nil {
		t.Fatalf("ToggleBlock() error = %v", err)
	}
	if !blocked {
		t.Fatalf("expected block to become enabled")
	}
	if engagementRepo.toggles[0].kind != models.EngagementKindBlocking || engagementRepo.toggles[1].kind != models.EngagementKindBlockedBy {
		t.Fatalf("expected blocking/blocked_by toggles, got %#v", engagementRepo.toggles)
	}

	engagementRepo.toggles = nil
	engagementRepo.has = map[models.EngagementKind]bool{models.EngagementKindSubscribing: true}
	subscribed, err := service.ToggleSubscribe(context.Background(), models.User{ID: userA, PublicID: 1}, 1, 2)
	if err != nil {
		t.Fatalf("ToggleSubscribe() error = %v", err)
	}
	if subscribed {
		t.Fatalf("expected existing subscribe to become disabled")
	}
	if engagementRepo.toggles[0].kind != models.EngagementKindSubscribing || engagementRepo.toggles[1].kind != models.EngagementKindSubscribedBy {
		t.Fatalf("expected subscribing/subscribed_by toggles, got %#v", engagementRepo.toggles)
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

func TestUserServiceSimpleRepositoryDelegates(t *testing.T) {
	authUser := models.User{ID: uuid.New(), PublicID: 10}
	nextCursor := int64(100)
	notificationCursor := time.Now().UTC()
	distance := 20.5
	userRepo := &fakeUserRepository{
		usersStartingWith:   []models.User{{ID: uuid.New(), UserName: "ali"}},
		fetchNearbyUsers:    []*models.User{{ID: uuid.New(), PublicID: 11}},
		fetchNearbyDistance: &distance,
		fetchLiveUsers:      []*models.User{{ID: uuid.New(), PublicID: 12}},
		notifications:       []*notifications.Notification{{ID: uuid.New()}},
		nextNotification:    &notificationCursor,
	}
	postRepo := &fakePostRepository{userMediasCursor: &nextCursor}
	engagementRepo := &fakeEngagementRepository{}
	service := NewUserService(userRepo, postRepo, &fakeMediaRepository{}, engagementRepo, &fakeNotificationRepository{})

	if err := service.UpsertUserPreference(context.Background(), authUser, "pref", "3", true); err != nil {
		t.Fatalf("UpsertUserPreference() error = %v", err)
	}
	if userRepo.upsertPreference == nil || userRepo.upsertPreference.preferenceItemID != "pref" || userRepo.upsertPreference.bitIndex != "3" || !userRepo.upsertPreference.enabled {
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
	if err := service.DeleteUser(types.Filter{AuthUser: &authUser}); err != nil || userRepo.deletedFilter.AuthUser == nil {
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
	if err != nil || len(checkins.Posts) != 1 || checkins.Posts[0].PostKind != post.PostKindCheckIn {
		t.Fatalf("FetchCheckIns() = %#v, %v", checkins, err)
	}
}
