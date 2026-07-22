package repositories

import (
	"context"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestRecordViewIsIdempotentExclusiveAndMatchAtomicIntegration(t *testing.T) {
	db := engagementViewIntegrationDB(t)
	if err := db.AutoMigrate(&models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate match engagement schema: %v", err)
	}

	actor, target := createReciprocalTestUsers(t, db)
	repository := NewMatchesRepository(db, nil)

	matched, err := repository.RecordView(context.Background(), actor.ID, target.ID, domainuser.MatchReactionLike)
	if err != nil || matched {
		t.Fatalf("first actor like = %v, %v; want unmatched", matched, err)
	}
	matched, err = repository.RecordView(context.Background(), actor.ID, target.ID, domainuser.MatchReactionLike)
	if err != nil || matched {
		t.Fatalf("replayed actor like = %v, %v; want unmatched", matched, err)
	}
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindLikeGiven, 1)
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindDislikeGiven, 0)
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindViewGiven, 1)

	matched, err = repository.RecordView(context.Background(), target.ID, actor.ID, domainuser.MatchReactionLike)
	if err != nil || !matched {
		t.Fatalf("reciprocal like = %v, %v; want matched", matched, err)
	}
	matched, err = repository.RecordView(context.Background(), target.ID, actor.ID, domainuser.MatchReactionLike)
	if err != nil || !matched {
		t.Fatalf("replayed reciprocal like = %v, %v; want matched", matched, err)
	}
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindMatched, 1)
	assertLogicalMatchDetailCount(t, db, target.ID, actor.ID, models.EngagementKindMatched, 1)

	matched, err = repository.RecordView(context.Background(), actor.ID, target.ID, domainuser.MatchReactionDislike)
	if err != nil || matched {
		t.Fatalf("actor dislike after match = %v, %v; want unmatched", matched, err)
	}
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindLikeGiven, 0)
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindDislikeGiven, 1)
	assertLogicalMatchDetailCount(t, db, actor.ID, target.ID, models.EngagementKindMatched, 0)
	assertLogicalMatchDetailCount(t, db, target.ID, actor.ID, models.EngagementKindMatched, 0)
}

func TestMatchListsUseExactEngagementCursorAndPublicProjectionIntegration(t *testing.T) {
	db := engagementViewIntegrationDB(t)
	if err := db.AutoMigrate(&models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate match list schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	users := make([]models.User, 6)
	for index := range users {
		users[index] = models.User{
			ID:           uuid.New(),
			PublicID:     basePublicID + int64(index),
			Domain:       models.CoolVibes,
			UserName:     "match-list-" + uuid.NewString(),
			DisplayName:  "Match List",
			UserRole:     constants.UserRoleUser,
			PrivacyLevel: constants.PrivacyPublic,
		}
	}
	if err := db.Omit(clause.Associations).Create(&users).Error; err != nil {
		t.Fatalf("create match list users: %v", err)
	}

	repository := NewMatchesRepository(db, nil)
	actor := users[0]
	for _, target := range users[1:] {
		if matched, err := repository.RecordView(context.Background(), actor.ID, target.ID, domainuser.MatchReactionLike); err != nil || matched {
			t.Fatalf("record actor like for %s = %v, %v", target.ID, matched, err)
		}
	}

	// Relationships to private and blocked targets may exist historically, but
	// neither target may be returned by the public matching read model.
	if err := db.Model(&models.User{}).Where("id = ?", users[4].ID).Update("privacy_level", constants.PrivacyPrivate).Error; err != nil {
		t.Fatalf("make target private: %v", err)
	}
	blockIntent, err := domainuser.NewSetInteractionState(domainuser.InteractionBlock, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(context.Background(), users[5].ID, actor.ID, blockIntent); err != nil {
		t.Fatalf("block actor: %v", err)
	}

	// Equal timestamps exercise the UUID tie-breaker. Pagination must return
	// every visible row exactly once.
	fixedTime := time.Date(2026, time.July, 22, 1, 2, 3, 456000000, time.UTC)
	if err := db.Model(&models.EngagementDetail{}).
		Where("engager_id = ? AND kind = ?", actor.ID, models.EngagementKindLikeGiven).
		Update("created_at", fixedTime).Error; err != nil {
		t.Fatalf("align relationship timestamps: %v", err)
	}

	first, err := repository.GetLikesAfter(context.Background(), actor.ID, nil, 2)
	if err != nil {
		t.Fatalf("first likes page: %v", err)
	}
	if len(first.Users) != 2 || first.Next == nil || first.Next.DetailID == uuid.Nil || !first.Next.OccurredAt.Equal(fixedTime) {
		t.Fatalf("first page = %+v; want 2 users and exact next cursor", first)
	}
	second, err := repository.GetLikesAfter(context.Background(), actor.ID, first.Next, 2)
	if err != nil {
		t.Fatalf("second likes page: %v", err)
	}
	if len(second.Users) != 1 || second.Next != nil {
		t.Fatalf("second page = %+v; want final single user", second)
	}

	seen := make(map[string]struct{})
	for _, user := range append(first.Users, second.Users...) {
		id := strconv.FormatInt(int64(user.PublicID), 10)
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("duplicate public user %s across cursor pages", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != 3 {
		t.Fatalf("visible paginated users = %d; want 3", len(seen))
	}

	payload, err := json.Marshal(append(first.Users, second.Users...))
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(payload)
	for _, forbidden := range []string{"balance", "user_role", "preferences_flags", "broadcast_info", "storage_path", actor.ID.String()} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("public match list leaks %q: %s", forbidden, serialized)
		}
	}
}

func assertLogicalMatchDetailCount(t *testing.T, db *gorm.DB, engagerID, engageeID uuid.UUID, kind models.EngagementKind, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(&models.EngagementDetail{}).
		Joins("JOIN engagements AS match_aggregate ON match_aggregate.id = engagement_details.engagement_id").
		Where("match_aggregate.contentable_type = ?", models.EngagementContentableTypeUser).
		Where("engagement_details.engager_id = ? AND engagement_details.engagee_id = ? AND engagement_details.kind = ?", engagerID, engageeID, kind).
		Count(&count).Error; err != nil {
		t.Fatalf("count %s relationship: %v", kind, err)
	}
	if count != expected {
		t.Fatalf("%s relationship %s -> %s count = %d; want %d", kind, engagerID, engageeID, count, expected)
	}
}
