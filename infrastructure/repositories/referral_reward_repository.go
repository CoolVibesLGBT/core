package repositories

import (
	"context"
	"core/application/ports"
	"core/application/types"
	domainuser "core/domain/user"
	"core/models"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type ReferralRewardRepository struct{ db *gorm.DB }

func NewReferralRewardRepository(db *gorm.DB) *ReferralRewardRepository {
	return &ReferralRewardRepository{db: db}
}

func referralReviewer(tx *gorm.DB, actor uuid.UUID, admin bool) error {
	var user models.User
	if err := tx.Select("id", "user_role", "is_bot").Take(&user, "id = ?", actor).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domainuser.ErrReferralReviewForbidden
		}
		return err
	}
	if !domainuser.CanAuthenticate(domainuser.AccountRole(user.UserRole), user.IsBot) || (admin && !domainuser.CanReviewReferral(domainuser.AccountRole(user.UserRole))) {
		return domainuser.ErrReferralReviewForbidden
	}
	return nil
}
func referralViewQuery(tx *gorm.DB) *gorm.DB {
	return tx.Table("referral_rewards AS reward").Select("reward.id, reward.amount, reward.status, reward.created_at, reward.reviewed_at, referrer.public_id AS referrer_public_id, referred.public_id AS referred_public_id").Joins("JOIN users referrer ON referrer.id = reward.referrer_id").Joins("JOIN users referred ON referred.id = reward.referred_id")
}
func (r *ReferralRewardRepository) ListReferralRewards(ctx context.Context, actor uuid.UUID, queue bool, status domainuser.ReferralRewardStatus, cursor *types.MatchListCursor, limit int) ([]ports.ReferralRewardView, *types.MatchListCursor, error) {
	tx := r.db.WithContext(ctx)
	if err := referralReviewer(tx, actor, queue); err != nil {
		return nil, nil, err
	}
	q := referralViewQuery(tx)
	if !queue {
		q = q.Where("reward.referrer_id = ? OR reward.referred_id = ?", actor, actor)
	}
	if status != "" {
		q = q.Where("reward.status = ?", status)
	}
	if cursor != nil {
		q = q.Where("(reward.created_at, reward.id) < (?, ?)", cursor.OccurredAt, cursor.DetailID)
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	rows := make([]ports.ReferralRewardView, 0)
	if err := q.Order("reward.created_at DESC, reward.id DESC").Limit(limit + 1).Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	var next *types.MatchListCursor
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		next = &types.MatchListCursor{OccurredAt: last.CreatedAt, DetailID: last.ID}
	}
	return rows, next, nil
}
func (r *ReferralRewardRepository) ReviewReferralReward(ctx context.Context, actor, id uuid.UUID, review domainuser.ReferralReview) (ports.ReferralRewardView, error) {
	var result ports.ReferralRewardView
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := referralReviewer(tx, actor, true); err != nil {
			return err
		}
		if review.Status() != domainuser.ReferralRewardPaid && review.Status() != domainuser.ReferralRewardRejected {
			return domainuser.ErrReferralReviewInvalid
		}
		var initial models.ReferralReward
		if err := tx.Take(&initial, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrNotFound
			}
			return err
		}
		if actor == initial.ReferrerID || actor == initial.ReferredID {
			return domainuser.ErrReferralReviewForbidden
		}
		identity := review.IdentityHash()
		if tx.Name() == "postgres" {
			if identity != nil {
				if err := lockViewAggregate(tx, "referral-identity:"+*identity).Error; err != nil {
					return err
				}
			}
			if err := lockViewAggregate(tx, "referral:user-referral:"+initial.ReferredID.String()).Error; err != nil {
				return err
			}
		}
		var reward models.ReferralReward
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(&reward, "id = ?", id).Error; err != nil {
			return err
		}
		if reward.Status != domainuser.ReferralRewardPending {
			sameIdentity := (identity == nil && reward.VerifiedIdentityHash == nil) || (identity != nil && reward.VerifiedIdentityHash != nil && *identity == *reward.VerifiedIdentityHash)
			if reward.Status != review.Status() || !sameIdentity {
				return domainuser.ErrReferralReviewConflict
			}
			return referralViewQuery(tx).Where("reward.id = ?", id).Take(&result).Error
		}
		if identity != nil {
			var count int64
			if err := tx.Model(&models.ReferralReward{}).Where("verified_identity_hash = ? AND id <> ?", *identity, id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return domainuser.ErrReferralIdentityUsed
			}
		}
		referral, err := domainuser.NewReferral(reward.ReferrerID, reward.ReferredID, reward.Amount)
		if err != nil {
			return err
		}
		// Lock authority and participants in one global order. A concurrent
		// demotion cannot race an approval after this point.
		var people []models.User
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).Where("id IN ?", []uuid.UUID{actor, reward.ReferrerID, reward.ReferredID}).Order("id ASC").Find(&people).Error; err != nil {
			return err
		}
		var administrator *models.User
		for i := range people {
			if people[i].ID == actor {
				administrator = &people[i]
			}
		}
		if administrator == nil || administrator.DeletedAt.Valid || administrator.IsBot || !domainuser.CanReviewReferral(domainuser.AccountRole(administrator.UserRole)) {
			return domainuser.ErrReferralReviewForbidden
		}
		if review.Status() == domainuser.ReferralRewardPaid {
			referrer, err := lockReferralUsers(tx, referral)
			if err != nil {
				return err
			}
			if err := creditApprovedReferralInTransaction(tx, referral, referrer); err != nil {
				return err
			}
		}
		now := time.Now().UTC()
		if err := tx.Model(&reward).Updates(map[string]any{"status": review.Status(), "reviewed_by_id": actor, "reviewed_at": now, "review_note": review.Note(), "verified_identity_hash": identity}).Error; err != nil {
			return err
		}
		return referralViewQuery(tx).Where("reward.id = ?", id).Take(&result).Error
	})
	return result, err
}
