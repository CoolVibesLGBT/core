package repositories

import (
	"core/application/types"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"core/models/notifications"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// togglePostInteraction keeps the reaction, its counter and both durable
// notification jobs in one transaction. Lock users before the post aggregate
// so blocking and post interactions use the same lock order.
func (r *PostRepository) togglePostInteraction(filters types.Filter, kind models.EngagementKind) error {
	if filters.AuthUser == nil || filters.AuthUser.ID == uuid.Nil {
		return constants.ErrUserUnauthorized
	}
	if r.db.Name() != "postgres" {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}
	return r.db.WithContext(filters.Context).Transaction(func(tx *gorm.DB) error {
		p, err := (&PostRepository{db: tx}).FindPostByPublicID(filters.Context, filters.PostID)
		if err != nil {
			return err
		}
		actor := filters.AuthUser.ID
		if err := lockUserEngagementAggregates(tx, actor, p.AuthorID); err != nil {
			return err
		}
		if actor != p.AuthorID {
			if err := ensureSocialInteractionAllowed(tx, actor, p.AuthorID, domainuser.InteractionFollow); err != nil {
				return err
			}
		}
		if err := lockViewAggregate(tx, engagementAggregateLockKey(models.EngagementContentableTypePost, p.ID)).Error; err != nil {
			return err
		}
		aggregate, err := loadOrCreateEngagementAggregate(tx, p.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}
		current, err := reciprocalInteractionDetailExists(tx, aggregate, actor, p.AuthorID, kind)
		if err != nil {
			return err
		}
		enabled := !current
		if err := setReciprocalInteractionDetail(tx, aggregate, actor, p.AuthorID, kind,
			fmt.Sprintf("post-interaction:%s:%s:%s", kind, p.ID, actor), enabled); err != nil {
			return err
		}
		opposite := models.EngagementKind("")
		if kind == models.EngagementKindLikeReceived {
			opposite = models.EngagementKindDisLikeReceived
		}
		if kind == models.EngagementKindDisLikeReceived {
			opposite = models.EngagementKindLikeReceived
		}
		if enabled && opposite != "" {
			if err := setReciprocalInteractionDetail(tx, aggregate, actor, p.AuthorID, opposite, "", false); err != nil {
				return err
			}
		}
		// Recount both reactions to repair counters left by legacy duplicate rows.
		for _, countKind := range []models.EngagementKind{kind, opposite} {
			if countKind != "" {
				if err := syncEngagementKindCount(tx, aggregate.ID, countKind); err != nil {
					return err
				}
			}
		}
		notificationType, ownerPayload, actorPayload, err := postInteractionMessages(kind, enabled)
		if err != nil {
			return err
		}
		for index, entry := range []struct {
			sender, receiver uuid.UUID
			payload          notifications.NotificationPayload
		}{
			{actor, p.AuthorID, ownerPayload}, {p.AuthorID, actor, actorPayload},
		} {
			// Keep only the actor confirmation when the actor owns the post.
			if actor == p.AuthorID && index == 0 {
				continue
			}
			entry.payload.Data = map[string]any{"post_id": strconv.FormatInt(p.PublicID, 10), "engagement_kind": string(kind), "enabled": enabled}
			if _, err := persistQueuedNotification(tx, entry.sender, entry.receiver, notificationType,
				entry.payload.Title, entry.payload.Body, entry.payload); err != nil {
				return err
			}
		}
		return nil
	})
}

func postInteractionMessages(kind models.EngagementKind, enabled bool) (string, notifications.NotificationPayload, notifications.NotificationPayload, error) {
	var notificationType, removedType, ownerTitle, actorTitle, ownerBody, actorBody, removedTitle, removedOwnerBody, removedActorBody string
	switch kind {
	case models.EngagementKindLikeReceived:
		notificationType, removedType = notifications.NotificationTypeLike, notifications.NotificationTypeUnlike
		ownerTitle, actorTitle, ownerBody, actorBody = "New Like", "Liked", "Someone liked your post", "You liked this post"
		removedTitle, removedOwnerBody, removedActorBody = "Like Removed", "Someone removed their like", "You removed your like"
	case models.EngagementKindDisLikeReceived:
		notificationType, removedType = notifications.NotificationTypeDislike, notifications.NotificationTypeUndislike
		ownerTitle, actorTitle, ownerBody, actorBody = "New Dislike", "Disliked", "Someone disliked your post", "You disliked this post"
		removedTitle, removedOwnerBody, removedActorBody = "Dislike Removed", "Someone removed their dislike", "You removed your dislike"
	case models.EngagementKindBanana:
		notificationType, removedType = notifications.NotificationTypeBanana, notifications.NotificationTypeBananaRemoved
		ownerTitle, actorTitle, ownerBody, actorBody = "New Banana 🍌", "Banana Sent 🍌", "Someone sent you a banana", "You sent a banana"
		removedTitle, removedOwnerBody, removedActorBody = "Banana Removed", "Someone removed their banana", "You removed your banana"
	case models.EngagementKindBookmark:
		notificationType, removedType = notifications.NotificationTypeBookmark, notifications.NotificationTypeUnbookmark
		ownerTitle, actorTitle, ownerBody, actorBody = "New Bookmark", "Post Bookmarked", "Someone bookmarked your post", "You bookmarked this post"
		removedTitle, removedOwnerBody, removedActorBody = "Bookmark Removed", "Someone removed their bookmark", "You removed your bookmark"
	default:
		return "", notifications.NotificationPayload{}, notifications.NotificationPayload{}, constants.ErrUnknownEngagementKind.Format(kind)
	}
	if !enabled {
		notificationType, ownerTitle, actorTitle, ownerBody, actorBody = removedType, removedTitle, removedTitle, removedOwnerBody, removedActorBody
	}
	return notificationType, notifications.NotificationPayload{Title: ownerTitle, Body: ownerBody}, notifications.NotificationPayload{Title: actorTitle, Body: actorBody}, nil
}
