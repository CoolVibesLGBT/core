package repositories

import (
	"core/application/ports"
	"core/application/types"
	"core/constants"
	"core/models"
	"core/models/post"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

func TestPublicPostReaderContractReturnsPersistenceFreeProjection(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	if !db.Migrator().HasTable(&post.Post{}) {
		t.Skip("post schema is not migrated in TEST_DATABASE_URL")
	}

	authorID := uuid.New()
	author := models.User{
		ID: authorID, PublicID: time.Now().UnixNano(), Domain: models.CoolVibes,
		UserName: "projection-author-" + uuid.NewString(), DisplayName: "Projection Author",
		Email: "private@example.test", Balance: decimal.NewFromInt(777), PreferencesFlags: "private-flags",
		UserRole: constants.UserRoleAdmin, BroadcastInfo: datatypes.JSON([]byte(`{"token":"private-broadcast"}`)),
	}
	if err := db.Omit(clause.Associations).Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	item := post.Post{
		ID: uuid.New(), PublicID: time.Now().UnixNano() + 1, AuthorID: author.ID,
		PostKind: post.PostKindPost, Domain: models.CoolVibes, Published: true,
	}
	if err := db.Omit(clause.Associations).Create(&item).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	repo := &PostRepository{db: db}
	var reader ports.PublicPostReader = repo
	projected, err := reader.FindPublicPostByPublicID(item.PublicID)
	if err != nil {
		t.Fatalf("FindPublicPostByPublicID() error = %v", err)
	}
	if projected.PublicID != types.SnowflakeID(item.PublicID) || projected.Author.PublicID != types.SnowflakeID(author.PublicID) {
		t.Fatalf("unexpected projection ids: %#v", projected)
	}
	encoded, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	payload := string(encoded)
	for _, forbidden := range []string{
		item.ID.String(), author.ID.String(), "private@example.test", "private-flags", "private-broadcast",
		`"balance"`, `"user_role"`, `"preferences_flags"`, `"broadcast_info"`,
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("repository projection leaked %q: %s", forbidden, payload)
		}
	}
}

var _ ports.PublicPostReader = (*PostRepository)(nil)
