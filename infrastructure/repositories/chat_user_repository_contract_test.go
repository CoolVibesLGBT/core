package repositories

import (
	"core/models"
	"core/models/post"
	"core/types"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChatMessagesByChatIDQueryFiltersDeletedMessagesAndClearTime(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &ChatRepository{db: db}
	userID := uuid.New()
	chatID := uuid.New()
	clearedAt := time.Now().UTC()

	var posts []post.Post
	tx := repo.messagesByChatIDQuery(userID, chatID, &clearedAt).
		Order("posts.created_at ASC").
		Find(&posts)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"posts.contentable_type",
		"posts.contentable_id",
		"LEFT JOIN engagements",
		"LEFT JOIN engagement_details",
		"ed.id IS NULL",
		"posts.created_at >",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
	vars := stringifyVars(tx.Statement.Vars)
	for _, value := range []string{"message_deleted_for_me", "message_deleted_for_all"} {
		if !strings.Contains(vars, value) {
			t.Fatalf("expected query vars to contain %q, got %s", value, vars)
		}
	}
}

func TestLiveUsersWithoutLocationQueryAppliesLiveCursorAndOrder(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &UserRepository{db: db}
	isLive := true
	cursor := int64(1001)

	var users []models.User
	tx := repo.liveUsersWithoutLocationQuery(types.Filter{IsLive: &isLive, Cursor: &cursor, Limit: 20}).
		Find(&users)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"FROM \"users\"",
		"is_live =",
		"public_id >",
		"ORDER BY public_id ASC",
		"LIMIT",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
}

func stringifyVars(values []interface{}) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, " ")
}
