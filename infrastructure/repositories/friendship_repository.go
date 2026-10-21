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
	var record models.Friendship
	if err := r.db.WithContext(ctx).Where("first_user_id = ? AND second_user_id = ?", first, second).Take(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	rows, err := r.friendshipRecords(ctx, []models.Friendship{record}, actor)
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
	var record models.Friendship
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
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("first_user_id = ? AND second_user_id = ?", first, second).Take(&record)
		if result.Error == nil && (record.Status == domainuser.FriendshipPending || record.Status == domainuser.FriendshipAccepted) {
			return nil
		}
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		now := time.Now().UTC()
		if record.ID == uuid.Nil {
			if r.node == nil {
				return constants.ErrFriendshipIDGeneratorRequired
			}
			record = models.Friendship{ID: uuid.New(), PublicID: r.node.Generate().Int64(), FirstUserID: first, SecondUserID: second, CreatedAt: now}
		} else {
			if r.node == nil {
				return constants.ErrFriendshipIDGeneratorRequired
			}
			record.PublicID = r.node.Generate().Int64()
		}
		record.RequestedByID, record.Status, record.RequestedAt, record.UpdatedAt = actor, domainuser.FriendshipPending, now, now
		if err := tx.Omit(clause.Associations).Save(&record).Error; err != nil {
			return err
		}
		payload := notifications.NotificationPayload{Data: map[string]string{"request_id": strconv.FormatInt(record.PublicID, 10), "status": string(domainuser.FriendshipPending)}}
		_, err := persistQueuedNotification(tx, actor, target.ID, notifications.NotificationTypeFriendRequest, "Friend request", "You have a new friend request.", payload)
		return err
	})
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	rows, err := r.friendshipRecords(ctx, []models.Friendship{record}, actor)
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	return rows[0], nil
}

func (r *FriendshipRepository) ChangeFriendship(ctx context.Context, actor uuid.UUID, requestID int64, desired domainuser.FriendshipStatus) (ports.FriendshipRecord, error) {
	var record models.Friendship
	if err := r.db.WithContext(ctx).Where("public_id = ? AND (first_user_id = ? OR second_user_id = ?)", requestID, actor, actor).Take(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.FriendshipRecord{}, ports.ErrNotFound
		}
		return ports.FriendshipRecord{}, err
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockPrivatePhotoAccessPair(tx, record.FirstUserID, record.SecondUserID); err != nil {
			return err
		}
		if err := lockUserEngagementAggregates(tx, record.FirstUserID, record.SecondUserID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", record.ID).Take(&record).Error; err != nil {
			return err
		}
		if record.PublicID != requestID {
			return ports.ErrNotFound
		}
		other := record.FirstUserID
		if other == actor {
			other = record.SecondUserID
		}
		if desired == domainuser.FriendshipAccepted {
			if err := ensureSocialInteractionAllowed(tx, actor, other, domainuser.InteractionFollow); err != nil {
				return err
			}
		}
		next, changed, err := domainuser.DecideFriendship(record.Status, desired, record.RequestedByID == actor)
		if err != nil || !changed {
			return err
		}
		record.Status, record.UpdatedAt = next, time.Now().UTC()
		if err := tx.Model(&record).Updates(map[string]any{"status": next, "updated_at": record.UpdatedAt}).Error; err != nil {
			return err
		}
		if err := resolveFriendshipNotification(tx, record); err != nil {
			return err
		}
		if next == domainuser.FriendshipAccepted {
			payload := notifications.NotificationPayload{Data: map[string]string{"request_id": strconv.FormatInt(record.PublicID, 10)}}
			_, err := persistQueuedNotification(tx, actor, other, notifications.NotificationTypeFriendRequestAccepted, "Friend request accepted", "Your friend request was accepted.", payload)
			return err
		}
		return nil
	})
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	rows, err := r.friendshipRecords(ctx, []models.Friendship{record}, actor)
	if err != nil {
		return ports.FriendshipRecord{}, err
	}
	return rows[0], nil
}

func resolveFriendshipNotification(tx *gorm.DB, record models.Friendship) error {
	return tx.Model(&notifications.Notification{}).
		Where("type = ? AND payload->'data'->>'request_id' = ?", notifications.NotificationTypeFriendRequest, strconv.FormatInt(record.PublicID, 10)).
		Updates(map[string]any{"is_read": true, "read_at": time.Now().UTC(), "payload": gorm.Expr("jsonb_set(payload,'{data,status}',to_jsonb(CAST(? AS text)),true)", string(record.Status))}).Error
}

func (r *FriendshipRepository) ListFriendships(ctx context.Context, actor uuid.UUID, scope domainuser.FriendshipScope, cursor *types.MatchListCursor, limit int) ([]ports.FriendshipRecord, *types.MatchListCursor, error) {
	limit = normalizedMatchLimit(limit)
	query := r.db.WithContext(ctx).Model(&models.Friendship{}).Select("friendships.*").
		Joins("JOIN users ON users.id = CASE WHEN friendships.first_user_id = ? THEN friendships.second_user_id ELSE friendships.first_user_id END", actor).
		Where("friendships.first_user_id = ? OR friendships.second_user_id = ?", actor, actor).
		Where("users.deleted_at IS NULL AND users.is_bot = FALSE AND users.user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending}).
		Where("users.domain = (?)", userDomainSubquery(r.db, actor))
	query = userBlockScope(query, actor, "users")
	switch scope {
	case domainuser.FriendshipScopeFriends:
		query = query.Where("friendships.status = ?", domainuser.FriendshipAccepted)
	case domainuser.FriendshipScopeIncoming:
		query = query.Where("friendships.status = ? AND friendships.requested_by_id <> ?", domainuser.FriendshipPending, actor)
	case domainuser.FriendshipScopeOutgoing:
		query = query.Where("friendships.status = ? AND friendships.requested_by_id = ?", domainuser.FriendshipPending, actor)
	default:
		return nil, nil, domainuser.ErrFriendshipTransition
	}
	if cursor != nil {
		if cursor.DetailID == uuid.Nil {
			query = query.Where("friendships.updated_at < ?", cursor.OccurredAt)
		} else {
			query = query.Where("friendships.updated_at < ? OR (friendships.updated_at = ? AND friendships.id < ?)", cursor.OccurredAt, cursor.OccurredAt, cursor.DetailID)
		}
	}
	var rows []models.Friendship
	if err := query.Order("friendships.updated_at DESC, friendships.id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
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

func (r *FriendshipRepository) friendshipRecords(ctx context.Context, rows []models.Friendship, actor uuid.UUID) ([]ports.FriendshipRecord, error) {
	result := make([]ports.FriendshipRecord, 0, len(rows))
	if len(rows) == 0 {
		return result, nil
	}
	ids := make([]uuid.UUID, 0, len(rows)*2)
	for _, row := range rows {
		ids = append(ids, row.FirstUserID, row.SecondUserID)
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
		// Pair order is a storage detail. Request direction comes exclusively
		// from RequestedByID and remains stable for either participant's view.
		if (row.RequestedByID != row.FirstUserID && row.RequestedByID != row.SecondUserID) ||
			(actor != row.FirstUserID && actor != row.SecondUserID) {
			return nil, domainuser.ErrFriendshipForbidden
		}
		recipientID := row.FirstUserID
		if recipientID == row.RequestedByID {
			recipientID = row.SecondUserID
		}
		requester, requesterExists := byID[row.RequestedByID]
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
			PublicID:          row.PublicID,
			RequesterPublicID: requester.PublicID,
			RecipientPublicID: recipient.PublicID,
			Status:            row.Status,
			RequestedAt:       row.RequestedAt,
			UpdatedAt:         row.UpdatedAt,
			OtherUser:         projection.summaryView(),
		})
	}

	return result, nil
}
