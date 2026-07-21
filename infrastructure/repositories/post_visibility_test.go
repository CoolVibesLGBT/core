package repositories

import (
	"core/constants"
	"core/models"
	"core/models/post"
	"core/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func TestPublicPostReadsExcludeHiddenPosts(t *testing.T) {
	db := eventRSVPIntegrationDB(t)
	if !db.Migrator().HasTable(&post.Post{}) {
		t.Skip("post schema is not migrated in TEST_DATABASE_URL")
	}
	author := models.User{
		ID: uuid.New(), PublicID: time.Now().UnixNano(), Domain: models.CoolVibes,
		UserName: "hidden-author-" + uuid.NewString(), DisplayName: "Hidden Author", UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	hidden := post.Post{
		ID: uuid.New(), PublicID: time.Now().UnixNano() + 1, AuthorID: author.ID,
		PostKind: post.PostKindPost, Domain: models.CoolVibes, Published: false,
	}
	slug := "hidden-" + uuid.NewString()
	hidden.Slug = &slug
	if err := db.Omit(clause.Associations).Create(&hidden).Error; err != nil {
		t.Fatalf("create hidden post: %v", err)
	}
	privateAudience := "private"
	privatePost := post.Post{
		ID: uuid.New(), PublicID: time.Now().UnixNano() + 2, AuthorID: author.ID,
		PostKind: post.PostKindPost, Domain: models.CoolVibes, Published: true, Audience: &privateAudience,
	}
	messageContentType := string(post.PostKindChat)
	chatMessage := post.Post{
		ID: uuid.New(), PublicID: time.Now().UnixNano() + 3, AuthorID: author.ID,
		PostKind: post.PostKindMessage, Domain: models.CoolVibes, Published: true, ContentableType: &messageContentType,
	}
	if err := db.Omit(clause.Associations).Create(&privatePost).Error; err != nil {
		t.Fatalf("create private post: %v", err)
	}
	if err := db.Omit(clause.Associations).Create(&chatMessage).Error; err != nil {
		t.Fatalf("create chat message: %v", err)
	}

	repo := &PostRepository{db: db}
	if _, err := repo.GetPostByPublicID(hidden.PublicID); err == nil {
		t.Fatal("GetPostByPublicID returned a hidden post")
	}
	if _, err := repo.GetPostByID(hidden.ID); err == nil {
		t.Fatal("GetPostByID returned a hidden post")
	}
	if _, err := repo.GetPostBySlug(types.Filter{Slug: &slug}); err == nil {
		t.Fatal("GetPostBySlug returned a hidden post")
	}
	if _, err := repo.GetPostByPublicID(privatePost.PublicID); err == nil {
		t.Fatal("GetPostByPublicID returned a private-audience post")
	}
	if _, err := repo.GetPostByPublicID(chatMessage.PublicID); err == nil {
		t.Fatal("GetPostByPublicID returned a chat message")
	}
	if _, err := repo.FindPostByPublicID(privatePost.PublicID); err == nil {
		t.Fatal("FindPostByPublicID returned a private-audience post")
	}
	if _, err := repo.FindPostByPublicID(chatMessage.PublicID); err == nil {
		t.Fatal("FindPostByPublicID returned a chat message")
	}

	timeline, err := repo.GetTimeline(types.Filter{Limit: 100})
	if err != nil {
		t.Fatalf("GetTimeline() error = %v", err)
	}
	for _, item := range timeline.Posts {
		if item.ID == hidden.ID {
			t.Fatal("timeline contains hidden post")
		}
	}
	byKind, err := repo.GetPostsByKind(types.Filter{PostKind: post.PostKindPost, Limit: 100})
	if err != nil {
		t.Fatalf("GetPostsByKind() error = %v", err)
	}
	for _, item := range byKind.Posts {
		if item.ID == hidden.ID {
			t.Fatal("kind feed contains hidden post")
		}
	}
}
