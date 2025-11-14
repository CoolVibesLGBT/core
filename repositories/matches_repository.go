package repositories

import (
	"context"
	"time"

	"coolvibes/models"
	"coolvibes/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchesRepository struct {
	db             *gorm.DB
	engagementRepo *EngagementRepository
}

func NewMatchesRepository(db *gorm.DB, engagementRepo *EngagementRepository) *MatchesRepository {
	return &MatchesRepository{
		db:             db,
		engagementRepo: engagementRepo,
	}
}

func (m *MatchesRepository) RecordView(ctx context.Context, fromUserId uuid.UUID, toUserId uuid.UUID, reaction types.ReactionType) (bool, error) {

	var kindGiven models.EngagementKind
	var kindReceived models.EngagementKind

	switch reaction {
	case types.ReactionLike:
		kindGiven = models.EngagementKindLikeGiven
		kindReceived = models.EngagementKindLikeReceived

	case types.ReactionDislike:
		kindGiven = models.EngagementKindDislikeGiven
		kindReceived = models.EngagementKindDisLikeReceived

	default:
		return false, nil
	}
	kindMatched := models.EngagementKindMatched

	recipientId := toUserId
	engagerId := fromUserId

	_, err := m.addEngagementPair(ctx, recipientId, engagerId, kindGiven)
	if err != nil {
		return false, err
	}

	_, err = m.addEngagementPair(ctx, engagerId, recipientId, kindReceived)
	if err != nil {
		return false, err
	}

	isMatched, err := m.IsMatched(ctx, fromUserId, toUserId)
	if err != nil {
		return false, err
	}

	if isMatched {
		_, err := m.addEngagementPair(ctx, recipientId, engagerId, kindMatched)
		if err != nil {
			return false, err
		}

		_, err = m.addEngagementPair(ctx, engagerId, recipientId, kindMatched)
		if err != nil {
			return false, err
		}
	}

	_, err = m.addEngagementPair(ctx, recipientId, engagerId, models.EngagementKindViewGiven)
	if err != nil {
		return false, err
	}

	_, err = m.addEngagementPair(ctx, engagerId, recipientId, models.EngagementKindViewReceived)
	if err != nil {
		return false, err
	}

	return isMatched, nil
}

// -----------------------------------------------
// 🔥 2) Match = iki taraf da LIKE göndermişse
// -----------------------------------------------

func (m *MatchesRepository) IsMatched(ctx context.Context, fromUserId, toUserId uuid.UUID) (bool, error) {

	a, err := m.engagementRepo.HasUserEngaged(ctx, fromUserId, toUserId, models.EngagementKindLikeGiven)
	if err != nil {
		return false, err
	}

	b, err := m.engagementRepo.HasUserEngaged(ctx, toUserId, fromUserId, models.EngagementKindLikeGiven)
	if err != nil {
		return false, err
	}

	return a && b, nil
}

// -----------------------------------------------
// 🔥 Engagement çiftlerini ekler
// userID → targetID
// -----------------------------------------------

func (m *MatchesRepository) addEngagementPair(ctx context.Context, recipientID, engagerID uuid.UUID, kind models.EngagementKind) (bool, error) {
	status, err := m.engagementRepo.ToggleEngagement(ctx, recipientID, engagerID, kind, engagerID, "user")
	if err != nil {
		return status, err
	}

	return true, err
}

// -----------------------------------------------
// 🔥 Engagement çiftlerini siler (toggle off)
// -----------------------------------------------

func (m *MatchesRepository) removeEngagementPair(
	ctx context.Context,
	userID, targetID uuid.UUID,
	kindGiven, kindReceived models.EngagementKind,
) error {

	// GIVEN sil
	_, err := m.engagementRepo.ToggleEngagement(ctx,
		targetID, userID, kindGiven,
		targetID, "user",
	)
	if err != nil {
		return err
	}

	// RECEIVED sil
	_, err = m.engagementRepo.ToggleEngagement(ctx,
		userID, targetID, kindReceived,
		userID, "user",
	)
	return err
}

// -----------------------------------------------
// 🔍 Son X saat içinde target’ı gördü mü?
// -----------------------------------------------

func (m *MatchesRepository) WasSeenRecently(
	ctx context.Context,
	userID, targetID uuid.UUID,
	hours int,
) (bool, error) {

	var count int64
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	err := m.db.WithContext(ctx).
		Model(&models.EngagementDetail{}).
		Where(`
			engager_id = ? AND recipient_id = ? 
			AND created_at >= ? 
			AND kind IN ('like_given','dislike_given')
		`, userID, targetID, since).
		Count(&count).Error

	return count > 0, err
}

// -----------------------------------------------
// 🔍 Beğendiği kullanıcılar (cursor destekli)
// -----------------------------------------------

func (m *MatchesRepository) GetLikesAfter(
	ctx context.Context,
	userID uuid.UUID,
	cursor *time.Time,
	limit int,
) ([]models.User, error) {

	var users []models.User

	q := m.db.WithContext(ctx).
		Model(&models.EngagementDetail{}).
		Select("users.*").
		Joins("JOIN users ON users.id = engagement_details.recipient_id").
		Where("engager_id = ? AND kind = ?", userID, models.EngagementKindLikeGiven)

	if cursor != nil {
		q = q.Where("engagement_details.created_at < ?", *cursor)
	}

	err := q.
		Order("engagement_details.created_at DESC").
		Limit(limit).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Scan(&users).Error

	return users, err
}

// -----------------------------------------------
// 🔍 MATCH olmuş kullanıcılar
// -----------------------------------------------

func (m *MatchesRepository) GetMatchesAfter(
	ctx context.Context,
	userID uuid.UUID,
	cursor *time.Time,
	limit int,
) ([]models.User, error) {

	// Karşılıklı beğenme (match)
	sub := m.db.
		Table("engagement_details AS a").
		Select("a.recipient_id").
		Joins(`
			INNER JOIN engagement_details b 
			ON a.recipient_id = b.engager_id 
			AND a.engager_id = b.recipient_id
			AND a.kind = 'like_given'
			AND b.kind = 'like_given'
		`).
		Where("a.engager_id = ?", userID)

	if cursor != nil {
		sub = sub.Where("a.created_at < ?", *cursor)
	}

	var users []models.User
	err := m.db.
		Model(&models.User{}).
		Where("id IN (?)", sub).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Order("created_at DESC").
		Limit(limit).
		Find(&users).Error

	return users, err
}

// -----------------------------------------------
// 🔍 Geçen 24 saatte görmediğin kullanıcılar
// -----------------------------------------------

func (m *MatchesRepository) GetUnseenUsers(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]models.User, error) {

	sub := m.db.
		Table("engagement_details").
		Select("recipient_id").
		Where("engager_id = ? AND created_at >= NOW() - INTERVAL '24 hours' AND kind = ?", userID, models.EngagementKindViewGiven)

	var users []models.User
	err := m.db.
		Model(&models.User{}).
		Where("id != ?", userID).
		Where("id NOT IN (?)", sub).
		Where("deleted_at IS NULL").
		Order("RANDOM()").
		Limit(limit).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Find(&users).Error

	return users, err
}

func (m *MatchesRepository) GetPassesAfter(
	ctx context.Context,
	userID uuid.UUID,
	cursor *time.Time,
	limit int,
) ([]models.User, error) {

	// Öncelikle dislike edilen kullanıcıların ID'lerini engagement_details tablosundan alıyoruz
	subQuery := m.db.WithContext(ctx).
		Model(&models.EngagementDetail{}).
		Select("recipient_id").
		Where("engager_id = ?", userID).
		Where("kind = ?", models.EngagementKindDislikeGiven) // dislike türü
	if cursor != nil {
		subQuery = subQuery.Where("created_at < ?", *cursor)
	}

	// Ardından bu ID'lere göre kullanıcıları çekiyoruz
	var users []models.User
	err := m.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id IN (?)", subQuery).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Order("created_at DESC").
		Limit(limit).
		Find(&users).Error

	return users, err
}
