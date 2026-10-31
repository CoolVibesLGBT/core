package repositories

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	"core/helpers"
	"core/models"
	"core/models/notifications"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FriendshipRepository struct {
	db   *gorm.DB
	node *helpers.Node
}

func NewFriendshipRepository(db *gorm.DB, node *helpers.Node) *FriendshipRepository {
	return &FriendshipRepository{db: db, node: node}
}

func friendshipPair(first, second uuid.UUID) (uuid.UUID, uuid.UUID) {
	if first.String() > second.String() {
		return second, first
	}
	return first, second
}

// friendshipDedupeKey is always computed from the canonical pair order, so
// there is exactly one EngagementDetail row per user pair, enforced by the
// column's existing unique index — the same guarantee the dedicated
// friendships table used to get from uidx_friendship_pair.
func friendshipDedupeKey(first, second uuid.UUID) string {
	return "friendship:" + first.String() + ":" + second.String()
}

// friendshipDetails is the JSON payload carried in EngagementDetail.Details
// for the single row representing a user pair's relationship. The status
// and requester invariants this used to get from chk_friendship_status and
// chk_friendship_requester are enforced by raw SQL CHECK constraints on this
// JSON path (see EnsureFriendshipDetailConstraints) since GORM cannot
// express them through tags.
type friendshipDetails struct {
	Status      domainuser.FriendshipStatus `json:"status"`
	RequestedBy uuid.UUID                   `json:"requested_by"`
	RequestedAt time.Time                   `json:"requested_at"`
	PublicID    int64                       `json:"public_id,string"`
}

func decodeFriendshipDetails(raw datatypes.JSON) (friendshipDetails, error) {
	var details friendshipDetails
	if len(raw) == 0 {
		return details, nil
	}
	err := json.Unmarshal(raw, &details)
	return details, err
}

// findFriendshipDetail loads the pair's single relationship row by its
// canonical dedupe key, or nil if the pair has never requested friendship.
func findFriendshipDetail(tx *gorm.DB, first, second uuid.UUID, lock bool) (*models.EngagementDetail, error) {
	query := tx.Where("dedupe_key = ?", friendshipDedupeKey(first, second))
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row models.EngagementDetail
	err := query.Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *FriendshipRepository) FindFriendship(ctx context.Context, actor uuid.UUID, targetPublicID int64) (*ports.FriendshipRecord, error) {
	var target models.User
	query := r.db.WithContext(ctx).Model(&models.User{}).Table("users").Select("users.id").Where("users.public_id = ? AND users.deleted_at IS NULL", targetPublicID).
		Where("users.domain = (?)", userDomainSubquery(r.db, actor)).Where("users.user_role NOT IN ?", []string{"banned", "deleted", "pending"})
	if err := userBlockScope(query, actor, "users").Take(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	first, second := friendshipPair(actor, target.ID)
	row, err := findFriendshipDetail(r.db.WithContext(ctx), first, second, false)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	rows, err := r.friendshipRecords(ctx, []models.EngagementDetail{*row}, actor)
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func (r *FriendshipRepository) RequestFriendship(ctx context.Context, actor uuid.UUID, targetPublicID int64) (ports.FriendshipRecord, error) {
	var target models.User
	if err := r.db.WithContext(ctx).Select("id").Where("public_id = ?", targetPublicID).Take(&target).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.FriendshipRecord{}, ports.ErrNotFound
		}
		return ports.FriendshipRecord{}, err
	}
	if actor == target.ID {
		return ports.FriendshipRecord{}, domainuser.ErrSelfInteraction
	}
	first, second := friendshipPair(actor, target.ID)
	var row models.EngagementDetail
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, first, second); err != nil {
			return err
		}
		if err := lockUserEngagementAggregates(tx, first, second); err != nil {
			return err
		}
		if err := ensureSocialInteractionAllowed(tx, actor, target.ID, domainuser.InteractionFollow); err != nil {
			return err
		}
		existing, err := findFriendshipDetail(tx, first, second, true)
		if err != nil {
			return err
		}
		if existing != nil {
			existingDetails, err := decodeFriendshipDetails(existing.Details)
			if err != nil {
				return err
			}
			if existingDetails.Status == domainuser.FriendshipPending || existingDetails.Status == domainuser.FriendshipAccepted {
				row = *existing
				return nil
			}
		}
		if r.node == nil {
			return constants.ErrFriendshipIDGeneratorRequired
		}
		now := time.Now().UTC()
		details := friendshipDetails{
			Status: domainuser.FriendshipPending, RequestedBy: actor,
			RequestedAt: now, PublicID: r.node.Generate().Int64(),
		}
		payload, err := json.Marshal(details)
		if err != nil {
			return err
		}
		if existing != nil {
			row = *existing
			row.Details, row.UpdatedAt = datatypes.JSON(payload), now
			if err := tx.Model(&models.EngagementDetail{}).Where("id = ?", row.ID).
				Updates(map[string]any{"details": row.Details, "updated_at": row.UpdatedAt}).Error; err != nil {
				return err
			}
		} else {
			aggregate, err := loadOrCreateEngagementAggregate(tx, first, models.EngagementContentableTypeUser)
			if err != nil {
				return err
			}
			dedupeKey := friendshipDedupeKey(first, second)
			row = models.EngagementDetail{
				ID: uuid.New(), EngagementID: aggregate.ID, DedupeKey: &dedupeKey,
				EngagerID: first, EngageeID: second, Kind: models.EngagementKindFriendship,
				Details: datatypes.JSON(payload), CreatedAt: now, UpdatedAt: now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		notifPayload := notifications.NotificationPayload{Data: map[string]string{"request_id": strconv.FormatInt(details.PublicID, 10), "status": string(domainuser.FriendshipPending)}}
		_, err = persistQueuedNotification(tx, actor, target.ID, notifications.NotificationTypeFriendRequest, "Friend request", "You have a new friend request.", notifPayload)
		return err
	})
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	rows, err := r.friendshipRecords(ctx, []models.EngagementDetail{row}, actor)
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	return rows[0], nil
}

func (r *FriendshipRepository) ChangeFriendship(ctx context.Context, actor uuid.UUID, requestID int64, desired domainuser.FriendshipStatus) (ports.FriendshipRecord, error) {
	var row models.EngagementDetail
	if err := r.db.WithContext(ctx).Model(&models.EngagementDetail{}).
		Where("kind = ? AND details->>'public_id' = ? AND (engager_id = ? OR engagee_id = ?)",
			models.EngagementKindFriendship, strconv.FormatInt(requestID, 10), actor, actor).
		Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.FriendshipRecord{}, ports.ErrNotFound
		}
		return ports.FriendshipRecord{}, err
	}
	first, second := row.EngagerID, row.EngageeID

	var details friendshipDetails
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, first, second); err != nil {
			return err
		}
		if err := lockUserEngagementAggregates(tx, first, second); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", row.ID).Take(&row).Error; err != nil {
			return err
		}
		var err error
		details, err = decodeFriendshipDetails(row.Details)
		if err != nil {
			return err
		}
		if details.PublicID != requestID {
			return ports.ErrNotFound
		}
		other := first
		if actor == first {
			other = second
		}
		if desired == domainuser.FriendshipAccepted {
			if err := ensureSocialInteractionAllowed(tx, actor, other, domainuser.InteractionFollow); err != nil {
				return err
			}
		}
		next, changed, err := domainuser.DecideFriendship(details.Status, desired, details.RequestedBy == actor)
		if err != nil || !changed {
			return err
		}
		details.Status = next
		now := time.Now().UTC()
		payload, err := json.Marshal(details)
		if err != nil {
			return err
		}
		if err := tx.Model(&models.EngagementDetail{}).Where("id = ?", row.ID).
			Updates(map[string]any{"details": datatypes.JSON(payload), "updated_at": now}).Error; err != nil {
			return err
		}
		row.Details, row.UpdatedAt = datatypes.JSON(payload), now
		if err := resolveFriendshipNotification(tx, details.PublicID, details.Status); err != nil {
			return err
		}
		if next == domainuser.FriendshipAccepted {
			notifPayload := notifications.NotificationPayload{Data: map[string]string{"request_id": strconv.FormatInt(details.PublicID, 10)}}
			_, err := persistQueuedNotification(tx, actor, other, notifications.NotificationTypeFriendRequestAccepted, "Friend request accepted", "Your friend request was accepted.", notifPayload)
			return err
		}
		return nil
	})
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	rows, err := r.friendshipRecords(ctx, []models.EngagementDetail{row}, actor)
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	return rows[0], nil
}

func resolveFriendshipNotification(tx *gorm.DB, publicID int64, status domainuser.FriendshipStatus) error {
	return tx.Model(&notifications.Notification{}).
		Where("type = ? AND payload->'data'->>'request_id' = ?", notifications.NotificationTypeFriendRequest, strconv.FormatInt(publicID, 10)).
		Updates(map[string]any{"is_read": true, "read_at": time.Now().UTC(), "payload": gorm.Expr("jsonb_set(payload,'{data,status}',to_jsonb(CAST(? AS text)),true)", string(status))}).Error
}

func (r *FriendshipRepository) ListFriendships(ctx context.Context, actor uuid.UUID, scope domainuser.FriendshipScope, cursor *types.MatchListCursor, limit int) ([]ports.FriendshipRecord, *types.MatchListCursor, error) {
	limit = normalizedMatchLimit(limit)
	query := r.db.WithContext(ctx).Model(&models.EngagementDetail{}).Select("engagement_details.*").
		Joins("JOIN users ON users.id = CASE WHEN engagement_details.engager_id = ? THEN engagement_details.engagee_id ELSE engagement_details.engager_id END", actor).
		Where("engagement_details.kind = ?", models.EngagementKindFriendship).
		Where("engagement_details.engager_id = ? OR engagement_details.engagee_id = ?", actor, actor).
		Where("users.deleted_at IS NULL AND users.is_bot = FALSE AND users.user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending}).
		Where("users.domain = (?)", userDomainSubquery(r.db, actor))
	query = userBlockScope(query, actor, "users")
	actorText := actor.String()
	switch scope {
	case domainuser.FriendshipScopeFriends:
		query = query.Where("engagement_details.details->>'status' = ?", string(domainuser.FriendshipAccepted))
	case domainuser.FriendshipScopeIncoming:
		query = query.Where("engagement_details.details->>'status' = ? AND engagement_details.details->>'requested_by' <> ?", string(domainuser.FriendshipPending), actorText)
	case domainuser.FriendshipScopeOutgoing:
		query = query.Where("engagement_details.details->>'status' = ? AND engagement_details.details->>'requested_by' = ?", string(domainuser.FriendshipPending), actorText)
	default:
		return nil, nil, domainuser.ErrFriendshipTransition
	}
	if cursor != nil {
		if cursor.DetailID == uuid.Nil {
			query = query.Where("engagement_details.updated_at < ?", cursor.OccurredAt)
		} else {
			query = query.Where("engagement_details.updated_at < ? OR (engagement_details.updated_at = ? AND engagement_details.id < ?)", cursor.OccurredAt, cursor.OccurredAt, cursor.DetailID)
		}
	}
	var rows []models.EngagementDetail
	if err := query.Order("engagement_details.updated_at DESC, engagement_details.id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	var next *types.MatchListCursor
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next = &types.MatchListCursor{OccurredAt: last.UpdatedAt, DetailID: last.ID}
	}
	records, err := r.friendshipRecords(ctx, rows, actor)
	return records, next, err
}

func (r *FriendshipRepository) friendshipRecords(ctx context.Context, rows []models.EngagementDetail, actor uuid.UUID) ([]ports.FriendshipRecord, error) {
	result := make([]ports.FriendshipRecord, 0, len(rows))
	if len(rows) == 0 {
		return result, nil
	}
	ids := make([]uuid.UUID, 0, len(rows)*2)
	for _, row := range rows {
		ids = append(ids, row.EngagerID, row.EngageeID)
	}
	var users []models.User
	if err := r.db.WithContext(ctx).Select("id", "public_id", "user_name", "display_name", "is_online", "avatar_id").
		Preload("Avatar", "is_public = TRUE").Preload("Avatar.File").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]models.User, len(users))
	for _, u := range users {
		byID[u.ID] = u
	}
	for _, row := range rows {
		details, err := decodeFriendshipDetails(row.Details)
		if err != nil {
			return nil, err
		}
		// Pair order (EngagerID/EngageeID) is a storage detail only. Request
		// direction comes exclusively from Details.RequestedBy and remains
		// stable for either participant's view.
		first, second := row.EngagerID, row.EngageeID
		if (details.RequestedBy != first && details.RequestedBy != second) ||
			(actor != first && actor != second) {
			return nil, domainuser.ErrFriendshipForbidden
		}
		recipientID := first
		if recipientID == details.RequestedBy {
			recipientID = second
		}
		requester, requesterExists := byID[details.RequestedBy]
		recipient, recipientExists := byID[recipientID]
		if !requesterExists || !recipientExists {
			return nil, ports.ErrNotFound
		}
		other := recipient
		if actor == recipientID {
			other = requester
		}
		projection := publicUserProjectionRow{
			PublicID: other.PublicID, UserName: other.UserName,
			DisplayName: other.DisplayName, IsOnline: other.IsOnline,
		}
		if other.Avatar != nil {
			url := other.Avatar.File.URL
			projection.AvatarURL = &url
			variants, err := json.Marshal(other.Avatar.File.Variants)
			if err != nil {
				return nil, err
			}
			projection.AvatarVariants = variants
		}
		result = append(result, ports.FriendshipRecord{
			PublicID:          details.PublicID,
			RequesterPublicID: requester.PublicID,
			RecipientPublicID: recipient.PublicID,
			Status:            details.Status,
			RequestedAt:       details.RequestedAt,
			UpdatedAt:         row.UpdatedAt,
			OtherUser:         projection.summaryView(),
		})
	}

	return result, nil
}
