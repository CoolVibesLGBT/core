package repositories

import (
	"core/models"
	postpayloads "core/models/post/payloads"
	"core/models/utils"
	"gorm.io/gorm"
)

func matchingText(db *gorm.DB, pattern string, fields ...string) *gorm.DB {
	query := repositorySubquery(db)
	for i, field := range fields {
		predicate := "COALESCE(" + field + "::text, '') ILIKE ? ESCAPE '\\'"
		if i == 0 {
			query = query.Where(predicate, pattern)
		} else {
			query = query.Or(predicate, pattern)
		}
	}
	return query
}

func applyPostTextSearch(query *gorm.DB, pattern string) *gorm.DB {
	// Unscoped preserves the existing search behavior for historical payloads;
	// the outer post query still enforces publication and deletion visibility.
	events := repositorySubquery(query).Unscoped().Model(&postpayloads.Event{}).Select("post_id").
		Where(matchingText(query, pattern, "title", "description"))
	locations := repositorySubquery(query).Unscoped().Model(&utils.Location{}).Select("contentable_id").
		Where("contentable_type = ?", utils.LocationOwnerPost).
		Where(matchingText(query, pattern, "address", "city", "town", "province", "country"))
	eventLocations := repositorySubquery(query).Unscoped().Model(&utils.Location{}).Select("contentable_id").
		Where("contentable_type = ?", utils.LocationOwnerEvent).
		Where(matchingText(query, pattern, "address", "city", "town", "province", "country"))
	eventPosts := repositorySubquery(query).Unscoped().Model(&postpayloads.Event{}).Select("post_id").Where("id IN (?)", eventLocations)
	hashtags := repositorySubquery(query).Model(&models.Hashtag{}).Select("taggable_id").Where("taggable_type = ?", "post").
		Where(matchingText(query, pattern, "tag", "slug"))
	return query.Where(matchingText(query, pattern, "title", "content", "summary", "metadata", "extras", "slug").
		Or("posts.id IN (?)", events).Or("posts.id IN (?)", locations).
		Or("posts.id IN (?)", eventPosts).Or("posts.id IN (?)", hashtags))
}
