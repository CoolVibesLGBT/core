package repositories

import (
	"context"
	"core/application/ports"
	domainuser "core/domain/user"
	"core/helpers"
	"core/models"
	"core/models/notifications"
	"errors"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

func TestFriendshipLifecyclePrivacyAndRetriesIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	ctx := context.Background()
	node, _ := helpers.NewDefaultNode()
	friends := NewFriendshipRepository(db, node)
	profiles := &UserRepository{db: db}
	if err := db.Model(&owner).Update("privacy_level", "friends_only").Error; err != nil {
		t.Fatal(err)
	}
	assertAccess := func(want bool) {
		t.Helper()
		allowed, err := profiles.CanViewUserProfile(ctx, owner.ID, viewer.ID)
		if err != nil || allowed != want {
			t.Fatalf("profile access=%v %v, want %v", allowed, err, want)
		}
	}
	assertAccess(false)
	request, err := friends.RequestFriendship(ctx, viewer.ID, owner.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := friends.RequestFriendship(ctx, owner.ID, viewer.PublicID)
	if err != nil || repeated.Status != domainuser.FriendshipPending || repeated.PublicID != request.PublicID {
		t.Fatalf("crossed request autoaccepted/duplicated: %+v %v", repeated, err)
	}
	if _, err := friends.ChangeFriendship(ctx, viewer.ID, request.PublicID, domainuser.FriendshipAccepted); !errors.Is(err, domainuser.ErrFriendshipForbidden) {
		t.Fatalf("requester approval=%v", err)
	}
	if _, err := friends.ChangeFriendship(ctx, uuid.New(), request.PublicID, domainuser.FriendshipAccepted); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("outsider approval=%v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := friends.ChangeFriendship(ctx, owner.ID, request.PublicID, domainuser.FriendshipAccepted); err != nil {
			t.Fatal(err)
		}
	}
	assertAccess(true)
	profile, err := profiles.FetchUserProfileForViewer(ctx, owner.UserName, viewer.ID)
	if err != nil || int64(profile.PublicID) != owner.PublicID {
		t.Fatalf("friends-only profile=%+v %v", profile, err)
	}
	rows, _, err := friends.ListFriendships(ctx, viewer.ID, domainuser.FriendshipScopeFriends, nil, 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("friends=%+v %v", rows, err)
	}
	var notices int64
	if err := db.Model(&notifications.Notification{}).Count(&notices).Error; err != nil || notices != 2 {
		t.Fatalf("notices duplicated=%d %v", notices, err)
	}
	if _, err := friends.ChangeFriendship(ctx, viewer.ID, request.PublicID, domainuser.FriendshipRemoved); err != nil {
		t.Fatal(err)
	}
	assertAccess(false)
	second, err := friends.RequestFriendship(ctx, viewer.ID, owner.PublicID)
	if err != nil || second.PublicID == request.PublicID {
		t.Fatalf("new request reused stale decision ID: %+v %v", second, err)
	}
	if _, err := friends.ChangeFriendship(ctx, owner.ID, request.PublicID, domainuser.FriendshipAccepted); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("stale request accepted: %v", err)
	}
}

func TestFriendshipBlockAndNotificationFailureAreAtomicIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	first, second := privatePhotoTestUsers(t, db)
	ctx := context.Background()
	node, _ := helpers.NewDefaultNode()
	friends := NewFriendshipRepository(db, node)
	record, err := friends.RequestFriendship(ctx, first.ID, second.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := friends.ChangeFriendship(ctx, second.ID, record.PublicID, domainuser.FriendshipAccepted); err != nil {
		t.Fatal(err)
	}
	repo := NewEngagementRepository(db)
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, second.ID, first.ID, socialIntent(t, domainuser.InteractionBlock, true)); err != nil {
		t.Fatal(err)
	}
	if _, err := friends.RequestFriendship(ctx, first.ID, second.PublicID); !errors.Is(err, domainuser.ErrInteractionBlocked) {
		t.Fatalf("blocked request=%v", err)
	}
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, second.ID, first.ID, socialIntent(t, domainuser.InteractionBlock, false)); err != nil {
		t.Fatal(err)
	}
	rows, _, err := friends.ListFriendships(ctx, first.ID, domainuser.FriendshipScopeFriends, nil, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("friendship restored after unblock=%+v %v", rows, err)
	}
	if err := db.Exec("DROP TABLE notification_deliveries").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := friends.RequestFriendship(ctx, first.ID, second.PublicID); err == nil {
		t.Fatal("request committed without its notification")
	}
	var stored models.EngagementDetail
	if err := db.First(&stored, "kind = ? AND details->>'public_id' = ?", models.EngagementKindFriendship, strconv.FormatInt(record.PublicID, 10)).Error; err != nil {
		t.Fatal(err)
	}
	details, err := decodeFriendshipDetails(stored.Details)
	if err != nil || details.Status != domainuser.FriendshipRemoved {
		t.Fatalf("failed request changed relationship=%+v %v", details, err)
	}
}

func TestMatchRemovalAndCompatibilityRankingIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	ctx := context.Background()
	repo := NewMatchesRepository(db, nil)
	other := models.User{ID: uuid.New(), PublicID: 30, Hobbies: pq.StringArray{"music"}, Languages: pq.StringArray{"en"}}
	if err := db.Omit("Avatar", "Cover", "Wallet", "Location", "Engagements").Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&actor).Updates(map[string]any{"hobbies": pq.StringArray{" MUSIC "}, "languages": pq.StringArray{"EN"}}).Error; err != nil {
		t.Fatal(err)
	}
	users, err := repo.GetUnseenUsers(ctx, actor.ID, 10)
	if err != nil || len(users) != 2 || int64(users[0].PublicID) != other.PublicID {
		t.Fatalf("compatibility ranking=%+v %v", users, err)
	}
	for _, u := range []models.User{actor, target} {
		if _, err := repo.RecordView(ctx, u.ID, map[uuid.UUID]uuid.UUID{actor.ID: target.ID, target.ID: actor.ID}[u.ID], domainuser.MatchReactionLike); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := repo.RemoveMatch(ctx, actor.ID, target.ID); err != nil {
			t.Fatal(err)
		}
	}
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindMatched, 0)
	assertLogicalMatchDetailCount(t, db, target.ID, actor.ID, models.EngagementKindMatched, 0)
	assertUserEngagementCount(t, db, actor.ID, "match_count", 0)
	assertUserEngagementCount(t, db, target.ID, "match_count", 0)
	maxDistance := 10.0
	users, err = repo.GetUnseenUsers(ctx, actor.ID, 10, domainuser.MatchCriteria{MaxDistanceKM: &maxDistance})
	if err != nil || len(users) != 0 {
		t.Fatalf("unknown positions passed distance filter=%+v %v", users, err)
	}
}

func TestFollowersAndMutualProfilePrivacyIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	ctx := context.Background()
	profiles := &UserRepository{db: db}
	repo := NewEngagementRepository(db)
	if err := db.Model(&owner).Update("privacy_level", "followers_only").Error; err != nil {
		t.Fatal(err)
	}
	allowed, err := profiles.CanViewUserProfile(ctx, owner.ID, viewer.ID)
	if err != nil || allowed {
		t.Fatal("nonfollower saw profile", err)
	}
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, viewer.ID, owner.ID, socialIntent(t, domainuser.InteractionFollow, true)); err != nil {
		t.Fatal(err)
	}
	allowed, err = profiles.CanViewUserProfile(ctx, owner.ID, viewer.ID)
	if err != nil || !allowed {
		t.Fatal("follower could not see profile", err)
	}
	if err := db.Model(&owner).Update("privacy_level", "mutuals_only").Error; err != nil {
		t.Fatal(err)
	}
	allowed, err = profiles.CanViewUserProfile(ctx, owner.ID, viewer.ID)
	if err != nil || allowed {
		t.Fatal("one-way follower saw mutuals profile", err)
	}
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, owner.ID, viewer.ID, socialIntent(t, domainuser.InteractionFollow, true)); err != nil {
		t.Fatal(err)
	}
	allowed, err = profiles.CanViewUserProfile(ctx, owner.ID, viewer.ID)
	if err != nil || !allowed {
		t.Fatal("mutual follower could not see profile", err)
	}
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, viewer.ID, owner.ID, socialIntent(t, domainuser.InteractionFollow, false)); err != nil {
		t.Fatal(err)
	}
	allowed, err = profiles.CanViewUserProfile(ctx, owner.ID, viewer.ID)
	if err != nil || allowed {
		t.Fatal("unfollow did not revoke profile", err)
	}
}

func TestConcurrentFriendRequestsAndBlockDoNotRestoreFriendshipIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	first, second := privatePhotoTestUsers(t, db)
	ctx := context.Background()
	node, _ := helpers.NewDefaultNode()
	friends := NewFriendshipRepository(db, node)
	var wg sync.WaitGroup
	ids := make(chan int64, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			row, err := friends.RequestFriendship(ctx, first.ID, second.PublicID)
			ids <- row.PublicID
			errs <- err
		})
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	requestID := int64(0)
	for id := range ids {
		if requestID == 0 {
			requestID = id
		}
		if id != requestID {
			t.Fatal("duplicate requests for one pair")
		}
	}
	var count int64
	if err := db.Model(&notifications.Notification{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("request notifications duplicated=%d %v", count, err)
	}
	results := make(chan error, 2)
	wg.Go(func() {
		_, err := friends.ChangeFriendship(ctx, second.ID, requestID, domainuser.FriendshipAccepted)
		results <- err
	})
	wg.Go(func() {
		_, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(ctx, first.ID, second.ID, socialIntent(t, domainuser.InteractionBlock, true))
		results <- err
	})
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil && !errors.Is(err, domainuser.ErrInteractionBlocked) {
			t.Fatal(err)
		}
	}
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(ctx, first.ID, second.ID, socialIntent(t, domainuser.InteractionBlock, false)); err != nil {
		t.Fatal(err)
	}
	rows, _, err := friends.ListFriendships(ctx, second.ID, domainuser.FriendshipScopeFriends, nil, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("friendship resurrected=%+v %v", rows, err)
	}
}
