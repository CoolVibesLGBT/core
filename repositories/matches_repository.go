package repositories

import (
	"time"

	"bifrost/helpers"
	userModel "bifrost/models/user"
	"bifrost/types"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchesRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func (r *MatchesRepository) DB() *gorm.DB {
	return r.db
}

func NewMatchesRepository(db *gorm.DB, snowFlakeNode *helpers.Node) *MatchesRepository {
	return &MatchesRepository{db: db, snowFlakeNode: snowFlakeNode}
}

func (r *MatchesRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func (m *MatchesRepository) RecordView(userID, targetID uuid.UUID, reaction types.ReactionType) (bool, error) {
	var existing userModel.MatchSeen

	// 1️⃣ Önce mevcut kaydı kontrol et
	err := m.db.
		Where("user_id = ? AND target_id = ?", userID, targetID).
		First(&existing).Error

	switch err {
	case nil:
		// 2️⃣ Kayıt varsa sadece güncelle
		existing.Reaction = string(reaction)
		existing.CreatedAt = time.Now()

		if err := m.db.Save(&existing).Error; err != nil {
			return false, err
		}
	case gorm.ErrRecordNotFound:
		// 3️⃣ Yoksa yeni kayıt oluştur
		entry := userModel.MatchSeen{
			UserID:   userID,
			TargetID: targetID,
			Reaction: string(reaction),
		}
		if err := m.db.Create(&entry).Error; err != nil {
			return false, err
		}
	default:
		// 4️⃣ Diğer DB hataları
		return false, err
	}

	// 5️⃣ Match kontrolü
	isMatched, err := m.IsMatched(userID, targetID, types.ReactionLike)
	if err != nil {
		return false, err
	}

	if isMatched {
		// Her iki taraf da LIKE ettiyse, IsMatch güncellenir
		err = m.db.Model(&userModel.MatchSeen{}).
			Where("(user_id = ? AND target_id = ?) OR (user_id = ? AND target_id = ?)",
				userID, targetID, targetID, userID).
			Updates(map[string]interface{}{
				"is_match": true,
				"reaction": string(types.ReactionMatched), // ya da istediğin reaction tipi
			}).Error
		if err != nil {
			return false, err
		}
	}

	return isMatched, nil
}

func (m *MatchesRepository) IsMatched(userID1, userID2 uuid.UUID, reaction types.ReactionType) (bool, error) {
	var count1 int64
	err := m.db.Model(&userModel.MatchSeen{}).
		Where("user_id = ? AND target_id = ? AND reaction = ?", userID1, userID2, reaction).
		Count(&count1).Error
	if err != nil {
		return false, err
	}

	var count2 int64
	err = m.db.Model(&userModel.MatchSeen{}).
		Where("user_id = ? AND target_id = ? AND reaction = ?", userID2, userID1, reaction).
		Count(&count2).Error
	if err != nil {
		return false, err
	}

	// Her iki taraf da like etmişse eşleşme var
	return count1 > 0 && count2 > 0, nil
}

// ⏱ Son X saat içinde gösterilmiş mi?
func (m *MatchesRepository) WasSeenRecently(userID, targetID uuid.UUID, hours int) (bool, error) {
	var count int64
	err := m.db.Model(&userModel.MatchSeen{}).
		Where("user_id = ? AND target_id = ? AND created_at >= ?", userID, targetID, time.Now().Add(-time.Duration(hours)*time.Hour)).
		Count(&count).Error
	return count > 0, err
}

// 💾 Geçmiş (history) – kullanıcının son gördükleri
func (m *MatchesRepository) GetSeenHistory(userID uuid.UUID, limit int) ([]userModel.MatchSeen, error) {
	var seen []userModel.MatchSeen
	err := m.db.
		Preload("Target").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&seen).Error
	return seen, err
}

func (m *MatchesRepository) GetMatchesAfter(userID uuid.UUID, cursor *time.Time, limit int) ([]userModel.User, error) {
	var matchedUsers []userModel.User

	// Alt sorgu: kullanıcının match olduğu diğer user_id'leri bul
	subQuery := m.db.
		Table("match_seens").
		Select(`
			CASE 
				WHEN user_id = ? THEN target_id 
				ELSE user_id 
			END
		`, userID).
		Where("(user_id = ? OR target_id = ?) AND is_match = TRUE", userID, userID)

	if cursor != nil {
		subQuery = subQuery.Where("created_at < ?", *cursor)
	}

	// Ana sorgu: bu ID’lerdeki kullanıcıları preload’lu getir
	err := m.db.
		Model(&userModel.User{}).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Preload("Fantasies.Fantasy").
		Preload("Interests.InterestItem.Interest").
		Preload("GenderIdentities").
		Preload("SexualOrientations").
		Preload("SexualRole").
		Preload("UserAttributes.Attribute").
		Where("id IN (?)", subQuery).
		Order("created_at DESC").
		Limit(limit).
		Find(&matchedUsers).Error

	return matchedUsers, err
}

func (m *MatchesRepository) GetLikesAfter(userID uuid.UUID, cursor *time.Time, limit int) ([]userModel.User, error) {
	var targetIDs []uuid.UUID

	// Önce beğenilen kullanıcıların ID'lerini al
	query := m.db.Model(&userModel.MatchSeen{}).
		Where("user_id = ?", userID).
		Where("reaction = ?", string(types.ReactionLike)).
		Where("target_id != ?", userID)

	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Pluck("target_id", &targetIDs).Error
	if err != nil {
		return nil, err
	}

	// Sonra bu kullanıcıları çek
	var users []userModel.User
	err = m.db.Model(&userModel.User{}).
		Where("id IN ?", targetIDs).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Preload("Fantasies.Fantasy").
		Preload("Interests.InterestItem.Interest").
		Preload("GenderIdentities").
		Preload("SexualOrientations").
		Preload("SexualRole").
		Preload("UserAttributes.Attribute").
		Find(&users).Error

	return users, err
}

func (m *MatchesRepository) GetPassesAfter(userID uuid.UUID, cursor *time.Time, limit int) ([]userModel.User, error) {
	var targetIDs []uuid.UUID

	// Pas verdiğin kullanıcıların target_id'lerini çek
	query := m.db.
		Model(&userModel.MatchSeen{}).
		Where("user_id = ?", userID).
		Where("reaction = ?", string(types.ReactionDislike)).
		Where("target_id != ?", userID)

	if cursor != nil {
		query = query.Where("created_at < ?", *cursor)
	}

	err := query.
		Order("created_at DESC").
		Limit(limit).
		Pluck("target_id", &targetIDs).Error
	if err != nil {
		return nil, err
	}

	// TargetID'lere göre user kayıtlarını çek
	var users []userModel.User
	err = m.db.
		Model(&userModel.User{}).
		Where("id IN ?", targetIDs).
		Preload("Location").
		Preload("Avatar.File").
		Preload("Cover.File").
		Preload("Fantasies.Fantasy").
		Preload("Interests.InterestItem.Interest").
		Preload("GenderIdentities").
		Preload("SexualOrientations").
		Preload("SexualRole").
		Preload("UserAttributes.Attribute").
		Find(&users).Error

	return users, err
}

func (m *MatchesRepository) GetUnseenUsers(userID uuid.UUID, limit int) ([]userModel.User, error) {
	var users []userModel.User

	subQuery := m.db.Model(&userModel.MatchSeen{}).
		Select("target_id").
		Where("user_id = ? AND created_at >= NOW() - INTERVAL '24 hours'", userID)

	err := m.db.Model(&userModel.User{}).
		Preload("Location").
		Preload("Avatar").
		Preload("Avatar.File").
		Preload("Cover").
		Preload("Cover.File").
		Preload("Fantasies.Fantasy").
		Preload("Interests.InterestItem.Interest").
		Preload("Avatar.File").
		Preload("Cover.File").
		Preload("GenderIdentities").
		Preload("SexualOrientations").
		Preload("SexualRole").
		Preload("UserAttributes.Attribute").
		Where("id != ?", userID).
		Where("id NOT IN (?)", subQuery).
		Where("deleted_at IS NULL").
		Order("RANDOM()").
		Limit(limit).
		Find(&users).Error

	return users, err
}
