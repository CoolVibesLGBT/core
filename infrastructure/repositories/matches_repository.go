package repositories

import (
	"context"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	"core/helpers"
	"core/models"
	"core/models/notifications"
	modelutils "core/models/utils"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MatchesRepository struct {
	db               *gorm.DB
	notificationRepo matchNotificationWriter
}

type matchNotificationWriter interface {
	CreateNotification(senderUser uuid.UUID, receiverUser uuid.UUID, notifType, title, message string, payload notifications.NotificationPayload) (*notifications.Notification, error)
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

// setCanonicalMatchDetail repairs legacy rows that were stored on the other
// user's aggregate and enforces one logical row for the relationship. Both
// user aggregates are locked by RecordView before this helper is called.
func setCanonicalMatchDetail(
	tx *gorm.DB,
	expectedAggregate *models.Engagement,
	otherAggregate *models.Engagement,
	engagerID uuid.UUID,
	engageeID uuid.UUID,
	kind models.EngagementKind,
	enabled bool,
) error {
	if tx == nil || expectedAggregate == nil || otherAggregate == nil {
		return errors.New("match engagement aggregate is required")
	}

	var existing []models.EngagementDetail
	if err := tx.
		Joins("JOIN engagements AS match_aggregate ON match_aggregate.id = engagement_details.engagement_id").
		Where("match_aggregate.contentable_type = ?", models.EngagementContentableTypeUser).
		Where("match_aggregate.contentable_id IN ?", []uuid.UUID{expectedAggregate.ContentableID, otherAggregate.ContentableID}).
		Where("engagement_details.engager_id = ? AND engagement_details.engagee_id = ? AND engagement_details.kind = ?", engagerID, engageeID, kind).
		Clauses(clause.OrderBy{Expression: clause.Expr{SQL: "CASE WHEN engagement_details.engagement_id = ? THEN 0 ELSE 1 END, engagement_details.created_at ASC, engagement_details.id ASC", Vars: []interface{}{expectedAggregate.ID}}}).
		Find(&existing).Error; err != nil {
		return err
	}

	if !enabled {
		if len(existing) > 0 {
			ids := make([]uuid.UUID, 0, len(existing))
			for _, detail := range existing {
				ids = append(ids, detail.ID)
			}
			if err := tx.Delete(&models.EngagementDetail{}, "id IN ?", ids).Error; err != nil {
				return err
			}
		}
		return nil
	}

	dedupeKey := matchDetailDedupeKey(kind, engagerID, engageeID)
	if len(existing) == 0 {
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
		if err := tx.Create(&detail).Error; err != nil {
			return err
		}
	} else {
		keeper := existing[0]
		if len(existing) > 1 {
			duplicateIDs := make([]uuid.UUID, 0, len(existing)-1)
			for _, detail := range existing[1:] {
				duplicateIDs = append(duplicateIDs, detail.ID)
			}
			if err := tx.Delete(&models.EngagementDetail{}, "id IN ?", duplicateIDs).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&models.EngagementDetail{}).
			Where("id = ?", keeper.ID).
			Updates(map[string]interface{}{
				"engagement_id": expectedAggregate.ID,
				"dedupe_key":    dedupeKey,
				"updated_at":    time.Now().UTC(),
			}).Error; err != nil {
			return err
		}
	}

	return nil
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
	if tx == nil || aggregate == nil {
		return errors.New("match engagement aggregate is required")
	}
	type engagementKindCount struct {
		Kind  models.EngagementKind `gorm:"column:kind"`
		Count int64                 `gorm:"column:count"`
	}
	var rows []engagementKindCount
	if err := tx.Model(&models.EngagementDetail{}).
		Select("kind, COUNT(*) AS count").
		Where("engagement_id = ? AND kind IN ?", aggregate.ID, matchingEngagementKinds).
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
	for _, kind := range matchingEngagementKinds {
		countKey, ok := models.EngagementCountKeys[kind]
		if !ok || countKey.CountKey == "" {
			return fmt.Errorf("missing engagement counter for kind %s", kind)
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
		return false, errors.New("matches repository is not configured")
	}
	if fromUserID == uuid.Nil || toUserID == uuid.Nil {
		return false, domainuser.ErrInvalidMatchReaction
	}
	if fromUserID == toUserID {
		return false, domainuser.ErrSelfInteraction
	}
	selectedPair, oppositePair, err := reaction.EngagementPairs()
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
			return gorm.ErrRecordNotFound
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

		if err := setCanonicalMatchDetail(tx, fromAggregate, toAggregate, fromUserID, toUserID, models.EngagementKind(selectedPair.Given), true); err != nil {
			return err
		}
		if err := setCanonicalMatchDetail(tx, toAggregate, fromAggregate, toUserID, fromUserID, models.EngagementKind(selectedPair.Received), true); err != nil {
			return err
		}
		if err := setCanonicalMatchDetail(tx, fromAggregate, toAggregate, fromUserID, toUserID, models.EngagementKind(oppositePair.Given), false); err != nil {
			return err
		}
		if err := setCanonicalMatchDetail(tx, toAggregate, fromAggregate, toUserID, fromUserID, models.EngagementKind(oppositePair.Received), false); err != nil {
			return err
		}

		// A view is an idempotent fact. Replaying a swipe must not toggle it off.
		if err := setCanonicalMatchDetail(tx, fromAggregate, toAggregate, fromUserID, toUserID, models.EngagementKindViewGiven, true); err != nil {
			return err
		}
		if err := setCanonicalMatchDetail(tx, toAggregate, fromAggregate, toUserID, fromUserID, models.EngagementKindViewReceived, true); err != nil {
			return err
		}

		hadForwardMatch, err := userMatchDetailExists(tx, fromUserID, toUserID, models.EngagementKindMatched)
		if err != nil {
			return err
		}
		hadReverseMatch, err := userMatchDetailExists(tx, toUserID, fromUserID, models.EngagementKindMatched)
		if err != nil {
			return err
		}
		otherUserLikes, err := userMatchDetailExists(tx, toUserID, fromUserID, models.EngagementKindLikeGiven)
		if err != nil {
			return err
		}
		matched = reaction.IsLike() && otherUserLikes
		newMatch = matched && !hadForwardMatch && !hadReverseMatch

		if err := setCanonicalMatchDetail(tx, fromAggregate, toAggregate, fromUserID, toUserID, models.EngagementKindMatched, matched); err != nil {
			return err
		}
		if err := setCanonicalMatchDetail(tx, toAggregate, fromAggregate, toUserID, fromUserID, models.EngagementKindMatched, matched); err != nil {
			return err
		}
		if err := syncMatchingAggregateCounts(tx, fromAggregate); err != nil {
			return err
		}
		return syncMatchingAggregateCounts(tx, toAggregate)
	})
	if err != nil {
		return false, err
	}

	if newMatch {
		m.notifyNewMatch(fromUserID, toUserID, identities)
	}
	return matched, nil
}

func (m *MatchesRepository) notifyNewMatch(fromUserID, toUserID uuid.UUID, identities map[uuid.UUID]matchUserIdentity) {
	if m.notificationRepo == nil {
		return
	}
	fromUser, fromOK := identities[fromUserID]
	toUser, toOK := identities[toUserID]
	if !fromOK || !toOK {
		return
	}

	for _, item := range []struct {
		sender, receiver uuid.UUID
		matchedPublicID  int64
	}{
		{sender: toUserID, receiver: fromUserID, matchedPublicID: toUser.PublicID},
		{sender: fromUserID, receiver: toUserID, matchedPublicID: fromUser.PublicID},
	} {
		payload := notifications.NotificationPayload{
			Data: map[string]string{"match_user_id": strconv.FormatInt(item.matchedPublicID, 10)},
		}
		if _, err := m.notificationRepo.CreateNotification(item.sender, item.receiver, string(notifications.NotificationTypeNewMatch), "It's a Match!", "You have a new match.", payload); err != nil {
			helpers.Println("Failed to create match notification for user", item.receiver, ":", err)
		}
	}
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

const matchBlockExclusionSQL = `NOT EXISTS (
	SELECT 1
	FROM engagement_details AS block_detail
	JOIN engagements AS block_aggregate
	  ON block_aggregate.id = block_detail.engagement_id
	 AND block_aggregate.contentable_type = 'user'
	WHERE block_detail.kind = 'blocking'
	  AND (
		(block_detail.engager_id = ? AND block_detail.engagee_id = users.id)
		OR
		(block_detail.engager_id = users.id AND block_detail.engagee_id = ?)
	  )
)`

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
		Where("users.domain = (SELECT auth_user.domain FROM users AS auth_user WHERE auth_user.id = ?)", userID).
		Where(matchBlockExclusionSQL, userID, userID)
	query = applyMatchPublicUserJoins(query)
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
) ([]types.PublicUserSummary, error) {

	limit = normalizedMatchLimit(limit)
	recentlySeen := m.db.
		Table("engagement_details AS seen_detail").
		Select("seen_detail.engagee_id").
		Joins("JOIN engagements AS seen_aggregate ON seen_aggregate.id = seen_detail.engagement_id AND seen_aggregate.contentable_type = ?", models.EngagementContentableTypeUser).
		Where("seen_detail.engager_id = ? AND seen_detail.created_at >= NOW() - INTERVAL '24 hours' AND seen_detail.kind = ?", userID, models.EngagementKindViewGiven)

	query := m.db.WithContext(ctx).
		Table("users").
		Select(unseenPublicUserProjection).
		Where("users.id != ?", userID).
		Where("users.id NOT IN (?)", recentlySeen).
		Where("users.domain = (SELECT auth_user.domain FROM users AS auth_user WHERE auth_user.id = ?)", userID).
		Where(matchBlockExclusionSQL, userID, userID)
	query = applyMatchPublicUserJoins(query)
	query = publicUserVisibilityScope(query)

	var rows []publicUserProjectionRow
	if err := query.Order("users.public_id ASC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	users := make([]types.PublicUserSummary, 0, len(rows))
	for index := range rows {
		users = append(users, rows[index].summaryView())
	}
	return users, nil
}

func (m *MatchesRepository) GetPassesAfter(
	ctx context.Context,
	userID uuid.UUID,
	cursor *types.MatchListCursor,
	limit int,
) (types.MatchListPage, error) {
	return m.matchListPage(ctx, userID, models.EngagementKindDislikeGiven, cursor, limit)
}
