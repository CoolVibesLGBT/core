package repositories

import (
	"context"
	"core/application/types"
	"core/application/usecases"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func socialGraphIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := privatePhotoIntegrationDB(t)
	if err := db.Exec(`CREATE TABLE locations (id uuid PRIMARY KEY, contentable_id uuid,
		contentable_type text, display text, city text, region text, country text, latitude double precision, longitude double precision, deleted_at timestamptz)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX social_test_user_aggregate ON engagements (contentable_type, contentable_id)`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func socialIntent(t *testing.T, kind domainuser.InteractionKind, enabled bool) domainuser.InteractionStateIntent {
	t.Helper()
	intent, err := domainuser.NewSetInteractionState(kind, enabled)
	if err != nil {
		t.Fatal(err)
	}
	return intent
}

func TestSocialProfileReactionsKeepMatchesAndOppositeReactionConsistentIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	repo := NewEngagementRepository(db)
	ctx := context.Background()
	like := socialIntent(t, domainuser.InteractionLike, true)
	for _, pair := range [][2]uuid.UUID{{actor.ID, target.ID}, {target.ID, actor.ID}} {
		if _, err := repo.ApplyReciprocalUserInteraction(ctx, pair[0], pair[1], like); err != nil {
			t.Fatal(err)
		}
	}
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindMatched, 1)
	assertLogicalMatchDetailCount(t, db, target.ID, actor.ID, models.EngagementKindMatched, 1)
	dislike, _ := domainuser.NewToggleReactionState(false)
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, actor.ID, target.ID, dislike); err != nil {
		t.Fatal(err)
	}
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindLikeGiven, 0)
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindDislikeGiven, 1)
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindMatched, 0)
	assertLogicalMatchDetailCount(t, db, target.ID, actor.ID, models.EngagementKindMatched, 0)
	assertUserEngagementCount(t, db, actor.ID, "like_given_count", 0)
	assertUserEngagementCount(t, db, actor.ID, "dislike_given_count", 1)
	assertUserEngagementCount(t, db, actor.ID, "match_count", 0)
}

func TestSocialProfileUnlikeRemovesSwipeMatchIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	matches := NewMatchesRepository(db, nil)
	ctx := context.Background()
	for _, pair := range [][2]uuid.UUID{{actor.ID, target.ID}, {target.ID, actor.ID}} {
		if _, err := matches.RecordView(ctx, pair[0], pair[1], domainuser.MatchReactionLike); err != nil {
			t.Fatal(err)
		}
	}
	unlike := socialIntent(t, domainuser.InteractionLike, false)
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(ctx, actor.ID, target.ID, unlike); err != nil {
		t.Fatal(err)
	}
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindMatched, 0)
	assertLogicalMatchDetailCount(t, db, target.ID, actor.ID, models.EngagementKindMatched, 0)
}

func TestSocialBlockRemovesRelationshipsAndDoesNotRestoreOnUnblockIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	repo := NewEngagementRepository(db)
	ctx := context.Background()
	for _, pair := range [][2]uuid.UUID{{actor.ID, target.ID}, {target.ID, actor.ID}} {
		for _, kind := range []domainuser.InteractionKind{domainuser.InteractionFollow, domainuser.InteractionSubscribe} {
			if _, err := repo.ApplyReciprocalUserInteraction(ctx, pair[0], pair[1], socialIntent(t, kind, true)); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := NewMatchesRepository(db, nil).RecordView(ctx, pair[0], pair[1], domainuser.MatchReactionLike); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, target.ID, actor.ID, socialIntent(t, domainuser.InteractionBlock, true)); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]uuid.UUID{{actor.ID, target.ID}, {target.ID, actor.ID}} {
		for _, kind := range []domainuser.InteractionKind{domainuser.InteractionFollow, domainuser.InteractionSubscribe, domainuser.InteractionLike} {
			if _, err := repo.ApplyReciprocalUserInteraction(ctx, pair[0], pair[1], socialIntent(t, kind, true)); err == nil {
				t.Fatalf("blocked interaction %s was accepted", kind)
			}
		}
	}
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, target.ID, actor.ID, socialIntent(t, domainuser.InteractionBlock, false)); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]uuid.UUID{{actor.ID, target.ID}, {target.ID, actor.ID}} {
		for _, kind := range []models.EngagementKind{models.EngagementKindFollowing, models.EngagementKindFollower,
			models.EngagementKindLikeGiven, models.EngagementKindLikeReceived, models.EngagementKindMatched,
			models.EngagementKindSubscribing, models.EngagementKindSubscribedBy} {
			assertLogicalMatchDetailCount(t, db, pair[0], pair[1], kind, 0)
		}
		for _, key := range []string{"following_count", "follower_count", "like_given_count", "like_received_count", "match_count", "subscribing_count", "subscribed_by_count", "blocking_count", "blocked_by_count"} {
			assertUserEngagementCount(t, db, pair[0], key, 0)
		}
	}
}

func TestSocialFollowRejectsUnavailableOrForeignDomainUsersIntegration(t *testing.T) {
	for _, state := range []string{"banned", "deleted", "pending", "bot", "different domain"} {
		t.Run(state, func(t *testing.T) {
			db := socialGraphIntegrationDB(t)
			actor, target := privatePhotoTestUsers(t, db)
			changes := map[string]any{"user_role": state}
			if state == "bot" {
				changes = map[string]any{"is_bot": true}
			} else if state == "different domain" {
				changes = map[string]any{"domain": "another-domain"}
			}
			if err := db.Model(&target).Updates(changes).Error; err != nil {
				t.Fatal(err)
			}
			_, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, socialIntent(t, domainuser.InteractionFollow, true))
			if err == nil {
				t.Fatalf("follow accepted unavailable target %s", state)
			}
		})
	}
}

func TestSocialUnseenUsesMostRecentSwipeTimeIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	matches := NewMatchesRepository(db, nil)
	ctx := context.Background()
	if _, err := matches.RecordView(ctx, actor.ID, target.ID, domainuser.MatchReactionDislike); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&models.EngagementDetail{}).
		Where("engager_id = ? AND kind = ?", actor.ID, models.EngagementKindViewGiven).
		Update("created_at", time.Now().Add(-48*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := matches.RecordView(ctx, actor.ID, target.ID, domainuser.MatchReactionDislike); err != nil {
		t.Fatal(err)
	}
	unseen, err := matches.GetUnseenUsers(ctx, actor.ID, 10)
	if err != nil || len(unseen) != 0 {
		t.Fatalf("recently swiped target reappeared = %+v, %v", unseen, err)
	}
}

func TestSocialConcurrentFollowIsIdempotentIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	repo := NewEngagementRepository(db)
	intent := socialIntent(t, domainuser.InteractionFollow, true)
	results := make(chan domainuser.InteractionStateTransition, 10)
	errorsCh := make(chan error, 10)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Go(func() {
			result, err := repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, intent)
			results <- result
			errorsCh <- err
		})
	}
	wg.Wait()
	close(results)
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	changed := 0
	for result := range results {
		if !result.Enabled {
			t.Fatal("follow unexpectedly disabled")
		}
		if result.Changed {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("follow transitions = %d; want 1", changed)
	}
	assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindFollowing, 1)
	assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindFollower, 1)
}

func TestSocialFollowPaginationDoesNotSkipEqualTimestampsIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, _ := privatePhotoTestUsers(t, db)
	repo := NewEngagementRepository(db)
	for i := 0; i < 5; i++ {
		target := models.User{ID: uuid.New(), PublicID: int64(100 + i)}
		if err := db.Omit("Avatar", "Cover", "Wallet", "Location", "Engagements").Create(&target).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := repo.ApplyReciprocalUserInteraction(context.Background(), actor.ID, target.ID, socialIntent(t, domainuser.InteractionFollow, true)); err != nil {
			t.Fatal(err)
		}
	}
	fixed := time.Date(2026, 10, 21, 0, 0, 0, 123000000, time.UTC)
	if err := db.Model(&models.EngagementDetail{}).Where("engager_id = ? AND kind = ?", actor.ID, models.EngagementKindFollowing).Update("created_at", fixed).Error; err != nil {
		t.Fatal(err)
	}
	var cursor *types.MatchListCursor
	seen := make(map[uuid.UUID]bool)
	for page := 0; page < 3; page++ {
		rows, next, err := repo.GetEngagements(context.Background(), models.EngagementContentableTypeUser, actor.ID, models.EngagementKindFollowing, cursor, 2)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			if seen[row.ID] {
				t.Fatal("duplicate follower page item")
			}
			seen[row.ID] = true
		}
		if page < 2 && (next == nil || next.DetailID == uuid.Nil) {
			t.Fatal("missing exact continuation cursor")
		}
		if page == 2 && next != nil {
			t.Fatal("last page returned a redundant cursor")
		}
		cursor = next
	}
	if len(seen) != 5 {
		t.Fatalf("paginated follows = %d; want 5", len(seen))
	}
}

func TestSocialEmptyFollowListIsSuccessfulIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	owner, _ := privatePhotoTestUsers(t, db)
	rows, next, err := NewEngagementRepository(db).GetEngagements(context.Background(), models.EngagementContentableTypeUser, owner.ID, models.EngagementKindFollower, nil, 10)
	if err != nil || len(rows) != 0 || next != nil {
		t.Fatalf("empty follower list = %+v, %+v, %v", rows, next, err)
	}
}

func TestSocialMatchedUsersAreNotRediscoveredAfter24HoursIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	matches := NewMatchesRepository(db, nil)
	ctx := context.Background()
	for _, pair := range [][2]uuid.UUID{{actor.ID, target.ID}, {target.ID, actor.ID}} {
		if _, err := matches.RecordView(ctx, pair[0], pair[1], domainuser.MatchReactionLike); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&models.EngagementDetail{}).Where("kind = ?", models.EngagementKindViewGiven).
		UpdateColumns(map[string]any{"created_at": time.Now().Add(-48 * time.Hour), "updated_at": time.Now().Add(-48 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	unseen, err := matches.GetUnseenUsers(ctx, actor.ID, 10)
	if err != nil || len(unseen) != 0 {
		t.Fatalf("already matched user rediscovered = %+v, %v", unseen, err)
	}
}

func TestSocialConcurrentProfileAndSwipeReactionsShareCanonicalStateIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	matches := NewMatchesRepository(db, nil)
	engagements := NewEngagementRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := matches.RecordView(ctx, target.ID, actor.ID, domainuser.MatchReactionLike); err != nil {
		t.Fatal(err)
	}
	intent := socialIntent(t, domainuser.InteractionLike, true)
	errorsCh := make(chan error, 8)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Go(func() {
			var err error
			if i%2 == 0 {
				_, err = matches.RecordView(ctx, actor.ID, target.ID, domainuser.MatchReactionLike)
			} else {
				_, err = engagements.ApplyReciprocalUserInteraction(ctx, actor.ID, target.ID, intent)
			}
			errorsCh <- err
		})
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, pair := range [][2]uuid.UUID{{actor.ID, target.ID}, {target.ID, actor.ID}} {
		assertLogicalMatchDetailCount(t, db, pair[0], pair[1], models.EngagementKindLikeGiven, 1)
		assertLogicalMatchDetailCount(t, db, pair[0], pair[1], models.EngagementKindMatched, 1)
		assertUserEngagementCount(t, db, pair[0], "like_given_count", 1)
		assertUserEngagementCount(t, db, pair[0], "match_count", 1)
	}
}

func TestSocialConcurrentFollowAndBlockCannotReconnectUsersIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	actor, target := privatePhotoTestUsers(t, db)
	repo := NewEngagementRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	follow := socialIntent(t, domainuser.InteractionFollow, true)
	block := socialIntent(t, domainuser.InteractionBlock, true)
	unblock := socialIntent(t, domainuser.InteractionBlock, false)
	for attempt := 0; attempt < 10; attempt++ {
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for _, action := range []struct {
			actor, target uuid.UUID
			intent        domainuser.InteractionStateIntent
		}{
			{actor.ID, target.ID, follow}, {target.ID, actor.ID, block},
		} {
			wg.Go(func() {
				_, err := repo.ApplyReciprocalUserInteraction(ctx, action.actor, action.target, action.intent)
				results <- err
			})
		}
		wg.Wait()
		close(results)
		for err := range results {
			if err != nil && !errors.Is(err, domainuser.ErrInteractionBlocked) {
				t.Fatal(err)
			}
		}
		if _, err := repo.ApplyReciprocalUserInteraction(ctx, target.ID, actor.ID, unblock); err != nil {
			t.Fatal(err)
		}
		assertReciprocalDetailCount(t, db, actor.ID, target.ID, models.EngagementKindFollowing, 0)
		assertReciprocalDetailCount(t, db, target.ID, actor.ID, models.EngagementKindFollower, 0)
	}
}

func TestSocialPublicFollowListDoesNotRevealBlockedParticipantsIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	hidden, viewer := privatePhotoTestUsers(t, db)
	viewer.Domain = models.CoolVibes
	owner := models.User{ID: uuid.New(), PublicID: 30, Domain: models.CoolVibes, PrivacyLevel: constants.PrivacyPublic}
	if err := db.Omit("Avatar", "Cover", "Wallet", "Location", "Engagements").Create(&owner).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewEngagementRepository(db)
	ctx := context.Background()
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, owner.ID, hidden.ID, socialIntent(t, domainuser.InteractionFollow, true)); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ApplyReciprocalUserInteraction(ctx, viewer.ID, hidden.ID, socialIntent(t, domainuser.InteractionBlock, true)); err != nil {
		t.Fatal(err)
	}
	service := usecases.NewUserService(&UserRepository{db: db}, nil, nil, repo, nil)
	rows, _, err := service.FetchUserEngagements(ctx, &viewer, owner.ID, models.EngagementContentableTypeUser, models.EngagementKindFollowing, nil, 10)
	if err != nil || len(rows) != 0 {
		t.Fatalf("blocked participant visible through another user's follow list = %+v, %v", rows, err)
	}
}
