package repositories

import (
	"context"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// recordCanonicalUserView is shared by profile visits and swipes. The caller
// holds both user aggregate locks. A first view always counts; a repeat view
// only bumps its timestamp when refresh is requested (e.g. a swipe
// re-surfacing a profile), so a plain profile revisit under
// recordProfileViewOnce stays a one-time counter.
func recordCanonicalUserView(tx *gorm.DB, viewer, target *models.Engagement, refresh bool) (bool, error) {
	dedupeKey := matchDetailDedupeKey(models.EngagementKindViewGiven, viewer.ContentableID, target.ContentableID)
	var existing models.EngagementDetail
	err := tx.Where("dedupe_key = ?", dedupeKey).Take(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	isNewView := errors.Is(err, gorm.ErrRecordNotFound)

	now := time.Now().UTC()
	updatedAt := now
	if !isNewView && !refresh {
		updatedAt = existing.UpdatedAt
	}

	for _, direction := range []struct {
		owner, other *models.Engagement
		kind         models.EngagementKind
	}{
		{viewer, target, models.EngagementKindViewGiven}, {target, viewer, models.EngagementKindViewReceived},
	} {
		if err := setCanonicalMatchDetail(tx, direction.owner,
			direction.owner.ContentableID, direction.other.ContentableID, direction.kind, true); err != nil {
			return false, err
		}
		if !updatedAt.Equal(now) {
			if err := tx.Model(&models.EngagementDetail{}).
				Where("engagement_id = ? AND kind = ? AND engager_id = ? AND engagee_id = ?",
					direction.owner.ID, direction.kind, direction.owner.ContentableID, direction.other.ContentableID).
				Update("updated_at", updatedAt).Error; err != nil {
				return false, err
			}
		}
	}
	for _, aggregate := range []*models.Engagement{viewer, target} {
		if err := syncUserAggregateKinds(tx, aggregate, []models.EngagementKind{models.EngagementKindViewGiven, models.EngagementKindViewReceived}); err != nil {
			return false, err
		}
	}
	return isNewView, nil
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
