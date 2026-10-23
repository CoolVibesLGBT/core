package repositories

import (
	"context"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// recordCanonicalUserView is shared by profile visits and swipes. The caller
// holds both user aggregate locks. Legacy profile visits stored the received
// direction backwards; only repair those on the target's own aggregate so an
// independent visit in the opposite direction is preserved.
func recordCanonicalUserView(tx *gorm.DB, viewer, target *models.Engagement, refresh bool) (bool, error) {
	var existing []models.EngagementDetail
	given := tx.Where("engagement_id IN ? AND kind = ? AND engager_id = ? AND engagee_id = ?",
		[]any{viewer.ID, target.ID}, models.EngagementKindViewGiven, viewer.ContentableID, target.ContentableID)
	received := tx.Where("engagement_id = ? AND kind = ?", target.ID, models.EngagementKindViewReceived).
		Where(tx.Where("engager_id = ? AND engagee_id = ?", target.ContentableID, viewer.ContentableID).
			Or("engager_id = ? AND engagee_id = ?", viewer.ContentableID, target.ContentableID))
	if err := tx.Where(given.Or(received)).Find(&existing).Error; err != nil {
		return false, err
	}
	now := time.Now().UTC()
	createdAt, updatedAt := now, now
	if len(existing) > 0 {
		createdAt, updatedAt = existing[0].CreatedAt, existing[0].UpdatedAt
		for _, detail := range existing[1:] {
			if detail.CreatedAt.Before(createdAt) {
				createdAt = detail.CreatedAt
			}
			if detail.UpdatedAt.After(updatedAt) {
				updatedAt = detail.UpdatedAt
			}
		}
	}
	if refresh {
		updatedAt = now
	}
	if err := tx.Where("engagement_id = ? AND kind = ? AND engager_id = ? AND engagee_id = ?",
		target.ID, models.EngagementKindViewReceived, viewer.ContentableID, target.ContentableID).
		Delete(&models.EngagementDetail{}).Error; err != nil {
		return false, err
	}
	for _, direction := range []struct {
		owner, other *models.Engagement
		kind         models.EngagementKind
	}{
		{viewer, target, models.EngagementKindViewGiven}, {target, viewer, models.EngagementKindViewReceived},
	} {
		if err := setCanonicalMatchDetail(tx, direction.owner, direction.other,
			direction.owner.ContentableID, direction.other.ContentableID, direction.kind, true); err != nil {
			return false, err
		}
		if err := tx.Model(&models.EngagementDetail{}).
			Where("engagement_id = ? AND kind = ? AND engager_id = ? AND engagee_id = ?",
				direction.owner.ID, direction.kind, direction.owner.ContentableID, direction.other.ContentableID).
			Updates(map[string]any{"created_at": createdAt, "updated_at": updatedAt}).Error; err != nil {
			return false, err
		}
	}
	for _, aggregate := range []*models.Engagement{viewer, target} {
		if err := syncUserAggregateKinds(tx, aggregate, []models.EngagementKind{models.EngagementKindViewGiven, models.EngagementKindViewReceived}); err != nil {
			return false, err
		}
	}
	return len(existing) == 0, nil
}

func (r *EngagementRepository) recordProfileViewOnce(ctx context.Context, viewerID, targetID uuid.UUID) (bool, error) {
	if r == nil || r.db == nil {
		return false, constants.ErrEngagementRepositoryNotConfigured
	}
	if r.db.Name() != "postgres" {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}
	counted := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockUserEngagementAggregates(tx, viewerID, targetID); err != nil {
			return err
		}
		if err := ensureSocialInteractionAllowed(tx, viewerID, targetID, domainuser.InteractionFollow); err != nil {
			return err
		}
		var visible int64
		if err := profileViewerScope(tx.Model(&models.User{}), viewerID).Where("users.id = ?", targetID).Count(&visible).Error; err != nil {
			return err
		}
		if visible == 0 {
			return domainuser.ErrInteractionUnavailable
		}
		viewer, err := loadOrCreateEngagementAggregate(tx, viewerID, models.EngagementContentableTypeUser)
		if err != nil {
			return err
		}
		target, err := loadOrCreateEngagementAggregate(tx, targetID, models.EngagementContentableTypeUser)
		if err != nil {
			return err
		}
		counted, err = recordCanonicalUserView(tx, viewer, target, false)
		if err != nil || !counted {
			return err
		}
		return enqueueProfileVisitNotification(tx, viewerID, targetID)
	})
	return counted, err
}
