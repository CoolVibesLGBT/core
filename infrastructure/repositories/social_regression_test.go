package repositories

import (
	"context"
	"core/application/types"
	domainuser "core/domain/user"
	"core/models"
	"core/models/notifications"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestProfileAndSwipeShareSingleViewIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	engagements := NewEngagementRepository(db)
	if _, err := engagements.RecordViewOnce(context.Background(), actor.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser); err != nil {
		t.Fatal(err)
	}
	if _, err := NewMatchesRepository(db, nil).RecordView(context.Background(), actor.ID, target.ID, domainuser.MatchReactionLike); err != nil {
		t.Fatal(err)
	}
	assertUserEngagementCount(t, db, target.ID, "view_received_count", 1)
}

// A swipe (refresh=true) bumps the shared view's last-seen time so the
// 24-hour discovery exclusion stays accurate; a plain profile revisit
// (refresh=false) must not disturb that timestamp or double-count the view.
func TestProfileViewReplayPreservesSwipeTimestampIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	ctx := context.Background()
	engagements := NewEngagementRepository(db)
	if _, err := NewMatchesRepository(db, nil).RecordView(ctx, actor.ID, target.ID, domainuser.MatchReactionDislike); err != nil {
		t.Fatal(err)
	}
	var before models.EngagementDetail
	if err := db.Where("engager_id = ? AND engagee_id = ? AND kind = ?", actor.ID, target.ID, models.EngagementKindViewGiven).First(&before).Error; err != nil {
		t.Fatal(err)
	}
	counted, err := engagements.RecordViewOnce(ctx, actor.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser)
	if err != nil || counted {
		t.Fatalf("profile replay: counted=%v err=%v", counted, err)
	}
	var after models.EngagementDetail
	if err := db.First(&after, "id = ?", before.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("profile replay changed the swipe's last-seen time")
	}
	assertUserEngagementCount(t, db, target.ID, "view_received_count", 1)
}

func socialRegressionPost(t *testing.T, db *gorm.DB, author uuid.UUID) (*PostRepository, int64) {
	t.Helper()
	if err := db.Exec("ALTER TABLE posts ADD COLUMN public_id bigint").Error; err != nil {
		t.Fatal(err)
	}
	id := int64(101)
	if err := db.Table("posts").Create(map[string]any{"id": uuid.New(), "public_id": id, "author_id": author, "post_kind": "post", "published": true, "audience": "public"}).Error; err != nil {
		t.Fatal(err)
	}
	nr := NewNotificationRepository(db, nil)
	ur := NewUserRepository(db, nil, nil, NewEngagementRepository(db), nr)
	return NewPostRepository(db, nil, nil, ur, nr), id
}

func TestPostLikeUsesLikeNotificationTypeIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, viewer := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	if err := repo.Like(types.Filter{Context: context.Background(), AuthUser: &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID}, PostID: id}); err != nil {
		t.Fatal(err)
	}
	var items []notifications.Notification
	if err := db.Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("notifications=%d", len(items))
	}
	for _, item := range items {
		if item.Type != notifications.NotificationTypeLike {
			t.Errorf("post like notification type=%q, want like", item.Type)
		}
	}
}

func TestPostReactionsAreExclusiveIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, viewer := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	filter := types.Filter{Context: context.Background(), AuthUser: &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID}, PostID: id}
	if err := repo.Like(filter); err != nil {
		t.Fatal(err)
	}
	if err := repo.Dislike(filter); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&models.EngagementDetail{}).Where("engager_id = ? AND kind IN ?", viewer.ID, []models.EngagementKind{models.EngagementKindLikeReceived, models.EngagementKindDisLikeReceived}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("simultaneous like/dislike rows=%d, want 1", count)
	}
}

func TestPostNotificationFailureRollsBackReactionIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, viewer := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	if err := db.Exec("DROP TABLE notification_deliveries").Error; err != nil {
		t.Fatal(err)
	}
	err := repo.Like(types.Filter{Context: context.Background(), AuthUser: &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID}, PostID: id})
	var count int64
	if e := db.Model(&models.EngagementDetail{}).Where("kind = ?", models.EngagementKindLikeReceived).Count(&count).Error; e != nil {
		t.Fatal(e)
	}
	if err == nil || count != 0 {
		t.Fatalf("notification failure left committed reaction: error=%v rows=%d", err, count)
	}
}

func TestPostLookupKeepsRequestContextIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, viewer := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	queries := 0
	if err := db.Callback().Query().Before("gorm:query").Register("social-regression:context", func(tx *gorm.DB) {
		if tx.Statement.Table == "posts" {
			queries++
			if tx.Statement.Context.Value(ctxKey{}) != "request" {
				t.Error("post lookup dropped request context")
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.View(types.Filter{Context: ctx, AuthUser: &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID}, PostID: id}); err != nil {
		t.Fatal(err)
	}
	if queries == 0 {
		t.Fatal("no post query checked")
	}
}

func TestProfileViewChecksBlockIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	repo := NewEngagementRepository(db)
	if _, err := repo.ApplyReciprocalUserInteraction(context.Background(), target.ID, actor.ID, socialIntent(t, "block", true)); err != nil {
		t.Fatal(err)
	}
	counted, err := repo.RecordViewOnce(context.Background(), actor.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser)
	if !errors.Is(err, domainuser.ErrInteractionBlocked) || counted {
		t.Fatalf("blocked profile view: counted=%v err=%v", counted, err)
	}
}

func TestFollowDirectionCountsAndCancellationIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	repo := NewEngagementRepository(db)
	for i := 0; i < 3; i++ {
		if _, err := repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, socialIntent(t, "follow", true)); err != nil {
			t.Fatal(err)
		}
	}
	assertUserEngagementCount(t, db, actor.ID, "following_count", 1)
	assertUserEngagementCount(t, db, target.ID, "follower_count", 1)
	assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindFollowing, 1)
	assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindFollower, 1)
	var count int64
	if err := db.Model(&notifications.Notification{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("follow replay duplicated notifications=%d", count)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, actor.ID, target.ID, socialIntent(t, "follow", false)); err == nil {
		t.Fatal("cancelled context was ignored")
	}
	assertUserEngagementCount(t, db, actor.ID, "following_count", 1)
}

func TestConcurrentNotificationFetchDoesNotShowTwiceIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	sender, receiver := privatePhotoTestUsers(t, db)
	repo := NewNotificationRepository(db, nil)
	if _, err := repo.CreateNotification(context.Background(), sender.ID, receiver.ID, "test", "Title", "Body", notifications.NotificationPayload{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	if err := db.Callback().Query().After("gorm:query").Register("social-regression:select-barrier", func(tx *gorm.DB) {
		if tx.Statement.Table == "notifications" {
			ready <- struct{}{}
			select {
			case <-release:
			case <-ctx.Done():
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	type result struct {
		count int
		err   error
	}
	results := make(chan result, 2)
	for i := 0; i < 2; i++ {
		go func() {
			items, err := repo.FetchAndMarkShownNotifications(ctx, receiver.ID, 1)
			results <- result{len(items), err}
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatal("concurrent notification claims did not finish")
		}
	}
	close(release)
	first, second := <-results, <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("fetch errors: %v %v", first.err, second.err)
	}
	if got := first.count + second.count; got != 1 {
		t.Fatalf("same notification shown %d times across concurrent polls, want 1", got)
	}
}

func TestBlockedUserCannotReactToPostIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, viewer := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(context.Background(), author.ID, viewer.ID, socialIntent(t, "block", true)); err != nil {
		t.Fatal(err)
	}
	err := repo.Like(types.Filter{Context: context.Background(), AuthUser: &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID}, PostID: id})
	var count int64
	if e := db.Model(&models.EngagementDetail{}).Where("kind = ? AND engager_id = ?", models.EngagementKindLikeReceived, viewer.ID).Count(&count).Error; e != nil {
		t.Fatal(e)
	}
	if !errors.Is(err, domainuser.ErrInteractionBlocked) || count != 0 {
		t.Fatalf("blocked post reaction: rows=%d err=%v", count, err)
	}
}

func TestConcurrentProfileAndSwipeViewsRemainUniqueIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	engagements, matches := NewEngagementRepository(db), NewMatchesRepository(db, nil)
	var wg sync.WaitGroup
	results := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Go(func() {
			var err error
			if i%2 == 0 {
				_, err = engagements.RecordViewOnce(context.Background(), actor.ID, target.ID, models.EngagementKindViewReceived, target.ID, models.EngagementContentableTypeUser)
			} else {
				_, err = matches.RecordView(context.Background(), actor.ID, target.ID, domainuser.MatchReactionLike)
			}
			results <- err
		})
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	assertUserEngagementCount(t, db, actor.ID, "view_given_count", 1)
	assertUserEngagementCount(t, db, target.ID, "view_received_count", 1)
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindViewGiven, 1)
	assertLogicalMatchDetailCount(t, db, target.ID, actor.ID, models.EngagementKindViewReceived, 1)
}

func TestConcurrentPostReactionsKeepExclusiveRowsAndExactCountersIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, viewer := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	filters := types.Filter{Context: context.Background(), AuthUser: &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID}, PostID: id}
	var wg sync.WaitGroup
	results := make(chan error, 15)
	for i := 0; i < 15; i++ {
		wg.Go(func() {
			if i%2 == 0 {
				results <- repo.Like(filters)
			} else {
				results <- repo.Dislike(filters)
			}
		})
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	var aggregate models.Engagement
	if err := db.Where("contentable_type = ?", "post").First(&aggregate).Error; err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, kind := range []models.EngagementKind{models.EngagementKindLikeReceived, models.EngagementKindDisLikeReceived} {
		var count int64
		if err := db.Model(&models.EngagementDetail{}).Where("engagement_id = ? AND kind = ?", aggregate.ID, kind).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		var counts map[string]int64
		if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
			t.Fatal(err)
		}
		if counts[models.EngagementCountKeys[kind].CountKey] != count {
			t.Fatalf("%s counter does not match stored rows", kind)
		}
		total += count
	}
	if total > 1 {
		t.Fatalf("concurrent reactions left %d rows", total)
	}
	var jobs int64
	if err := db.Model(&notifications.Delivery{}).Count(&jobs).Error; err != nil || jobs != 30 {
		t.Fatalf("notification jobs=%d err=%v", jobs, err)
	}
}

func TestAllPostInteractionNotificationsHaveSpecificTypesIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, viewer := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	filters := types.Filter{Context: context.Background(), AuthUser: &types.Actor{ID: viewer.ID, PublicID: viewer.PublicID}, PostID: id}
	for _, operation := range []struct {
		run           func(types.Filter) error
		kind, removed string
	}{
		{repo.Like, notifications.NotificationTypeLike, notifications.NotificationTypeUnlike},
		{repo.Dislike, notifications.NotificationTypeDislike, notifications.NotificationTypeUndislike},
		{repo.Banana, notifications.NotificationTypeBanana, notifications.NotificationTypeBananaRemoved},
		{repo.Bookmark, notifications.NotificationTypeBookmark, notifications.NotificationTypeUnbookmark},
	} {
		for _, kind := range []string{operation.kind, operation.removed} {
			if err := operation.run(filters); err != nil {
				t.Fatal(err)
			}
			var items []notifications.Notification
			if err := db.Where("type = ?", kind).Find(&items).Error; err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Fatalf("%s notifications=%d", kind, len(items))
			}
			for _, item := range items {
				data, ok := item.Payload.Data.(map[string]any)
				if !ok || data["post_id"] != "101" {
					t.Fatalf("notification missing public post identity: %+v", item.Payload)
				}
			}
		}
	}
}

func TestNotificationAndIdentityOperationsRespectCancelledContextIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	nr := NewNotificationRepository(db, nil)
	ur := NewUserRepository(db, nil, nil, NewEngagementRepository(db), nr)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ur.GetUserUUIDByPublicID(ctx, target.PublicID); !errors.Is(err, context.Canceled) {
		t.Fatalf("identity lookup cancellation=%v", err)
	}
	if _, err := nr.CreateNotification(ctx, actor.ID, target.ID, "test", "Title", "Body", notifications.NotificationPayload{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("notification write cancellation=%v", err)
	}
	item, err := nr.CreateNotification(context.Background(), actor.ID, target.ID, "test", "Title", "Body", notifications.NotificationPayload{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := nr.FetchAndMarkShownNotifications(ctx, target.ID, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("notification fetch cancellation=%v", err)
	}
	if err := db.First(item, "id = ?", item.ID).Error; err != nil || item.IsShown {
		t.Fatalf("cancelled fetch acknowledged item: %v %v", item.IsShown, err)
	}
}

func TestPostOwnerGetsOneInteractionNotificationIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	author, _ := privatePhotoTestUsers(t, db)
	repo, id := socialRegressionPost(t, db, author.ID)
	if err := repo.Like(types.Filter{Context: context.Background(), AuthUser: &types.Actor{ID: author.ID, PublicID: author.PublicID}, PostID: id}); err != nil {
		t.Fatal(err)
	}
	var items []notifications.Notification
	if err := db.Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Title != "Liked" {
		t.Fatalf("self interaction notifications=%+v", items)
	}
}
