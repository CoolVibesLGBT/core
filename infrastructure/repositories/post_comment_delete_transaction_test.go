package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"core/application/ports"
	"core/application/types"
	"core/constants"
	"core/helpers"
	"core/models"
	"core/models/post"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type commentDeleteFixture struct {
	db        *gorm.DB
	repo      *PostRepository
	author    models.User
	parent    post.Post
	comments  []post.Post
	aggregate models.Engagement
	details   []models.EngagementDetail
}

func prepareCommentDeleteFixture(t *testing.T, commentCount int, includeBookmark bool) commentDeleteFixture {
	t.Helper()
	db := locationRaceIntegrationDB(t)
	if err := db.AutoMigrate(&models.User{}, &post.Post{}, &models.Engagement{}, &models.EngagementDetail{}); err != nil {
		t.Fatalf("migrate comment deletion schema: %v", err)
	}

	basePublicID := time.Now().UTC().UnixNano()
	parentAuthor := models.User{
		ID: uuid.New(), PublicID: basePublicID, Domain: models.CoolVibes,
		UserName: "comment-parent-" + uuid.NewString(), DisplayName: "Parent author",
		UserRole: constants.UserRoleUser,
	}
	commentAuthor := models.User{
		ID: uuid.New(), PublicID: basePublicID + 1, Domain: models.CoolVibes,
		UserName: "comment-author-" + uuid.NewString(), DisplayName: "Comment author",
		UserRole: constants.UserRoleUser,
	}
	if err := db.Omit(clause.Associations).Create(&[]models.User{parentAuthor, commentAuthor}).Error; err != nil {
		t.Fatalf("create comment deletion users: %v", err)
	}

	audience := "public"
	parent := post.Post{
		ID: uuid.New(), PublicID: basePublicID + 2, AuthorID: parentAuthor.ID,
		PostKind: post.PostKindPost, Domain: models.CoolVibes,
		Published: true, Audience: &audience,
	}
	if err := db.Omit(clause.Associations).Create(&parent).Error; err != nil {
		t.Fatalf("create parent post: %v", err)
	}

	comments := make([]post.Post, 0, commentCount)
	for i := 0; i < commentCount; i++ {
		comment := post.Post{
			ID: uuid.New(), ParentID: &parent.ID, PublicID: basePublicID + int64(3+i),
			AuthorID: commentAuthor.ID, PostKind: post.PostKindPost,
			Domain: models.CoolVibes, Published: true, Audience: &audience,
		}
		comments = append(comments, comment)
	}
	if len(comments) > 0 {
		if err := db.Omit(clause.Associations).Create(&comments).Error; err != nil {
			t.Fatalf("create comments: %v", err)
		}
	}

	counts := map[string]any{"comment_count": int64(commentCount)}
	if includeBookmark {
		counts["bookmark_count"] = int64(1)
	}
	countsJSON, err := json.Marshal(counts)
	if err != nil {
		t.Fatal(err)
	}
	aggregate := models.Engagement{
		ID: uuid.New(), ContentableID: parent.ID,
		ContentableType: models.EngagementContentableTypePost,
		Counts:          datatypes.JSON(countsJSON),
	}
	if err := db.Omit(clause.Associations).Create(&aggregate).Error; err != nil {
		t.Fatalf("create parent aggregate: %v", err)
	}

	details := make([]models.EngagementDetail, 0, commentCount+1)
	for i := range comments {
		dedupeKey := commentEngagementDedupeKey(comments[i].ID)
		details = append(details, models.EngagementDetail{
			ID: uuid.New(), EngagementID: aggregate.ID, DedupeKey: &dedupeKey,
			EngagerID: commentAuthor.ID, EngageeID: parentAuthor.ID,
			Kind: models.EngagementKindComment,
			Details: datatypes.JSON([]byte(fmt.Sprintf(
				`{"comment_post_id":%q}`,
				comments[i].ID.String(),
			))),
		})
	}
	if includeBookmark {
		details = append(details, models.EngagementDetail{
			ID: uuid.New(), EngagementID: aggregate.ID,
			EngagerID: commentAuthor.ID, EngageeID: parentAuthor.ID,
			Kind: models.EngagementKindBookmark,
		})
	}
	if len(details) > 0 {
		if err := db.Omit(clause.Associations).Create(&details).Error; err != nil {
			t.Fatalf("create comment engagement details: %v", err)
		}
	}

	engagementRepo := NewEngagementRepository(db)
	userRepo := NewUserRepository(db, nil, nil, engagementRepo, nil)
	fixture := commentDeleteFixture{
		db: db, repo: NewPostRepository(db, nil, nil, userRepo, nil),
		author: commentAuthor, parent: parent, comments: comments,
		aggregate: aggregate, details: details,
	}
	t.Cleanup(func() { cleanupCommentDeleteFixture(db, fixture, parentAuthor) })
	return fixture
}

func cleanupCommentDeleteFixture(db *gorm.DB, fixture commentDeleteFixture, parentAuthor models.User) {
	db.Unscoped().Where("engagement_id = ?", fixture.aggregate.ID).Delete(&models.EngagementDetail{})
	db.Unscoped().Where("id = ?", fixture.aggregate.ID).Delete(&models.Engagement{})
	postIDs := []uuid.UUID{fixture.parent.ID}
	for _, comment := range fixture.comments {
		postIDs = append(postIDs, comment.ID)
	}
	db.Unscoped().Where("id IN ?", postIDs).Delete(&post.Post{})
	db.Unscoped().Where("id IN ?", []uuid.UUID{fixture.author.ID, parentAuthor.ID}).Delete(&models.User{})
}

func commentDeleteFilter(comment post.Post, author models.User, ctx context.Context) types.Filter {
	return types.Filter{
		Context: ctx,
		PostID:  comment.PublicID,
		AuthUser: &types.Actor{
			ID: author.ID, PublicID: author.PublicID, Role: string(author.UserRole),
		},
	}
}

func commentAggregateCount(t *testing.T, db *gorm.DB, aggregateID uuid.UUID, key string) int64 {
	t.Helper()
	var aggregate models.Engagement
	if err := db.First(&aggregate, "id = ?", aggregateID).Error; err != nil {
		t.Fatalf("load aggregate: %v", err)
	}
	var counts map[string]any
	if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
		t.Fatalf("decode aggregate counts: %v", err)
	}
	value, ok := counts[key].(float64)
	if !ok {
		t.Fatalf("aggregate %s = %#v; want numeric count", key, counts[key])
	}
	return int64(value)
}

func TestDeleteCommentMutatesPostAndExactParentDetailAtomicallyIntegration(t *testing.T) {
	fixture := prepareCommentDeleteFixture(t, 2, false)

	if err := fixture.repo.Delete(commentDeleteFilter(fixture.comments[0], fixture.author, context.Background())); err != nil {
		t.Fatalf("Delete(comment) error = %v", err)
	}

	var deleted post.Post
	if err := fixture.db.Unscoped().First(&deleted, "id = ?", fixture.comments[0].ID).Error; err != nil {
		t.Fatalf("load deleted comment: %v", err)
	}
	if !deleted.DeletedAt.Valid {
		t.Fatal("comment was not soft-deleted")
	}
	if err := fixture.db.First(&post.Post{}, "id = ?", fixture.comments[1].ID).Error; err != nil {
		t.Fatalf("sibling comment was deleted: %v", err)
	}

	var firstDetailCount, secondDetailCount int64
	fixture.db.Model(&models.EngagementDetail{}).
		Where("dedupe_key = ?", commentEngagementDedupeKey(fixture.comments[0].ID)).
		Count(&firstDetailCount)
	fixture.db.Model(&models.EngagementDetail{}).
		Where("dedupe_key = ?", commentEngagementDedupeKey(fixture.comments[1].ID)).
		Count(&secondDetailCount)
	if firstDetailCount != 0 || secondDetailCount != 1 {
		t.Fatalf("comment details after delete = %d/%d; want 0/1", firstDetailCount, secondDetailCount)
	}
	if got := commentAggregateCount(t, fixture.db, fixture.aggregate.ID, "comment_count"); got != 1 {
		t.Fatalf("comment_count = %d; want 1", got)
	}
}

func TestCreateCommentPersistsPostAndParentAggregateInOneTransactionIntegration(t *testing.T) {
	fixture := prepareCommentDeleteFixture(t, 0, false)
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatalf("create comment snowflake node: %v", err)
	}
	creationRepo := *fixture.repo
	creationRepo.snowFlakeNode = node
	creationRepo.userRepo = nil // notification delivery is outside this aggregate test

	created, err := creationRepo.CreateContentablePost(
		context.Background(),
		ports.FormData{Values: map[string][]string{
			"parentPostId": {fmt.Sprintf("%d", fixture.parent.PublicID)},
			"title":        {"atomic comment"},
			"audience":     {"public"},
		}},
		&fixture.author,
		string(post.PostKindPost),
		nil,
	)
	if err != nil {
		t.Fatalf("CreateContentablePost(comment) error = %v", err)
	}
	t.Cleanup(func() { fixture.db.Unscoped().Where("id = ?", created.ID).Delete(&post.Post{}) })
	if created.ParentID == nil || *created.ParentID != fixture.parent.ID {
		t.Fatalf("created comment parent = %v; want %s", created.ParentID, fixture.parent.ID)
	}

	var detail models.EngagementDetail
	if err := fixture.db.
		Where("engagement_id = ? AND dedupe_key = ?", fixture.aggregate.ID, commentEngagementDedupeKey(created.ID)).
		First(&detail).Error; err != nil {
		t.Fatalf("load transactional comment detail: %v", err)
	}
	if detail.Kind != models.EngagementKindComment || detail.EngagerID != fixture.author.ID {
		t.Fatalf("comment detail = %#v; want author-owned comment detail", detail)
	}
	if got := commentAggregateCount(t, fixture.db, fixture.aggregate.ID, "comment_count"); got != 1 {
		t.Fatalf("comment_count = %d; want 1", got)
	}
}

func TestCreateCommentRollsBackPostWhenAggregateMutationFailsIntegration(t *testing.T) {
	fixture := prepareCommentDeleteFixture(t, 0, false)
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatalf("create comment snowflake node: %v", err)
	}
	creationRepo := *fixture.repo
	creationRepo.snowFlakeNode = node
	creationRepo.userRepo = nil

	forcedErr := errors.New("forced comment detail create failure")
	callbackName := "test:force-comment-detail-create:" + uuid.NewString()
	if err := fixture.db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Schema != nil && tx.Statement.Schema.Table == "engagement_details" {
			_ = tx.AddError(forcedErr)
		}
	}); err != nil {
		t.Fatalf("register forced create failure callback: %v", err)
	}
	t.Cleanup(func() { _ = fixture.db.Callback().Create().Remove(callbackName) })

	_, err = creationRepo.CreateContentablePost(
		context.Background(),
		ports.FormData{Values: map[string][]string{
			"parentPostId": {fmt.Sprintf("%d", fixture.parent.PublicID)},
			"title":        {"rolled back comment"},
			"audience":     {"public"},
		}},
		&fixture.author,
		string(post.PostKindPost),
		nil,
	)
	if !errors.Is(err, forcedErr) {
		t.Fatalf("CreateContentablePost(comment) error = %v; want wrapped %v", err, forcedErr)
	}

	var childCount int64
	if err := fixture.db.Model(&post.Post{}).
		Where("parent_id = ?", fixture.parent.ID).
		Count(&childCount).Error; err != nil {
		t.Fatalf("count rolled-back comments: %v", err)
	}
	if childCount != 0 {
		t.Fatalf("active comments after aggregate failure = %d; want 0", childCount)
	}
	if got := commentAggregateCount(t, fixture.db, fixture.aggregate.ID, "comment_count"); got != 0 {
		t.Fatalf("comment_count after rollback = %d; want 0", got)
	}
}

func TestDeleteCommentRollsBackSoftDeleteWhenDetailMutationFailsIntegration(t *testing.T) {
	fixture := prepareCommentDeleteFixture(t, 1, false)
	forcedErr := errors.New("forced comment detail delete failure")
	callbackName := "test:force-comment-detail-delete:" + uuid.NewString()
	if err := fixture.db.Callback().Delete().Before("gorm:delete").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Schema != nil && tx.Statement.Schema.Table == "engagement_details" {
			_ = tx.AddError(forcedErr)
		}
	}); err != nil {
		t.Fatalf("register forced failure callback: %v", err)
	}
	t.Cleanup(func() { _ = fixture.db.Callback().Delete().Remove(callbackName) })

	err := fixture.repo.Delete(commentDeleteFilter(fixture.comments[0], fixture.author, context.Background()))
	if !errors.Is(err, forcedErr) {
		t.Fatalf("Delete(comment) error = %v; want wrapped %v", err, forcedErr)
	}
	if err := fixture.db.First(&post.Post{}, "id = ?", fixture.comments[0].ID).Error; err != nil {
		t.Fatalf("failed aggregate mutation did not roll back soft delete: %v", err)
	}
	var detailCount int64
	if err := fixture.db.Model(&models.EngagementDetail{}).
		Where("id = ?", fixture.details[0].ID).
		Count(&detailCount).Error; err != nil {
		t.Fatalf("count rolled-back detail: %v", err)
	}
	if detailCount != 1 {
		t.Fatalf("detail count after rollback = %d; want 1", detailCount)
	}
	if got := commentAggregateCount(t, fixture.db, fixture.aggregate.ID, "comment_count"); got != 1 {
		t.Fatalf("comment_count after rollback = %d; want 1", got)
	}
}

func TestDeleteCommentAndRemoveSiblingEngagementShareCanonicalLockIntegration(t *testing.T) {
	fixture := prepareCommentDeleteFixture(t, 1, true)
	bookmarkDetail := fixture.details[len(fixture.details)-1]
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	errs := make(chan error, 2)
	var writers sync.WaitGroup
	writers.Add(2)
	go func() {
		defer writers.Done()
		errs <- fixture.repo.Delete(commentDeleteFilter(fixture.comments[0], fixture.author, ctx))
	}()
	go func() {
		defer writers.Done()
		errs <- fixture.repo.userRepo.engagementRepo.RemoveEngagementDetail(ctx, bookmarkDetail.ID)
	}()
	writers.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent aggregate mutation error = %v", err)
		}
	}
	if ctx.Err() != nil {
		t.Fatalf("concurrent aggregate mutations timed out (possible deadlock): %v", ctx.Err())
	}

	var remainingDetails int64
	if err := fixture.db.Model(&models.EngagementDetail{}).
		Where("engagement_id = ?", fixture.aggregate.ID).
		Count(&remainingDetails).Error; err != nil {
		t.Fatalf("count remaining details: %v", err)
	}
	if remainingDetails != 0 {
		t.Fatalf("remaining engagement details = %d; want 0", remainingDetails)
	}
	if got := commentAggregateCount(t, fixture.db, fixture.aggregate.ID, "comment_count"); got != 0 {
		t.Fatalf("comment_count = %d; want 0", got)
	}
	if got := commentAggregateCount(t, fixture.db, fixture.aggregate.ID, "bookmark_count"); got != 0 {
		t.Fatalf("bookmark_count = %d; want 0", got)
	}
}
