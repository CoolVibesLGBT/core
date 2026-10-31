package repositories

import (
	"context"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"core/models/notifications"
	modelutils "core/models/utils"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MatchesRepository struct {
	db               *gorm.DB
	notificationRepo matchNotificationWriter
}

type matchNotificationWriter interface {
	CreateNotification(ctx context.Context, senderUser uuid.UUID, receiverUser uuid.UUID, notifType, title, message string, payload notifications.NotificationPayload) (*notifications.Notification, error)
}

func NewMatchesRepository(db *gorm.DB, notificationRepo matchNotificationWriter) *MatchesRepository {
	return &MatchesRepository{
		db:               db,
		notificationRepo: notificationRepo,
	}
}

type matchUserIdentity struct {
	ID           uuid.UUID              `gorm:"column:id"`
	PublicID     int64                  `gorm:"column:public_id"`
	Domain       models.DomainKind      `gorm:"column:domain"`
	PrivacyLevel constants.PrivacyLevel `gorm:"column:privacy_level"`
}

func matchDetailDedupeKey(kind models.EngagementKind, engagerID, engageeID uuid.UUID) string {
	return fmt.Sprintf("match:%s:%s:%s", kind, engagerID, engageeID)
}

// setCanonicalMatchDetail enforces exactly one EngagementDetail row per
// (kind, engager, engagee) triple, keyed by its dedupe_key unique index.
func setCanonicalMatchDetail(
	tx *gorm.DB,
	expectedAggregate *models.Engagement,
	engagerID uuid.UUID,
	engageeID uuid.UUID,
	kind models.EngagementKind,
	enabled bool,
) error {
	if tx == nil || expectedAggregate == nil {
		return constants.ErrMatchEngagementAggregateRequired
	}

	dedupeKey := matchDetailDedupeKey(kind, engagerID, engageeID)
	var existing models.EngagementDetail
	err := tx.Where("dedupe_key = ?", dedupeKey).Take(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	found := err == nil

	if !enabled {
		if found {
			return tx.Delete(&models.EngagementDetail{}, "id = ?", existing.ID).Error
		}
		return nil
	}

	if !found {
		now := time.Now().UTC()
		detail := models.EngagementDetail{
			ID:           uuid.New(),
			EngagementID: expectedAggregate.ID,
			DedupeKey:    &dedupeKey,
			EngagerID:    engagerID,
			EngageeID:    engageeID,
			Kind:         kind,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		return tx.Create(&detail).Error
	}
	return tx.Model(&models.EngagementDetail{}).
		Where("id = ?", existing.ID).
		Update("updated_at", time.Now().UTC()).Error
}

var matchingEngagementKinds = []models.EngagementKind{
	models.EngagementKindLikeGiven,
	models.EngagementKindLikeReceived,
	models.EngagementKindDislikeGiven,
	models.EngagementKindDisLikeReceived,
	models.EngagementKindMatched,
	models.EngagementKindViewGiven,
	models.EngagementKindViewReceived,
}

func syncMatchingAggregateCounts(tx *gorm.DB, aggregate *models.Engagement) error {
	return syncUserAggregateKinds(tx, aggregate, matchingEngagementKinds)
}

func syncUserAggregateKinds(tx *gorm.DB, aggregate *models.Engagement, kinds []models.EngagementKind) error {
	if tx == nil || aggregate == nil {
		return constants.ErrMatchEngagementAggregateRequired
	}
	type engagementKindCount struct {
		Kind  models.EngagementKind `gorm:"column:kind"`
		Count int64                 `gorm:"column:count"`
	}
	var rows []engagementKindCount
	if err := tx.Model(&models.EngagementDetail{}).
		Select("kind, COUNT(*) AS count").
		Where("engagement_id = ? AND kind IN ?", aggregate.ID, kinds).
		Group("kind").
		Find(&rows).Error; err != nil {
		return err
	}

	counts := make(map[string]interface{})
	if len(aggregate.Counts) > 0 && string(aggregate.Counts) != "null" {
		if err := json.Unmarshal(aggregate.Counts, &counts); err != nil {
			return err
		}
	}
	if counts == nil {
		counts = make(map[string]interface{})
	}
	for _, kind := range kinds {
		countKey, ok := models.EngagementCountKeys[kind]
		if !ok || countKey.CountKey == "" {
			return constants.ErrEngagementCounterMissing.Format(kind)
		}
		counts[countKey.CountKey] = int64(0)
	}
	for _, row := range rows {
		counts[models.EngagementCountKeys[row.Kind].CountKey] = row.Count
	}
	payload, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	aggregate.Counts = datatypes.JSON(payload)
	return tx.Model(&models.Engagement{}).
		Where("id = ?", aggregate.ID).
		Updates(map[string]interface{}{"counts": aggregate.Counts, "updated_at": time.Now().UTC()}).Error
}

// Both profile reactions and swipe reactions must update the same canonical
// rows. A selected reaction replaces its opposite; removing it selects none.
func setUserReaction(tx *gorm.DB, actor, target *models.Engagement, pair domainuser.EngagementPair, enabled bool) error {
	if err := setCanonicalMatchDetail(tx, actor, actor.ContentableID, target.ContentableID, models.EngagementKind(pair.Given), enabled); err != nil {
		return err
	}
	if err := setCanonicalMatchDetail(tx, target, target.ContentableID, actor.ContentableID, models.EngagementKind(pair.Received), enabled); err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	opposite, err := domainuser.InteractionEngagementPair(domainuser.InteractionLike, pair.Given != domainuser.EngagementLikeGiven)
	if err != nil {
		return err
	}
	if err := setCanonicalMatchDetail(tx, actor, actor.ContentableID, target.ContentableID, models.EngagementKind(opposite.Given), false); err != nil {
		return err
	}
	return setCanonicalMatchDetail(tx, target, target.ContentableID, actor.ContentableID, models.EngagementKind(opposite.Received), false)
}

func syncMutualUserMatch(tx *gorm.DB, actor, target *models.Engagement) (matched, newMatch bool, err error) {
	forward, err := userMatchDetailExists(tx, actor.ContentableID, target.ContentableID, models.EngagementKindLikeGiven)
	if err != nil {
		return false, false, err
	}
	reverse, err := userMatchDetailExists(tx, target.ContentableID, actor.ContentableID, models.EngagementKindLikeGiven)
	if err != nil {
		return false, false, err
	}
	hadForward, err := userMatchDetailExists(tx, actor.ContentableID, target.ContentableID, models.EngagementKindMatched)
	if err != nil {
		return false, false, err
	}
	hadReverse, err := userMatchDetailExists(tx, target.ContentableID, actor.ContentableID, models.EngagementKindMatched)
	if err != nil {
		return false, false, err
	}
	matched = forward && reverse
	if matched {
		for _, pair := range [][2]uuid.UUID{{actor.ContentableID, target.ContentableID}, {target.ContentableID, actor.ContentableID}} {
			blocked, err := userMatchDetailExists(tx, pair[0], pair[1], models.EngagementKindBlocking)
			if err != nil {
				return false, false, err
			}
			if blocked {
				matched = false
				break
			}
		}
	}
	newMatch = matched && !hadForward && !hadReverse
	if err := setCanonicalMatchDetail(tx, actor, actor.ContentableID, target.ContentableID, models.EngagementKindMatched, matched); err != nil {
		return false, false, err
	}
	if err := setCanonicalMatchDetail(tx, target, target.ContentableID, actor.ContentableID, models.EngagementKindMatched, matched); err != nil {
		return false, false, err
	}
	return matched, newMatch, nil
}

func userMatchDetailExists(tx *gorm.DB, engagerID, engageeID uuid.UUID, kind models.EngagementKind) (bool, error) {
	var count int64
	err := tx.Model(&models.EngagementDetail{}).
		Joins("JOIN engagements AS match_aggregate ON match_aggregate.id = engagement_details.engagement_id").
		Where("match_aggregate.contentable_type = ?", models.EngagementContentableTypeUser).
		Where("engagement_details.engager_id = ? AND engagement_details.engagee_id = ? AND engagement_details.kind = ?", engagerID, engageeID, kind).
		Count(&count).Error
	return count > 0, err
}

func (m *MatchesRepository) RecordView(ctx context.Context, fromUserID uuid.UUID, toUserID uuid.UUID, reaction domainuser.MatchReaction) (bool, error) {
	if m == nil || m.db == nil {
		return false, constants.ErrMatchesRepositoryNotConfigured
	}
	if fromUserID == uuid.Nil || toUserID == uuid.Nil {
		return false, domainuser.ErrInvalidMatchReaction
	}
	if fromUserID == toUserID {
		return false, domainuser.ErrSelfInteraction
	}
	selectedPair, _, err := reaction.EngagementPairs()
	if err != nil {
		return false, err
	}

	isPostgres := m.db.Name() == "postgres"
	if !isPostgres {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}

	var identities map[uuid.UUID]matchUserIdentity
	var matched bool
	var newMatch bool
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var users []matchUserIdentity
		query := tx.Table("users").
			Select("id, public_id, domain, privacy_level").
			Where("id IN ?", []uuid.UUID{fromUserID, toUserID}).
			Where("deleted_at IS NULL AND is_bot = FALSE").
			Where("user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending})
		if err := query.Find(&users).Error; err != nil {
			return err
		}
		if len(users) != 2 {
			return domainuser.ErrMatchTargetUnavailable
		}
		identities = make(map[uuid.UUID]matchUserIdentity, len(users))
		for _, user := range users {
			identities[user.ID] = user
		}
		fromIdentity, fromExists := identities[fromUserID]
		toIdentity, toExists := identities[toUserID]
		if !fromExists || !toExists || fromIdentity.Domain != toIdentity.Domain || toIdentity.PrivacyLevel != constants.PrivacyPublic {
			return domainuser.ErrMatchTargetUnavailable
		}

		if isPostgres {
			if err := lockUserEngagementAggregates(tx, fromUserID, toUserID); err != nil {
				return err
			}
		}
		actorBlockedTarget, err := userMatchDetailExists(tx, fromUserID, toUserID, models.EngagementKindBlocking)
		if err != nil {
			return err
		}
		targetBlockedActor, err := userMatchDetailExists(tx, toUserID, fromUserID, models.EngagementKindBlocking)
		if err != nil {
			return err
		}
		if actorBlockedTarget || targetBlockedActor {
			return domainuser.ErrMatchTargetUnavailable
		}
		fromAggregate, err := loadUserEngagementAggregate(tx, fromUserID)
		if err != nil {
			return err
		}
		toAggregate, err := loadUserEngagementAggregate(tx, toUserID)
		if err != nil {
			return err
		}
		fromAggregate, err = ensureUserEngagementAggregate(tx, fromAggregate, fromUserID)
		if err != nil {
			return err
		}
		toAggregate, err = ensureUserEngagementAggregate(tx, toAggregate, toUserID)
		if err != nil {
			return err
		}

		if err := setUserReaction(tx, fromAggregate, toAggregate, selectedPair, true); err != nil {
			return err
		}

		if _, err := recordCanonicalUserView(tx, fromAggregate, toAggregate, true); err != nil {
			return err
		}

		matched, newMatch, err = syncMutualUserMatch(tx, fromAggregate, toAggregate)
		if err != nil {
			return err
		}
		if err := syncMatchingAggregateCounts(tx, fromAggregate); err != nil {
			return err
		}
		if err := syncMatchingAggregateCounts(tx, toAggregate); err != nil {
			return err
		}
		if newMatch {
			return enqueueMatchNotifications(tx, fromUserID, toUserID)
		}
		return nil
	})
	if err != nil {
		return false, err
	}

	return matched, nil
}

const matchPublicUserProjection = `
	engagement_details.created_at AS cursor_created_at,
	engagement_details.id AS cursor_id,
	users.public_id,
	users.user_name,
	users.display_name,
	users.bio,
	users.is_online,
	profile_location.display AS location_display,
	profile_location.city AS location_city,
	profile_location.region AS location_region,
	profile_location.country AS location_country,
	avatar_file.url AS avatar_url,
	avatar_file.variants AS avatar_variants
`

const unseenPublicUserProjection = `
	users.public_id,
	users.user_name,
	users.display_name,
	users.bio,
	users.is_online,
	profile_location.display AS location_display,
	profile_location.city AS location_city,
	profile_location.region AS location_region,
	profile_location.country AS location_country,
	avatar_file.url AS avatar_url,
	avatar_file.variants AS avatar_variants
`

func normalizedMatchLimit(limit int) int {
	if limit <= 0 {
		return constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		return constants.MAXIMUM_LIMIT
	}
	return limit
}

func applyMatchPublicUserJoins(query *gorm.DB) *gorm.DB {
	return query.
		Joins(`
			LEFT JOIN locations AS profile_location
			  ON profile_location.contentable_id = users.id
			 AND profile_location.contentable_type = ?
			 AND profile_location.deleted_at IS NULL
		`, modelutils.LocationOwnerUser).
		Joins("LEFT JOIN medias AS avatar_media ON avatar_media.id = users.avatar_id AND avatar_media.is_public = TRUE").
		Joins("LEFT JOIN file_metadata AS avatar_file ON avatar_file.id = avatar_media.file_id")
}

func (m *MatchesRepository) matchListPage(
	ctx context.Context,
	userID uuid.UUID,
	kind models.EngagementKind,
	cursor *types.MatchListCursor,
	limit int,
) (types.MatchListPage, error) {
	limit = normalizedMatchLimit(limit)
	query := m.db.WithContext(ctx).
		Table("engagement_details").
		Select(matchPublicUserProjection).
		Joins("JOIN engagements AS relationship_aggregate ON relationship_aggregate.id = engagement_details.engagement_id AND relationship_aggregate.contentable_type = ?", models.EngagementContentableTypeUser).
		Joins("JOIN users ON users.id = engagement_details.engagee_id").
		Where("engagement_details.engager_id = ? AND engagement_details.kind = ?", userID, kind).
		Where("users.domain = (?)", userDomainSubquery(m.db, userID))
	query = applyMatchPublicUserJoins(userBlockScope(query, userID, "users"))
	query = publicUserVisibilityScope(query)

	if cursor != nil {
		if cursor.DetailID == uuid.Nil {
			query = query.Where("engagement_details.created_at < ?", cursor.OccurredAt)
		} else {
			query = query.Where(
				"engagement_details.created_at < ? OR (engagement_details.created_at = ? AND engagement_details.id < ?)",
				cursor.OccurredAt,
				cursor.OccurredAt,
				cursor.DetailID,
			)
		}
	}

	var rows []publicUserProjectionRow
	if err := query.
		Order("engagement_details.created_at DESC, engagement_details.id DESC").
		Limit(limit + 1).
		Scan(&rows).Error; err != nil {
		return types.MatchListPage{}, err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	page := types.MatchListPage{Users: make([]types.PublicUserSummary, 0, len(rows))}
	for index := range rows {
		page.Users = append(page.Users, rows[index].summaryView())
	}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		page.Next = &types.MatchListCursor{OccurredAt: last.CursorCreatedAt, DetailID: last.CursorID}
	}
	return page, nil
}

func (m *MatchesRepository) GetLikesAfter(
	ctx context.Context,
	userID uuid.UUID,
	cursor *types.MatchListCursor,
	limit int,
) (types.MatchListPage, error) {
	return m.matchListPage(ctx, userID, models.EngagementKindLikeGiven, cursor, limit)
}

func (m *MatchesRepository) GetMatchesAfter(
	ctx context.Context,
	userID uuid.UUID,
	cursor *types.MatchListCursor,
	limit int,
) (types.MatchListPage, error) {
	return m.matchListPage(ctx, userID, models.EngagementKindMatched, cursor, limit)
}

// -----------------------------------------------
// 🔍 Geçen 24 saatte görmediğin kullanıcılar
// -----------------------------------------------

func (m *MatchesRepository) GetUnseenUsers(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
	options ...domainuser.MatchCriteria,
) ([]types.PublicUserSummary, error) {
	criteria := domainuser.MatchCriteria{}
	if len(options) > 1 {
		return nil, domainuser.ErrInvalidMatchCriteria
	}
	if len(options) == 1 {
		criteria = options[0]
	}
	if err := criteria.Validate(); err != nil {
		return nil, err
	}

	limit = normalizedMatchLimit(limit)
	recentlySeen := m.db.
		Table("engagement_details AS seen_detail").
		Select("seen_detail.engagee_id").
		Joins("JOIN engagements AS seen_aggregate ON seen_aggregate.id = seen_detail.engagement_id AND seen_aggregate.contentable_type = ?", models.EngagementContentableTypeUser).
		Where("seen_detail.engagee_id IS NOT NULL").
		Where("seen_detail.engager_id = ? AND seen_detail.updated_at >= ? AND seen_detail.kind = ?", userID, time.Now().UTC().Add(-24*time.Hour), models.EngagementKindViewGiven)

	query := m.db.WithContext(ctx).
		Table("users").
		Select(unseenPublicUserProjection).
		Where("users.id != ?", userID).
		Where("users.id NOT IN (?)", recentlySeen).
		Where("users.id NOT IN (?)", userRelationshipTargets(m.db, userID, models.EngagementKindMatched, false)).
		Where("users.domain = (?)", userDomainSubquery(m.db, userID))
	var principal models.User
	if err := m.db.WithContext(ctx).Preload("Location", func(db *gorm.DB) *gorm.DB {
		return db.Select("id", "contentable_id", "contentable_type", "latitude", "longitude")
	}).First(&principal, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []types.PublicUserSummary{}, nil
		}
		return nil, err
	}
	query = publicUserVisibilityScope(userBlockScope(query, userID, "users")).Model(&models.User{})
	ids, err := rankMatchingUsers(query, principal, criteria, limit)
	if err != nil {
		return nil, err
	}
	return matchingSummaryPage(m.db.WithContext(ctx), userID, ids)
}

func (m *MatchesRepository) GetPassesAfter(
	ctx context.Context,
	userID uuid.UUID,
	cursor *types.MatchListCursor,
	limit int,
) (types.MatchListPage, error) {
	return m.matchListPage(ctx, userID, models.EngagementKindDislikeGiven, cursor, limit)
}

func (m *MatchesRepository) RemoveMatch(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == uuid.Nil || targetID == uuid.Nil {
		return domainuser.ErrInvalidMatchReaction
	}
	if actorID == targetID {
		return domainuser.ErrSelfInteraction
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockUserEngagementAggregates(tx, actorID, targetID); err != nil {
			return err
		}
		actor, err := loadUserEngagementAggregate(tx, actorID)
		if err != nil {
			return err
		}
		target, err := loadUserEngagementAggregate(tx, targetID)
		if err != nil {
			return err
		}
		if actor == nil && target == nil {
			return nil
		}
		actor, err = ensureUserEngagementAggregate(tx, actor, actorID)
		if err != nil {
			return err
		}
		target, err = ensureUserEngagementAggregate(tx, target, targetID)
		if err != nil {
			return err
		}
		pair, _ := domainuser.InteractionEngagementPair(domainuser.InteractionLike, true)
		if err := setUserReaction(tx, actor, target, pair, false); err != nil {
			return err
		}
		if err := setUserReaction(tx, target, actor, pair, false); err != nil {
			return err
		}
		if _, _, err := syncMutualUserMatch(tx, actor, target); err != nil {
			return err
		}
		if err := syncMatchingAggregateCounts(tx, actor); err != nil {
			return err
		}
		return syncMatchingAggregateCounts(tx, target)
	})
}
