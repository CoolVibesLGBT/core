package repositories

import (
	"core/application/types"
	"core/constants"
	"core/models"
	"core/models/post"
	"time"

	"github.com/google/uuid"
)

func (r *PostRepository) FetchPublicUserPostCollection(filters types.Filter, bookmarks bool) ([]types.PublicPost, error) {
	if filters.AuthUser == nil || filters.AuthUser.ID == uuid.Nil || (bookmarks && filters.UserUUID != filters.AuthUser.ID) {
		return nil, constants.ErrUserUnauthorized
	}
	db := r.db.WithContext(filters.Context)
	kind := models.EngagementKindLikeReceived
	if bookmarks {
		kind = models.EngagementKindBookmark
	}
	interactions := repositorySubquery(db).Table("engagements e").Select("e.contentable_id").
		Joins("JOIN engagement_details d ON d.engagement_id = e.id").
		Where("e.contentable_type = ? AND d.engager_id = ? AND d.kind = ?", models.EngagementContentableTypePost, filters.UserUUID, kind)
	users := repositorySubquery(db).Model(&models.User{}).Select("users.id").
		Where("users.domain = (?)", userDomainSubquery(db, filters.AuthUser.ID)).
		Where("users.user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending})
	users = userAudienceScope(userBlockScope(users, filters.AuthUser.ID, "users"), filters.AuthUser.ID, "users")
	limit := filters.Limit
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}
	query := db.Model(&post.Post{}).
		Where("posts.id IN (?)", interactions).
		Where("posts.author_id IN (?)", users).
		Where("? IN (?)", filters.UserUUID, users).
		Where("posts.published = TRUE AND posts.parent_id IS NULL AND posts.post_kind = ?", post.PostKindPost).
		Where("COALESCE(NULLIF(posts.audience, ''), 'public') = 'public'").
		Where("posts.expires_at IS NULL OR posts.expires_at > ?", time.Now().UTC()).
		Preload("Location").Preload("Poll").Preload("Poll.Choices").
		Preload("Event").Preload("Event.Location").Preload("Event.Attendees", preloadEventAttendees).
		Preload("Author").Preload("Author.Cover").Preload("Author.Avatar").
		Preload("Hashtags").Preload("Attachments").Preload("Attachments.File").
		Order("posts.public_id DESC").Limit(limit)
	if filters.Cursor != nil {
		query = query.Where("posts.public_id < ?", *filters.Cursor)
	}
	var items []post.Post
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return publicPostViews(items), nil
}
