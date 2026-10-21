package repositories

import (
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

func repositorySubquery(db *gorm.DB) *gorm.DB { return db.Session(&gorm.Session{NewDB: true}) }
func userDomainSubquery(db *gorm.DB, viewer uuid.UUID) *gorm.DB {
	return repositorySubquery(db).Model(&models.User{}).Select("domain").Where("id = ?", viewer)
}
func userEngagementIDs(db *gorm.DB) *gorm.DB {
	return repositorySubquery(db).Model(&models.Engagement{}).Select("id").Where("contentable_type = ?", models.EngagementContentableTypeUser)
}
func userRelationshipTargets(db *gorm.DB, viewer uuid.UUID, kind models.EngagementKind, reverse bool) *gorm.DB {
	query := repositorySubquery(db).Model(&models.EngagementDetail{}).Where("engagement_id IN (?)", userEngagementIDs(db)).Where("kind = ?", kind)
	if reverse {
		return query.Select("engager_id").Where("engagee_id = ?", viewer)
	}
	return query.Select("engagee_id").Where("engagee_id IS NOT NULL").Where("engager_id = ?", viewer)
}
func userBlockScope(query *gorm.DB, viewer uuid.UUID, alias string) *gorm.DB {
	return query.Where(alias+".id NOT IN (?)", userRelationshipTargets(query, viewer, models.EngagementKindBlocking, false)).
		Where(alias+".id NOT IN (?)", userRelationshipTargets(query, viewer, models.EngagementKindBlocking, true))
}
func userAudienceScope(query *gorm.DB, viewer uuid.UUID, alias string) *gorm.DB {
	fresh := func() *gorm.DB { return repositorySubquery(query) }
	firstFriends := fresh().Model(&models.Friendship{}).Select("second_user_id").Where("first_user_id = ? AND status = ?", viewer, domainuser.FriendshipAccepted)
	secondFriends := fresh().Model(&models.Friendship{}).Select("first_user_id").Where("second_user_id = ? AND status = ?", viewer, domainuser.FriendshipAccepted)
	friends := fresh().Where(alias+".privacy_level = ?", constants.PrivacyFriendsOnly).Where(fresh().Where(alias+".id IN (?)", firstFriends).Or(alias+".id IN (?)", secondFriends))
	following := userRelationshipTargets(query, viewer, models.EngagementKindFollowing, false)
	followers := userRelationshipTargets(query, viewer, models.EngagementKindFollowing, true)
	followsOnly := fresh().Where(alias+".privacy_level = ?", constants.PrivacyFollowersOnly).Where(alias+".id IN (?)", following)
	mutuals := fresh().Where(alias+".privacy_level = ?", constants.PrivacyMutualsOnly).Where(alias+".id IN (?)", following).Where(alias+".id IN (?)", followers)
	allowed := fresh().Where(alias+".id = ?", viewer).Or(alias+".privacy_level = ?", constants.PrivacyPublic).Or(friends).Or(followsOnly).Or(mutuals)
	return query.Where(allowed)
}

func localLiveSessionScope(query *gorm.DB, alias string) *gorm.DB {
	owners := repositorySubquery(query).Model(&models.BroadcastSession{}).Select("owner_id").Where("ended_at IS NULL AND expires_at > ?", time.Now().UTC())
	allowed := repositorySubquery(query).Where(alias+".is_bot = ?", true).Or(alias+".id IN (?)", owners)
	return query.Where(allowed)
}
