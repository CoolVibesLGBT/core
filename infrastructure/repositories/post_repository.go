package repositories

import (
	"context"
	legacyviews "core/application/legacyviews"
	"core/application/ports"
	"core/constants"
	domainpost "core/domain/post"
	domainuser "core/domain/user"
	domainwallet "core/domain/wallet"
	"core/extensions"
	"core/helpers"
	"core/models"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	"core/models/taxonomy"
	"core/models/utils"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"core/application/types"
	post_payloads "core/models/post/payloads"
	"sort"
	"sync"

	"fmt"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PostRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	notificationRepo *NotificationRepository
}

var fallbackEventRSVPLock sync.Mutex
var fallbackPollVoteLock sync.Mutex

func preloadPublicEngagementDetails(db *gorm.DB) *gorm.DB {
	// View aggregates are public counts, but the viewer identities are not.
	// Keep every existing actionable engagement detail while omitting post-view
	// detail rows (and therefore their Engager/Engagee associations) from feeds.
	return db.Where("kind <> ?", models.EngagementKindView)
}

func preloadEventAttendees(db *gorm.DB) *gorm.DB {
	return db.
		Select(`
			event_attendees.*,
			users.public_id AS user_public_id,
			users.user_name AS username,
			users.display_name AS displayname,
			COALESCE(
				NULLIF(file_metadata.variants #>> '{image,icon,url}', ''),
				NULLIF(file_metadata.variants #>> '{image,thumbnail,url}', ''),
				file_metadata.url
			) AS avatar_url
		`).
		Joins("LEFT JOIN users ON users.id = event_attendees.user_id AND users.deleted_at IS NULL").
		Joins("LEFT JOIN medias avatar_media ON avatar_media.id = users.avatar_id AND avatar_media.is_public = TRUE").
		Joins("LEFT JOIN file_metadata ON file_metadata.id = avatar_media.file_id").
		Where("event_attendees.status IN ?", []string{
			string(post_payloads.EventAttendanceGoing),
			string(post_payloads.EventAttendanceNotGoing),
			string(post_payloads.EventAttendanceMaybe),
			"interested",
			"declined",
		}).
		Order("event_attendees.joined_at DESC, event_attendees.id DESC")
}

func (r *PostRepository) DB() *gorm.DB {
	return r.db
}

func (r *PostRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewPostRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository) *PostRepository {
	return &PostRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo}
}

func (r *PostRepository) CreatePost(post *post.Post) error {
	if post.ID == uuid.Nil {
		post.ID = uuid.New()
	}

	// PublicID için Snowflake tarzı ID veya timestamp tabanlı basit artan ID
	if post.PublicID == 0 {
		post.PublicID = r.snowFlakeNode.Generate().Int64()
	}

	// CreatedAt ve UpdatedAt
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	// GORM ile kaydet
	if err := r.db.Create(post).Error; err != nil {
		return err
	}

	return nil
}

// CreatePoll polls ve seçeneklerini kaydeder
func (r *PostRepository) CreatePoll(poll *post_payloads.Poll) error {
	// Transaction başlat
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Poll kaydet
		if err := tx.Create(poll).Error; err != nil {
			return err
		}
		/*
			// PollChoice'ları kaydet
			for i := range poll.Choices {
				poll.Choices[i].PollID = poll.ID
				fmt.Println("ANKET SECIM", poll.Choices[i].Label, poll.Choices[i].ID, poll.ID)
				if err := tx.Create(&poll.Choices[i]).Error; err != nil {
					return err
				}
			}
		*/

		return nil
	})
}

func (r *PostRepository) CreateEvent(event *post_payloads.Event) error {
	return r.db.Create(event).Error
}

func lockEventRSVP(tx *gorm.DB, postPublicID int64) *gorm.DB {
	return tx.Exec(
		"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
		fmt.Sprintf("event-rsvp:%d", postPublicID),
	)
}

// SetEventRSVP serializes all RSVP writes for an event, repairs historical
// duplicate rows, and returns the canonical participant projection and counts.
// Passing nil clears the current user's RSVP; sending the same status again
// also toggles it off.
func (r *PostRepository) SetEventRSVP(
	ctx context.Context,
	postPublicID int64,
	userID uuid.UUID,
	desired *post_payloads.EventAttendanceStatus,
) (*post_payloads.EventRSVPResult, error) {
	if postPublicID <= 0 || userID == uuid.Nil {
		return nil, post_payloads.ErrEventNotFound
	}
	if desired != nil {
		status, ok := post_payloads.ParseEventAttendanceStatus(string(*desired))
		if !ok || status != *desired {
			return nil, fmt.Errorf("invalid event RSVP status: %s", *desired)
		}
	}

	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackEventRSVPLock.Lock()
		defer fallbackEventRSVPLock.Unlock()
	}

	var result post_payloads.EventRSVPResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isPostgres {
			if err := lockEventRSVP(tx, postPublicID).Error; err != nil {
				return err
			}
		}

		var event post_payloads.Event
		err := tx.Model(&post_payloads.Event{}).
			Select("events.*").
			Joins("JOIN posts ON posts.id = events.post_id").
			Where("posts.public_id = ? AND posts.published = TRUE AND posts.deleted_at IS NULL AND events.deleted_at IS NULL", postPublicID).
			Where("posts.post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
			Where("COALESCE(NULLIF(posts.audience, ''), 'public') = 'public'").
			First(&event).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return post_payloads.ErrEventNotFound
		}
		if err != nil {
			return err
		}
		if event.IsRSVPClosedAt(time.Now().UTC()) {
			return post_payloads.ErrEventClosed
		}

		var existingRows []post_payloads.EventAttendee
		if err := tx.
			Where("event_id = ?", event.ID).
			Order("joined_at DESC, id DESC").
			Find(&existingRows).Error; err != nil {
			return err
		}

		// Keep the newest row per user. The event-level lock guarantees that no
		// writer can insert another duplicate while this repair is in progress.
		seenUsers := make(map[uuid.UUID]struct{}, len(existingRows))
		uniqueRows := make([]post_payloads.EventAttendee, 0, len(existingRows))
		duplicateIDs := make([]uuid.UUID, 0)
		for _, attendee := range existingRows {
			if _, exists := seenUsers[attendee.UserID]; exists {
				duplicateIDs = append(duplicateIDs, attendee.ID)
				continue
			}
			seenUsers[attendee.UserID] = struct{}{}
			uniqueRows = append(uniqueRows, attendee)
		}
		if len(duplicateIDs) > 0 {
			if err := tx.Delete(&post_payloads.EventAttendee{}, "id IN ?", duplicateIDs).Error; err != nil {
				return err
			}
		}

		var current *post_payloads.EventAttendee
		for i := range uniqueRows {
			if uniqueRows[i].UserID == userID {
				current = &uniqueRows[i]
				break
			}
		}

		var nextStatus *post_payloads.EventAttendanceStatus
		currentStatus, currentStatusValid := post_payloads.EventAttendanceStatus(""), false
		if current != nil {
			currentStatus, currentStatusValid = post_payloads.ParseEventAttendanceStatus(string(current.Status))
		}

		shouldClear := desired == nil || (current != nil && currentStatusValid && currentStatus == *desired)
		if shouldClear {
			if current != nil {
				if err := tx.Delete(&post_payloads.EventAttendee{}, "id = ?", current.ID).Error; err != nil {
					return err
				}
			}
		} else {
			if *desired == post_payloads.EventAttendanceGoing && (current == nil || currentStatus != post_payloads.EventAttendanceGoing) && event.Capacity != nil && *event.Capacity > 0 {
				var goingCount int64
				if err := tx.Model(&post_payloads.EventAttendee{}).
					Where("event_id = ? AND status = ?", event.ID, post_payloads.EventAttendanceGoing).
					Count(&goingCount).Error; err != nil {
					return err
				}
				if goingCount >= int64(*event.Capacity) {
					return post_payloads.ErrEventAtCapacity
				}
			}

			if current == nil {
				attendee := post_payloads.EventAttendee{
					ID:       uuid.New(),
					EventID:  event.ID,
					UserID:   userID,
					Status:   *desired,
					JoinedAt: time.Now().UTC(),
				}
				if err := tx.Create(&attendee).Error; err != nil {
					return err
				}
			} else if err := tx.Model(&post_payloads.EventAttendee{}).
				Where("id = ?", current.ID).
				Updates(map[string]interface{}{"status": *desired, "updated_at": time.Now().UTC()}).Error; err != nil {
				return err
			}

			status := *desired
			nextStatus = &status
		}

		var rowsAfterWrite []post_payloads.EventAttendee
		if err := tx.Where("event_id = ?", event.ID).Find(&rowsAfterWrite).Error; err != nil {
			return err
		}
		counts := post_payloads.EventAttendanceCounts{}
		for _, attendee := range rowsAfterWrite {
			status, ok := post_payloads.ParseEventAttendanceStatus(string(attendee.Status))
			if !ok {
				continue
			}
			switch status {
			case post_payloads.EventAttendanceGoing:
				counts.Going++
			case post_payloads.EventAttendanceNotGoing:
				counts.NotGoing++
			case post_payloads.EventAttendanceMaybe:
				counts.Maybe++
			}
		}
		if err := tx.Model(&post_payloads.Event{}).
			Where("id = ?", event.ID).
			Updates(map[string]interface{}{
				"going_count":     counts.Going,
				"not_going_count": counts.NotGoing,
				"maybe_count":     counts.Maybe,
				"updated_at":      time.Now().UTC(),
			}).Error; err != nil {
			return err
		}

		var updatedEvent post_payloads.Event
		if err := tx.
			Preload("Attendees", preloadEventAttendees).
			First(&updatedEvent, "id = ?", event.ID).Error; err != nil {
			return err
		}
		result = post_payloads.EventRSVPResult{
			Status:    nextStatus,
			Attendees: updatedEvent.Attendees,
			Counts:    updatedEvent.AttendanceCounts(),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *PostRepository) GetPostByIDEx(id uuid.UUID) (*post.Post, error) {
	var p post.Post

	err := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Author").
		Preload("Tags").
		Preload("Attachments").
		Preload("Children").
		Preload("Children.Location").
		Preload("Children.Poll").
		Preload("Children.Poll.Choices").
		Preload("Children.Event").
		Preload("Children.Event.Location").
		Preload("Children.Event.Attendees", preloadEventAttendees).
		Preload("Children.Author").
		Preload("Children.Tags").
		Preload("Children.Attachments").
		First(&p, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("post with id %s not found", id)
		}
		return nil, err
	}

	return &p, nil
}

func (r *PostRepository) GetPostByID(id uuid.UUID) (*post.Post, error) {
	return r.getPostByID(id, false)
}

// GetPostByIDIncludingUnpublished is the explicit internal lookup used after
// creation and by chat workflows. Public handlers use GetPostByID instead.
func (r *PostRepository) GetPostByIDIncludingUnpublished(id uuid.UUID) (*post.Post, error) {
	return r.getPostByID(id, true)
}

func (r *PostRepository) getPostByID(id uuid.UUID, includeUnpublished bool) (*post.Post, error) {
	var ids []uuid.UUID
	publishedCondition := " AND published = TRUE AND deleted_at IS NULL AND post_kind NOT IN ('chat', 'message') AND COALESCE(NULLIF(audience, ''), 'public') = 'public'"
	childPublishedCondition := " AND p.published = TRUE AND p.post_kind NOT IN ('chat', 'message') AND COALESCE(NULLIF(p.audience, ''), 'public') = 'public'"
	if includeUnpublished {
		publishedCondition = " AND deleted_at IS NULL"
		childPublishedCondition = ""
	}
	cte := fmt.Sprintf(`
		WITH RECURSIVE post_tree AS (
			SELECT id
			FROM posts
			WHERE id = ?%s
			UNION ALL
			SELECT p.id
			FROM posts p
			INNER JOIN post_tree pt ON pt.id = p.parent_id
			WHERE p.deleted_at IS NULL%s
		)
		SELECT id FROM post_tree;
	`, publishedCondition, childPublishedCondition)
	if err := r.db.Raw(cte, id).Scan(&ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: post with id %s", ports.ErrNotFound, id)
	}

	var posts []post.Post
	if err := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Engagements").
		Preload("Engagements.EngagementDetails", preloadPublicEngagementDetails).
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("id IN ?", ids).
		Order("created_at ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("%w: post with id %s", ports.ErrNotFound, id)
	}

	postMap := make(map[uuid.UUID]*post.Post, len(posts))
	for i := range posts {
		posts[i].Children = nil // temizle
		postMap[posts[i].ID] = &posts[i]
	}

	var buildTree func(parent *post.Post)
	buildTree = func(parent *post.Post) {
		for _, p := range posts {
			if p.ParentID != nil && *p.ParentID == parent.ID {
				child := postMap[p.ID]
				buildTree(child)
				parent.Children = append(parent.Children, *child)
			}
		}
	}

	root, ok := postMap[id]
	if !ok {
		return nil, fmt.Errorf("%w: post with id %s", ports.ErrNotFound, id)
	}

	buildTree(root)

	var sortChildren func(p *post.Post)
	sortChildren = func(p *post.Post) {
		sort.SliceStable(p.Children, func(i, j int) bool {
			if p.Children[i].PublicID != p.Children[j].PublicID {
				return p.Children[i].PublicID < p.Children[j].PublicID
			}
			return p.Children[i].CreatedAt.Before(p.Children[j].CreatedAt)
		})
		for i := range p.Children {
			sortChildren(&p.Children[i])
		}
	}
	sortChildren(root)

	return root, nil
}

func (r *PostRepository) GetPostBySlug(filters types.Filter) (*post.Post, error) {
	if filters.Slug == nil || strings.TrimSpace(*filters.Slug) == "" {
		return nil, errors.New("post slug is required")
	}
	var ids []uuid.UUID

	cte := `
	WITH RECURSIVE post_tree AS (
		SELECT id
		FROM posts
		WHERE slug = ? AND published = TRUE AND deleted_at IS NULL
		  AND post_kind NOT IN ('chat', 'message')
		  AND COALESCE(NULLIF(audience, ''), 'public') = 'public'
		UNION ALL
		SELECT p.id
		FROM posts p
		INNER JOIN post_tree pt ON pt.id = p.parent_id
		WHERE p.published = TRUE AND p.deleted_at IS NULL
		  AND p.post_kind NOT IN ('chat', 'message')
		  AND COALESCE(NULLIF(p.audience, ''), 'public') = 'public'
	)
	SELECT id FROM post_tree;
	`

	if err := r.db.Raw(cte, *filters.Slug).Scan(&ids).Error; err != nil {

		return nil, err
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: post with slug %s", ports.ErrNotFound, *filters.Slug)
	}

	var posts []post.Post
	if err := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Engagements").
		Preload("Engagements.EngagementDetails", preloadPublicEngagementDetails).
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("id IN ?", ids).
		Order("created_at ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("%w: post with slug %s", ports.ErrNotFound, *filters.Slug)
	}

	// map oluştur
	postMap := make(map[uuid.UUID]*post.Post, len(posts))
	for i := range posts {
		posts[i].Children = nil
		postMap[posts[i].ID] = &posts[i]
	}

	// tree build
	var buildTree func(parent *post.Post)
	buildTree = func(parent *post.Post) {
		for _, p := range posts {
			if p.ParentID != nil && *p.ParentID == parent.ID {
				child := postMap[p.ID]
				buildTree(child)
				parent.Children = append(parent.Children, *child)
			}
		}
	}

	// root bul (slug'a karşılık gelen)
	var root *post.Post
	for _, p := range posts {
		if p.Slug != nil && filters.Slug != nil && *p.Slug == *filters.Slug {
			root = postMap[p.ID]
			break
		}

	}

	if root == nil {
		return nil, fmt.Errorf("%w: post with slug %s", ports.ErrNotFound, *filters.Slug)
	}

	buildTree(root)

	// children sort
	var sortChildren func(p *post.Post)
	sortChildren = func(p *post.Post) {
		sort.SliceStable(p.Children, func(i, j int) bool {
			if p.Children[i].PublicID != p.Children[j].PublicID {
				return p.Children[i].PublicID < p.Children[j].PublicID
			}
			return p.Children[i].CreatedAt.Before(p.Children[j].CreatedAt)
		})
		for i := range p.Children {
			sortChildren(&p.Children[i])
		}
	}

	sortChildren(root)

	return root, nil
}

func (r *PostRepository) GetPostByPublicID(id int64) (*post.Post, error) {
	var p post.Post

	err := r.db.
		First(&p, "public_id = ? AND published = TRUE AND post_kind NOT IN ? AND COALESCE(NULLIF(audience, ''), 'public') = 'public'", id, []post.PostKind{post.PostKindChat, post.PostKindMessage}).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: post with id %d", ports.ErrNotFound, id)
		}
		return nil, err
	}

	return r.GetPostByID(p.ID)
}

func (r *PostRepository) GetPostByIDWithoutRelations(id uuid.UUID) (*post.Post, error) {
	var p post.Post

	err := r.db.
		Preload("Author").
		Where("id = ?", id).
		First(&p).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("post with id %s not found", id)
		}
		return nil, err
	}

	return &p, nil
}

func (r *PostRepository) GetTimeline(filters types.Filter) (legacyviews.TimelineResult, error) {
	var posts []post.Post

	query := r.db.Model(&post.Post{}).
		Where("published = ?", true).
		Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'").
		Where("post_kind IN ?", []string{string(post.PostKindPost), string(post.PostKindNews), string(post.PostKindStatus), string(post.PostKindVideo)}).
		Where("parent_id IS NULL").
		Order("public_id DESC").
		Limit(filters.Limit).
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails", preloadPublicEngagementDetails).
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Hashtags").
		Preload("Mentions").
		Preload("Attachments").
		Preload("Attachments.File")

	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return legacyviews.TimelineResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		var cursorErr error
		nextCursor, cursorErr = types.NewPublicIDCursor(posts[len(posts)-1].PublicID)
		if cursorErr != nil {
			return legacyviews.TimelineResult{}, cursorErr
		}
	}

	return legacyviews.TimelineResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func (r *PostRepository) GetTimelineVibes(filters types.Filter) (legacyviews.TimelineResult, error) {
	var posts []post.Post

	query := r.db.Model(&post.Post{}).
		//	Joins("INNER JOIN medias ON medias.owner_id = posts.id AND medias.owner_type = ?", "post").
		Preload("Author").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails", preloadPublicEngagementDetails).
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("post_kind IN ?", []string{string(post.PostKindPost), string(post.PostKindStatus)}).
		Where("published = ?", true).
		Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'").
		Order("posts.public_id DESC").
		Limit(filters.Limit).
		Group("posts.id")

	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil {
		query = query.Where("posts.public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return legacyviews.TimelineResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		var cursorErr error
		nextCursor, cursorErr = types.NewPublicIDCursor(posts[len(posts)-1].PublicID)
		if cursorErr != nil {
			return legacyviews.TimelineResult{}, cursorErr
		}
	}

	return legacyviews.TimelineResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func (r *PostRepository) GetPostsByKind(filters types.Filter) (legacyviews.PostsResult, error) {
	var posts []post.Post

	query := r.postsByKindQuery(filters)

	if err := query.Find(&posts).Error; err != nil {
		return legacyviews.PostsResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		var cursorErr error
		nextCursor, cursorErr = types.NewPublicIDCursor(posts[len(posts)-1].PublicID)
		if cursorErr != nil {
			return legacyviews.PostsResult{}, cursorErr
		}
	}

	return legacyviews.PostsResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func (r *PostRepository) postsByKindQuery(filters types.Filter) *gorm.DB {
	query := r.db.Model(&post.Post{}).
		Where("published = ?", true).
		Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'").
		Where("post_kind = ?", filters.PostKind).
		Where("parent_id IS NULL").
		Order("public_id DESC").
		Limit(filters.Limit).
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails", preloadPublicEngagementDetails).
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Hashtags").
		Preload("Mentions").
		Preload("Attachments").
		Preload("Attachments.File")

	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.PostKind == post.PostKindStory {
		query = query.Where("created_at > ?", time.Now().Add(-24*time.Hour))
	}

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	return query
}

func (r *PostRepository) FindPostsByKind(filters types.Filter) (legacyviews.PostsResult, error) {
	var posts []post.Post

	limit := filters.Limit
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}

	query := r.db.Model(&post.Post{}).
		Where("published = ?", true).
		Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'").
		Where("parent_id IS NULL").
		Order("public_id DESC").
		Limit(limit).
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails", preloadPublicEngagementDetails).
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Hashtags").
		Preload("Mentions").
		Preload("Attachments").
		Preload("Attachments.File")

	if filters.Domain != nil {
		domain := strings.TrimSpace(string(*filters.Domain))
		if domain != "" && domain != string(models.AllDomains) && domain != string(models.UnknownDomain) {
			query = query.Where("domain = ?", domain)
		}
	}

	if filters.PostKind != "" {
		if filters.PostKind == post.PostKindEvent {
			// Older event posts could be created through the generic status form
			// and therefore have post_kind = 'status' while still owning an event.
			// Include those records in event search as well.
			query = query.Where(`
				(post_kind = ? OR EXISTS (
					SELECT 1 FROM events event_kind_search
					WHERE event_kind_search.post_id = posts.id
					AND event_kind_search.deleted_at IS NULL
				))
			`, filters.PostKind)
		} else {
			query = query.Where("post_kind = ?", filters.PostKind)
		}
	} else {
		// The unscoped search result is for social content. Events and places
		// are queried separately so the API can return stable result groups.
		query = query.Where("post_kind IN ?", []string{
			string(post.PostKindPost),
			string(post.PostKindStatus),
			string(post.PostKindVideo),
			string(post.PostKindNews),
		})
		query = query.Where(`NOT EXISTS (
			SELECT 1 FROM events social_event_search
			WHERE social_event_search.post_id = posts.id
			AND social_event_search.deleted_at IS NULL
		)`)
	}

	if filters.Search != nil {
		searchText := strings.TrimSpace(*filters.Search)
		if searchText != "" {
			search := "%" + escapeSearchPattern(searchText) + "%"
			query = query.Where(`
				COALESCE(title::text, '') ILIKE ? ESCAPE '\'
				OR COALESCE(content::text, '') ILIKE ? ESCAPE '\'
				OR COALESCE(summary::text, '') ILIKE ? ESCAPE '\'
				OR COALESCE(metadata::text, '') ILIKE ? ESCAPE '\'
				OR COALESCE(extras::text, '') ILIKE ? ESCAPE '\'
				OR COALESCE(slug, '') ILIKE ? ESCAPE '\'
				OR EXISTS (
					SELECT 1 FROM events search_events
					WHERE search_events.post_id = posts.id
					AND (
						COALESCE(search_events.title::text, '') ILIKE ? ESCAPE '\'
						OR COALESCE(search_events.description::text, '') ILIKE ? ESCAPE '\'
					)
				)
				OR EXISTS (
					SELECT 1 FROM locations search_locations
					LEFT JOIN events search_events_location
						ON search_events_location.id = search_locations.contentable_id
						AND search_locations.contentable_type = 'event'
					WHERE (
						(search_locations.contentable_type = 'post' AND search_locations.contentable_id = posts.id)
						OR (search_locations.contentable_type = 'event' AND search_events_location.post_id = posts.id)
					)
					AND (
						COALESCE(search_locations.address, '') ILIKE ? ESCAPE '\'
						OR COALESCE(search_locations.city, '') ILIKE ? ESCAPE '\'
						OR COALESCE(search_locations.town, '') ILIKE ? ESCAPE '\'
						OR COALESCE(search_locations.province, '') ILIKE ? ESCAPE '\'
						OR COALESCE(search_locations.country, '') ILIKE ? ESCAPE '\'
					)
				)
				OR EXISTS (
					SELECT 1 FROM hashtags search_hashtags
					WHERE search_hashtags.taggable_id = posts.id
					AND search_hashtags.taggable_type = 'post'
					AND (
						COALESCE(search_hashtags.tag, '') ILIKE ? ESCAPE '\'
						OR COALESCE(search_hashtags.slug, '') ILIKE ? ESCAPE '\'
					)
				)
			`,
				search, search, search, search, search, search,
				search, search,
				search, search, search, search, search,
				search, search,
			)
		}
	}
	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return legacyviews.PostsResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		var cursorErr error
		nextCursor, cursorErr = types.NewPublicIDCursor(posts[len(posts)-1].PublicID)
		if cursorErr != nil {
			return legacyviews.PostsResult{}, cursorErr
		}
	}

	return legacyviews.PostsResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func escapeSearchPattern(raw string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(raw)
}

func (r *PostRepository) GetUserPosts(userId uuid.UUID, filters types.Filter) ([]post.Post, error) {
	var posts []post.Post

	query := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Avatar").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("author_id = ? AND parent_id IS NULL AND post_kind = ? AND published = TRUE", userId, filters.PostKind).
		Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'").
		Order("posts.public_id DESC").
		Limit(filters.Limit)

	if filters.Cursor != nil {
		query = query.Where("posts.public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepository) GetUserPostReplies(filters types.Filter) ([]post.Post, error) {
	var posts []post.Post

	query := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices").
		Preload("Event").
		Preload("Parent", "published = TRUE AND post_kind NOT IN ? AND COALESCE(NULLIF(audience, ''), 'public') = 'public'", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Preload("Event.Location").
		Preload("Event.Attendees", preloadEventAttendees).
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Avatar").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Joins("JOIN posts visible_parents ON visible_parents.id = posts.parent_id AND visible_parents.published = TRUE AND visible_parents.deleted_at IS NULL AND visible_parents.post_kind NOT IN ('chat', 'message') AND COALESCE(NULLIF(visible_parents.audience, ''), 'public') = 'public'").
		Where("posts.author_id = ? AND posts.parent_id IS NOT NULL AND posts.post_kind = ? AND posts.published = TRUE", filters.UserUUID, filters.PostKind).
		Where("posts.post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(posts.audience, ''), 'public') = 'public'").
		Order("posts.public_id DESC").
		Limit(filters.Limit)

	// Cursor varsa sadece daha eski postlar
	if filters.Cursor != nil {
		query = query.Where("posts.public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepository) GetUserMedias(filters types.Filter) ([]legacyviews.MediaWithUser, *int64, error) {
	var medias []media.Media

	query := r.db.Model(&media.Media{}).
		Preload("File").
		Joins("JOIN posts public_media_posts ON public_media_posts.id = medias.owner_id").
		Where("medias.user_id = ?", filters.UserUUID).
		Where("medias.is_public = TRUE").
		Where("medias.owner_type IN ?", []media.OwnerType{media.OwnerPost, media.OwnerNews, media.OwnerVideo}).
		Where("public_media_posts.published = TRUE AND public_media_posts.deleted_at IS NULL").
		Where("public_media_posts.post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(public_media_posts.audience, ''), 'public') = 'public'").
		Order("medias.public_id DESC").
		Limit(filters.Limit)

	if filters.Cursor != nil {
		query = query.Where("medias.public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&medias).Error; err != nil {
		return nil, nil, err
	}

	// userID'leri topla
	userIDs := make([]uuid.UUID, 0, len(medias))
	for _, m := range medias {
		userIDs = append(userIDs, m.UserID)
	}

	var users []models.User
	if len(userIDs) > 0 {
		if err := r.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, nil, err
		}
	}

	userMap := make(map[uuid.UUID]models.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Sonuçları MediaWithUser tipine dönüştür
	results := make([]legacyviews.MediaWithUser, 0, len(medias))
	for _, m := range medias {
		results = append(results, legacyviews.MediaWithUser{
			Media: m,
			User:  userMap[m.UserID],
		})
	}

	var lastCursor *int64 = nil
	if len(medias) > 0 {
		lastCursor = &medias[len(medias)-1].PublicID
	}

	return results, lastCursor, nil
}

func postKindForContentableType(contentableType string) (post.PostKind, bool) {
	switch contentableType {
	case "chat":
		return post.PostKindMessage, true
	case "post":
		return post.PostKindPost, true
	case "event":
		return post.PostKindEvent, true
	case "status":
		return post.PostKindStatus, true
	case "classified":
		return post.PostKindClassified, true
	case "job_offer":
		return post.PostKindJobOffer, true
	case "job_search":
		return post.PostKindJobSearch, true
	case "news":
		return post.PostKindNews, false
	case "place":
		return post.PostKindPlace, true
	case "checkin":
		return post.PostKindCheckIn, true
	case "video":
		return post.PostKindVideo, true
	case "story":
		return post.PostKindStory, true
	default:
		return post.PostKindStatus, false
	}
}

func resolvePostKindForContentableType(contentableType string, hasEvent bool) (post.PostKind, bool) {
	if hasEvent {
		return post.PostKindEvent, true
	}
	return postKindForContentableType(contentableType)
}

func mediaOwnerForContentableType(contentableType string) (media.OwnerType, media.MediaRole) {
	switch contentableType {
	case "chat":
		return media.OwnerPost, media.RoleChatMedia
	case "news":
		return media.OwnerNews, media.RolePost
	case "video":
		return media.OwnerVideo, media.RoleVideo
	case "story":
		return media.OwnerPost, media.RoleStory
	default:
		return media.OwnerPost, media.RolePost
	}
}

func (r *PostRepository) GetRecentHashtags(filters types.Filter) ([]types.HashtagStats, error) {
	var results []types.HashtagStats
	cutoff := time.Now().Add(-48 * time.Hour)

	err := r.db.Model(&models.Hashtag{}).
		Preload("RelatedHashtags").
		Select("hashtags.tag, COUNT(*) as count").
		Joins("JOIN posts hashtag_posts ON hashtag_posts.id = hashtags.taggable_id AND hashtags.taggable_type = ?", models.EngagementContentableTypePost).
		Where("hashtags.created_at >= ?", cutoff).
		Where("hashtag_posts.published = TRUE AND hashtag_posts.deleted_at IS NULL").
		Where("hashtag_posts.post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(hashtag_posts.audience, ''), 'public') = 'public'").
		Group("hashtags.tag").
		Order("count DESC").
		Limit(filters.Limit).
		Scan(&results).Error

	return results, err
}

type contentablePostCreation struct {
	post       *post.Post
	parentPost *post.Post
}

const commentEngagementDedupePrefix = "post-comment:"

func commentEngagementDedupeKey(commentID uuid.UUID) string {
	return commentEngagementDedupePrefix + commentID.String()
}

func syncPostCommentCountInTransaction(tx *gorm.DB, aggregateID, parentPostID uuid.UUID) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}

	var activeComments int64
	if err := tx.Model(&post.Post{}).
		Where("parent_id = ?", parentPostID).
		Count(&activeComments).Error; err != nil {
		return err
	}

	var aggregate models.Engagement
	if err := tx.Select("id", "counts").First(&aggregate, "id = ?", aggregateID).Error; err != nil {
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
	counts[models.EngagementCountKeys[models.EngagementKindComment].CountKey] = activeComments
	payload, err := json.Marshal(counts)
	if err != nil {
		return err
	}
	result := tx.Model(&models.Engagement{}).
		Where("id = ?", aggregateID).
		Updates(map[string]interface{}{
			"counts":     datatypes.JSON(payload),
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("parent post engagement aggregate was not updated")
	}
	return nil
}

func addCommentEngagementInTransaction(tx *gorm.DB, authorID uuid.UUID, parentPost *post.Post, comment *post.Post) error {
	if tx == nil {
		return errors.New("transaction is nil")
	}
	if authorID == uuid.Nil || parentPost == nil || parentPost.ID == uuid.Nil || parentPost.AuthorID == uuid.Nil || comment == nil || comment.ID == uuid.Nil {
		return errors.New("comment engagement identifiers are required")
	}

	if tx.Name() == "postgres" {
		if err := lockViewAggregate(tx, engagementAggregateLockKey(models.EngagementContentableTypePost, parentPost.ID)).Error; err != nil {
			return err
		}
	}
	aggregate, err := loadOrCreateEngagementAggregate(tx, parentPost.ID, models.EngagementContentableTypePost)
	if err != nil {
		return err
	}

	dedupeKey := commentEngagementDedupeKey(comment.ID)
	details, err := json.Marshal(map[string]string{"comment_post_id": comment.ID.String()})
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	detail := &models.EngagementDetail{
		ID:           uuid.New(),
		EngagementID: aggregate.ID,
		DedupeKey:    &dedupeKey,
		EngagerID:    authorID,
		EngageeID:    parentPost.AuthorID,
		Kind:         models.EngagementKindComment,
		Details:      datatypes.JSON(details),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := createEngagementDetailInTransaction(tx, detail); err != nil {
		return err
	}
	return syncPostCommentCountInTransaction(tx, aggregate.ID, parentPost.ID)
}

func (r *PostRepository) transactionScoped(db *gorm.DB) *PostRepository {
	scoped := *r
	scoped.db = db
	if r.mediaRepo != nil {
		mediaRepo := *r.mediaRepo
		mediaRepo.db = db
		scoped.mediaRepo = &mediaRepo
	}
	if r.userRepo != nil {
		userRepo := *r.userRepo
		userRepo.db = db
		scoped.userRepo = &userRepo
	}
	return &scoped
}

func (r *PostRepository) CreateContentablePost(ctx context.Context, formData ports.FormData, author *models.User, contentableType string, contentableID *uuid.UUID) (createdPost *post.Post, retErr error) {
	createdMediaPaths := make([]string, 0, len(formData.Files))
	committed := false
	defer func() {
		if committed {
			return
		}
		cleanupErr := cleanupStoredUploads(createdMediaPaths)
		if recovered := recover(); recovered != nil {
			if cleanupErr != nil {
				helpers.Error("post rollback media cleanup error: %v", cleanupErr)
			}
			panic(recovered)
		}
		retErr = errors.Join(retErr, cleanupErr)
	}()

	var creation *contentablePostCreation
	retErr = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		creation, err = r.transactionScoped(tx).createContentablePostInTransaction(ctx, formData, author, contentableType, contentableID, &createdMediaPaths)
		return err
	})
	if retErr != nil {
		return nil, retErr
	}
	committed = true

	r.runPostCommitCommentSideEffects(ctx, author, creation)

	return creation.post, nil
}

func (r *PostRepository) runPostCommitCommentSideEffects(ctx context.Context, author *models.User, creation *contentablePostCreation) {
	if creation == nil || creation.parentPost == nil {
		return
	}
	// The parent aggregate is committed with the comment itself. Only external
	// notification delivery remains a best-effort post-commit side effect.
	if author == nil || author.ID == creation.parentPost.AuthorID || r.userRepo == nil {
		return
	}
	if err := r.sendNotificationWithType(
		ctx,
		author.ID,
		creation.parentPost.AuthorID,
		notifications.NotificationTypeComment,
		commentNotificationPayload(author, creation.parentPost, creation.post),
	); err != nil {
		helpers.Error("comment notification error: %v", err)
	}
}

func (r *PostRepository) createContentablePostInTransaction(ctx context.Context, formData ports.FormData, author *models.User, contentableType string, contentableID *uuid.UUID, createdMediaPaths *[]string) (*contentablePostCreation, error) {
	type PollForm struct {
		ID            string   `form:"id"`
		Question      string   `form:"question"`
		Duration      string   `form:"duration"`
		Kind          string   `form:"kind"`           // single, multiple, ranked, weighted
		MaxSelectable string   `form:"max_selectable"` // only for multiple, weighted
		Options       []string `form:"options"`        // seçenekler
		InitialRanks  []string `form:"ranks"`          // ranked için: her option'ın başlangıç sırası
		Weights       []string `form:"weights"`        // only for weighted, optional
	}

	type PostForm struct {
		ParentId string     `form:"parentPostId"`
		Title    string     `form:"title"`
		Slug     string     `form:"slug"`
		Summary  string     `form:"summary"`
		Language string     `form:"language"`
		Content  string     `form:"content"`
		Audience string     `form:"audience"`
		Hashtags []string   `form:"hashtags[]"`
		Mentions []string   `form:"mentions[]"`
		Polls    []PollForm `form:"polls"`

		EventTitle       string `form:"event[title]"`
		EventDescription string `form:"event[description]"`
		EventDate        string `form:"event[date]"`
		EventTime        string `form:"event[time]"`
		EventKind        string `form:"event[kind]"`
		EventCapacity    string `form:"event[capacity]"`
		EventIsPaid      string `form:"event[is_paid]"`
		EventPrice       string `form:"event[price]"`
		EventCurrency    string `form:"event[currency]"`
		EventIsOnline    string `form:"event[is_online]"`
		EventIsOnlineURL string `form:"event[online_url]"`

		LocationAddress string  `form:"location[address]"`
		LocationLat     float64 `form:"location[lat]"`
		LocationLng     float64 `form:"location[lng]"`
		CountryCode     string  `form:"location[country_code]"`
		Region          string  `form:"location[region]"`
		City            string  `form:"location[city]"`
		ZipCode         string  `form:"location[zip_code]"`
		Province        string  `form:"location[province]"`
		Town            string  `form:"location[town]"`
		Postcode        string  `form:"location[postcode]"`
		Country         string  `form:"location[country]"`
		Extras          string  `form:"extras"`
	}

	decoder := form.NewDecoder()
	postForm := PostForm{}

	if err := decoder.Decode(&postForm, formData.Values); err != nil {
		return nil, err
	}

	tx := r.DB()

	var parentUUID *uuid.UUID
	var parentPost *post.Post
	if len(postForm.ParentId) > 0 {
		parentIDInt, err := strconv.ParseInt(postForm.ParentId, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid parentId %s: %w", postForm.ParentId, err)
		}
		parentPost, err = r.FindPostByPublicID(parentIDInt)
		if err != nil {
			return nil, err
		}
		parentUUID = &parentPost.ID
	}

	defaultLanguage := helpers.DefaultIfEmpty(postForm.Language, author.DefaultLanguage)

	postKindType, isPublished := resolvePostKindForContentableType(contentableType, strings.TrimSpace(postForm.EventTitle) != "")

	postForm.Slug = helpers.GenerateSlug(
		func() string {
			if postForm.Slug != "" {
				return postForm.Slug
			}
			return postForm.Title
		}(),
	)
	newPost := &post.Post{
		ID:              uuid.New(),
		ParentID:        parentUUID,
		PublicID:        r.Node().Generate().Int64(),
		AuthorID:        author.ID,
		Published:       isPublished,
		Domain:          author.Domain,
		PostKind:        postKindType,
		ContentCategory: post.ContentNormal,
		Title:           utils.MakeLocalizedString(defaultLanguage, postForm.Title),
		Content:         utils.MakeLocalizedString(defaultLanguage, postForm.Content),
		Summary:         utils.MakeLocalizedString(defaultLanguage, postForm.Summary),
		Slug:            &postForm.Slug,
		Audience:        &postForm.Audience,
		ContentableType: &contentableType,
		ContentableID:   contentableID,
	}
	if contentableType == string(post.PostKindChat) {
		newPost.ExpiresInSeconds = formData.ExpiresInSeconds
		newPost.ViewOnce = formData.ViewOnce
		newPost.ClientID = formData.ClientID
	}

	if err := tx.Create(newPost).Error; err != nil {
		return nil, err
	}

	for _, f := range formData.Files {
		ownerType, role := mediaOwnerForContentableType(contentableType)

		mediaModel, err := r.mediaRepo.AddMedia(newPost.ID, ownerType, author.ID, role, f)
		if err != nil {
			return nil, err
		}
		*createdMediaPaths = append(*createdMediaPaths, mediaModel.File.StoragePath)
		newPost.Attachments = append(newPost.Attachments, mediaModel)
	}

	// Polls ekleme
	for _, pollInfo := range postForm.Polls {

		maxSelectable := 1
		if len(pollInfo.MaxSelectable) > 0 {
			value, err := strconv.Atoi(strings.TrimSpace(pollInfo.MaxSelectable))
			if err != nil {
				return nil, domainpost.ErrInvalidPollMaximum
			}
			maxSelectable = value
		}
		definition, err := domainpost.NewPollDefinition(
			pollInfo.Question,
			domainpost.PollKind(strings.ToLower(strings.TrimSpace(pollInfo.Kind))),
			maxSelectable,
			pollInfo.Options,
		)
		if err != nil {
			return nil, err
		}

		poll := &post_payloads.Poll{
			ID:              uuid.New(),
			ContentableID:   newPost.ID,
			ContentableType: post_payloads.ContentablePollPost,
			Question:        *utils.MakeLocalizedString(defaultLanguage, definition.Question),
			Duration:        pollInfo.Duration,
			Kind:            post_payloads.PollKind(definition.Kind),
			MaxSelectable:   definition.MaxSelectable,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		for index, choiceLabel := range definition.Options {
			poll.Choices = append(poll.Choices, post_payloads.PollChoice{
				ID:           uuid.New(),
				DisplayOrder: index,
				PollID:       poll.ID,
				Label:        *utils.MakeLocalizedString(defaultLanguage, choiceLabel),
				VoteCount:    0,
			})
		}
		if err := r.CreatePoll(poll); err != nil {
			return nil, err
		}
		newPost.Poll = append(newPost.Poll, poll)
	}

	// Location. Zero is a valid latitude/longitude, so presence must be read
	// from the request rather than inferred from the decoded numeric value.
	latitudeValues := formData.Values["location[lat]"]
	longitudeValues := formData.Values["location[lng]"]
	hasLatitude := len(latitudeValues) > 0 && strings.TrimSpace(latitudeValues[0]) != ""
	hasLongitude := len(longitudeValues) > 0 && strings.TrimSpace(longitudeValues[0]) != ""
	if hasLatitude != hasLongitude {
		return nil, errors.New("latitude and longitude must be provided together")
	}
	var locationPoint *extensions.PostGISPoint
	var locationLatitude, locationLongitude *float64
	if hasLatitude {
		if _, err := domainuser.NewCoordinates(postForm.LocationLat, postForm.LocationLng); err != nil {
			return nil, err
		}
		locationPoint = &extensions.PostGISPoint{
			Lat: postForm.LocationLat,
			Lng: postForm.LocationLng,
		}
		locationLatitude = &postForm.LocationLat
		locationLongitude = &postForm.LocationLng
		locationPost := &utils.Location{
			ID:              uuid.New(),
			ContentableType: utils.LocationOwnerPost,
			ContentableID:   newPost.ID,
			Address:         &postForm.LocationAddress,
			Latitude:        locationLatitude,
			Longitude:       locationLongitude,
			CountryCode:     &postForm.CountryCode,
			Region:          &postForm.Region,
			City:            &postForm.City,
			ZipCode:         &postForm.ZipCode,
			Province:        &postForm.Province,
			Town:            &postForm.Town,
			Postcode:        &postForm.Postcode,
			Country:         &postForm.Country,
			LocationPoint:   locationPoint,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := r.userRepo.UpsertLocation(locationPost); err != nil {
			return nil, err
		}
	}

	// Event
	if len(postForm.EventTitle) > 0 {
		var startTime *time.Time
		hasEventDate := strings.TrimSpace(postForm.EventDate) != ""
		hasEventTime := strings.TrimSpace(postForm.EventTime) != ""
		if hasEventDate != hasEventTime {
			return nil, errors.New("event date and time must be provided together")
		}
		if hasEventDate {
			parsedTime, err := time.Parse("2006-01-02 15:04", postForm.EventDate+" "+postForm.EventTime)
			if err != nil {
				return nil, fmt.Errorf("invalid event date or time: %w", err)
			}
			startTime = &parsedTime
		}

		isPaid := false
		if strings.TrimSpace(postForm.EventIsPaid) != "" {
			parsed, err := strconv.ParseBool(postForm.EventIsPaid)
			if err != nil {
				return nil, fmt.Errorf("invalid event paid flag: %w", err)
			}
			isPaid = parsed
		}
		isOnline := false
		if strings.TrimSpace(postForm.EventIsOnline) != "" {
			parsed, err := strconv.ParseBool(postForm.EventIsOnline)
			if err != nil {
				return nil, fmt.Errorf("invalid event online flag: %w", err)
			}
			isOnline = parsed
		}

		var pricePtr *float64
		if postForm.EventPrice != "" {
			price, err := strconv.ParseFloat(postForm.EventPrice, 64)
			if err != nil || price < 0 {
				return nil, errors.New("invalid event price")
			}
			pricePtr = &price
		}

		evt := &post_payloads.Event{
			ID:          uuid.New(),
			PostID:      newPost.ID,
			Title:       *utils.MakeLocalizedString(defaultLanguage, postForm.EventTitle),
			Description: *utils.MakeLocalizedString(defaultLanguage, postForm.EventDescription),
			Kind:        postForm.EventKind,
			IsPaid:      isPaid,
			Price:       pricePtr,
			Currency:    &postForm.EventCurrency,
			IsOnline:    isOnline,
			OnlineURL:   &postForm.EventIsOnlineURL,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			StartTime:   startTime,
		}

		if err := tx.Create(evt).Error; err != nil {
			return nil, err
		}

		locationEvent := &utils.Location{
			ID:              uuid.New(),
			ContentableType: utils.LocationOwnerEvent,
			ContentableID:   evt.ID,
			Address:         &postForm.LocationAddress,
			Latitude:        locationLatitude,
			Longitude:       locationLongitude,
			LocationPoint:   locationPoint,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := tx.Create(locationEvent).Error; err != nil {
			return nil, err
		}
		evt.Location = locationEvent
		newPost.Event = evt
	}

	// Mentions
	for _, mentionText := range postForm.Mentions {
		mentionUser, err := r.userRepo.GetUserByNameOrEmailOrNickname(mentionText)
		if err == nil {
			mentionItem := models.Mention{
				ID:     uuid.New(),
				UserID: mentionUser.ID,
			}
			newPost.Mentions = append(newPost.Mentions, &mentionItem)
		}
	}

	var hashtagItems []*models.Hashtag
	seen := make(map[string]bool)

	for _, raw := range postForm.Hashtags {
		normalized := helpers.SlugifyStrict(strings.TrimPrefix(raw, "#"))

		if normalized == "" {
			continue
		}

		if seen[normalized] {
			continue
		}
		seen[normalized] = true

		hashtagItem := &models.Hashtag{
			Domain: author.Domain,
			ID:     uuid.New(),
			Tag:    normalized,
			Slug:   helpers.GenerateSlug(normalized),
		}

		hashtagItems = append(hashtagItems, hashtagItem)
	}
	for i := range hashtagItems {
		for j := range hashtagItems {
			if i != j {
				hashtagItems[i].RelatedHashtags =
					append(hashtagItems[i].RelatedHashtags, hashtagItems[j])
			}
		}
	}
	newPost.Hashtags = hashtagItems

	if postForm.Extras != "" {
		var extras any
		if err := json.Unmarshal([]byte(postForm.Extras), &extras); err != nil {
			return nil, err
		}
		extrasBytes, err := json.Marshal(extras)
		if err != nil {
			return nil, err
		}
		newPost.Extras = datatypes.JSON(extrasBytes)

	}

	if err := tx.Save(newPost).Error; err != nil {
		return nil, err
	}
	if parentPost != nil {
		if err := addCommentEngagementInTransaction(tx, author.ID, parentPost, newPost); err != nil {
			return nil, err
		}
	}

	return &contentablePostCreation{post: newPost, parentPost: parentPost}, nil
}

type pollVoteSelection struct {
	ChoiceID      uuid.UUID `gorm:"column:choice_id"`
	PollID        uuid.UUID `gorm:"column:poll_id"`
	Kind          string    `gorm:"column:kind"`
	MaxSelectable int       `gorm:"column:max_selectable"`
	ChoiceCount   int       `gorm:"column:choice_count"`
}

func decrementPollChoiceCount(tx *gorm.DB, choiceID uuid.UUID) error {
	return tx.Model(&post_payloads.PollChoice{}).
		Where("id = ?", choiceID).
		UpdateColumn("vote_count", gorm.Expr("CASE WHEN vote_count > 0 THEN vote_count - 1 ELSE 0 END")).Error
}

func (r *PostRepository) Vote(ctx context.Context, choiceID uuid.UUID, weight int, rank int, userID uuid.UUID) error {
	if choiceID == uuid.Nil || userID == uuid.Nil {
		return domainpost.ErrInvalidPollChoiceData
	}
	if r.db.Name() != "postgres" {
		fallbackPollVoteLock.Lock()
		defer fallbackPollVoteLock.Unlock()
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var selection pollVoteSelection
		if err := tx.
			Table("poll_choices").
			Select(`
				poll_choices.id AS choice_id,
				poll_choices.poll_id,
				polls.kind,
				polls.max_selectable,
				(SELECT COUNT(*) FROM poll_choices AS all_choices WHERE all_choices.poll_id = polls.id) AS choice_count
			`).
			Joins("JOIN polls ON polls.id = poll_choices.poll_id AND polls.contentable_type = ?", models.EngagementContentableTypePost).
			Joins("JOIN posts ON posts.id = polls.contentable_id").
			Where("poll_choices.id = ?", choiceID).
			Where("posts.published = TRUE AND posts.deleted_at IS NULL").
			Where("posts.post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
			Where("COALESCE(NULLIF(posts.audience, ''), 'public') = 'public'").
			Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "polls"}}).
			Take(&selection).Error; err != nil {
			return fmt.Errorf("choice not found: %w", err)
		}

		var existing []post_payloads.PollVote
		if err := tx.
			Model(&post_payloads.PollVote{}).
			Joins("JOIN poll_choices AS selected_choices ON selected_choices.id = poll_votes.choice_id").
			Where("selected_choices.poll_id = ? AND poll_votes.user_id = ?", selection.PollID, userID).
			Order("poll_votes.created_at ASC, poll_votes.id ASC").
			Find(&existing).Error; err != nil {
			return err
		}

		for _, vote := range existing {
			if vote.ChoiceID != choiceID {
				continue
			}
			if err := tx.Delete(&post_payloads.PollVote{}, "id = ?", vote.ID).Error; err != nil {
				return err
			}
			return decrementPollChoiceCount(tx, choiceID)
		}

		policy := domainpost.PollVotePolicy{
			Kind:          domainpost.PollKind(selection.Kind),
			MaxSelectable: selection.MaxSelectable,
			ChoiceCount:   selection.ChoiceCount,
		}
		rankUsed := false
		for _, vote := range existing {
			if rank > 0 && vote.Rank == rank {
				rankUsed = true
			}
		}
		if err := policy.ValidateNewVote(len(existing), weight, rank, rankUsed); err != nil {
			return err
		}

		// A single-choice vote is a replace command. Repair any historical
		// duplicate rows while holding the poll lock before inserting the new
		// selection, keeping every cached vote_count consistent.
		if policy.Kind == "" || policy.Kind == domainpost.PollSingle {
			for _, vote := range existing {
				if err := tx.Delete(&post_payloads.PollVote{}, "id = ?", vote.ID).Error; err != nil {
					return err
				}
				if err := decrementPollChoiceCount(tx, vote.ChoiceID); err != nil {
					return err
				}
			}
		}

		newVote := post_payloads.PollVote{
			ID:       uuid.New(),
			ChoiceID: choiceID,
			UserID:   userID,
			Weight:   weight,
			Rank:     rank,
		}
		if err := tx.Create(&newVote).Error; err != nil {
			return err
		}
		return tx.Model(&post_payloads.PollChoice{}).
			Where("id = ?", choiceID).
			UpdateColumn("vote_count", gorm.Expr("vote_count + 1")).Error
	})
}

func (r *PostRepository) FindPostByPublicID(id int64) (*post.Post, error) {
	return r.findPostByPublicID(id, false)
}

func (r *PostRepository) findPostByPublicID(id int64, includeUnpublished bool) (*post.Post, error) {
	var p post.Post
	query := r.db.Where("public_id = ?", id)
	if !includeUnpublished {
		query = query.
			Where("published = TRUE").
			Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
			Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'")
	}
	err := query.First(&p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("post with id %d not found", id)
		}
		return nil, err
	}
	return &p, nil
}

func (r *PostRepository) ExistsBySlug(filters types.Filter) (bool, error) {
	var count int64
	err := r.db.Model(&post.Post{}).
		Where("slug = ?", filters.Search).
		Where("post_kind = ?", filters.PostKind).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) SendNotification(ctx context.Context, senderUserUUID uuid.UUID, receiverUserUUID uuid.UUID, notificationPayload notifications.NotificationPayload) error {
	return r.sendNotificationWithType(ctx, senderUserUUID, receiverUserUUID, notifications.NotificationTypeReferral, notificationPayload)
}

func (r *PostRepository) sendNotificationWithType(ctx context.Context, senderUserUUID uuid.UUID, receiverUserUUID uuid.UUID, notificationType string, notificationPayload notifications.NotificationPayload) error {
	canSendNotification := true
	senderUser, err := r.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: senderUserUUID})
	if err != nil {
		canSendNotification = false
	}
	receiverUser, err := r.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: receiverUserUUID})
	if err != nil {
		canSendNotification = false
	}

	if canSendNotification {
		err = r.notificationRepo.SendNotificationToUser(*senderUser, *receiverUser, notificationType, notificationPayload.Title, notificationPayload.Body, notificationPayload)
		if err != nil {
			fmt.Printf("Bildirim gönderilemedi user %s -> %s: %v\n", senderUserUUID, receiverUserUUID, err)
			return err
		}
	}
	return nil
}

func commentNotificationPayload(author *models.User, parentPost *post.Post, comment *post.Post) notifications.NotificationPayload {
	body := "Someone commented on your post"
	if author != nil {
		name := strings.TrimSpace(author.DisplayName)
		if name == "" {
			name = strings.TrimSpace(author.UserName)
		}
		if name != "" {
			body = name + " commented on your post"
		}
	}

	return notifications.NotificationPayload{
		Title: "New Comment",
		Body:  body,
		Tag:   fmt.Sprintf("post-comment-%d", parentPost.PublicID),
		Data: map[string]any{
			"post_id":      parentPost.PublicID,
			"comment_id":   comment.PublicID,
			"post_uuid":    parentPost.ID.String(),
			"comment_uuid": comment.ID.String(),
			"type":         notifications.NotificationTypeComment,
		},
	}
}

func (r *PostRepository) Like(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindLikeReceived, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}

		if isOk {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Like",
					Body:  "Someone liked your post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Liked",
					Body:  "You liked this post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Like Removed",
					Body:  "Someone removed their like",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Like Removed",
					Body:  "You removed your like",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		}
	}
	return nil
}

func (r *PostRepository) Dislike(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindDisLikeReceived, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}

		if isOk {
			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Dislike",
					Body:  "Someone disliked your post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Disliked",
					Body:  "You disliked this post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {
			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Dislike Removed",
					Body:  "Someone removed their dislike",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Dislike Removed",
					Body:  "You removed your dislike",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		}
	}
	return nil
}

func (r *PostRepository) Banana(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindBanana, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}
		if isOk {
			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Banana 🍌",
					Body:  "Someone sent you a banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Banana Sent 🍌",
					Body:  "You sent a banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Banana Removed",
					Body:  "Someone removed their banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Banana Removed",
					Body:  "You removed your banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}
		}
	}
	return nil
}

func (r *PostRepository) Report(ctx context.Context, postId int64, kind string, description string, authUser *models.User) error {
	if authUser == nil || authUser.ID == uuid.Nil {
		return ports.ErrReportTargetNotFound
	}

	var reportedPost post.Post
	err := r.db.WithContext(ctx).
		Where("public_id = ? AND published = TRUE", postId).
		Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'").
		First(&reportedPost).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ports.ErrReportTargetNotFound
		}
		return err
	}

	return createReport(ctx, r.db, reportedPost.ID, models.EngagementContentableTypePost, authUser.ID, kind, description)
}

func (r *PostRepository) Bookmark(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindBookmark, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}

		if isOk {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Bookmark",
					Body:  "Someone bookmarked your post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Post Bookmarked",
					Body:  "You bookmarked this post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Bookmark Removed",
					Body:  "Someone removed their bookmark",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Bookmark Removed",
					Body:  "You removed your bookmark",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		}

	}
	return nil
}

func (r *PostRepository) View(filters types.Filter) (bool, error) {
	if filters.AuthUser == nil {
		return false, errors.New(constants.ErrUserUnauthorized.String())
	}

	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return false, err
	}
	if post == nil {
		return false, errors.New(constants.ErrPostNotFound.String())
	}
	if post.AuthorID == filters.AuthUser.ID {
		return false, nil
	}

	return r.userRepo.engagementRepo.RecordViewOnce(
		filters.Context,
		filters.AuthUser.ID,
		post.AuthorID,
		models.EngagementKindView,
		post.ID,
		models.EngagementContentableTypePost,
	)
}

func (r *PostRepository) Delete(filters types.Filter) error {
	if r == nil || r.db == nil || filters.AuthUser == nil || filters.AuthUser.ID == uuid.Nil {
		return errors.New(constants.ErrPostDeleteDenied.String())
	}

	ctx := filters.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// SQLite and other adapters do not support PostgreSQL advisory locks. Hold
	// the process fallback across commit so comment aggregate writes still have
	// one owner at a time in those environments.
	if r.db.Name() != "postgres" {
		fallbackEngagementViewLock.Lock()
		defer fallbackEngagementViewLock.Unlock()
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target post.Post
		find := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("public_id = ?", filters.PostID).
			First(&target)
		if errors.Is(find.Error, gorm.ErrRecordNotFound) {
			return errors.New(constants.ErrPostNotFound.String())
		}
		if find.Error != nil {
			return find.Error
		}

		allowed := target.AuthorID == filters.AuthUser.ID ||
			filters.AuthUser.Role == string(constants.UserRoleModerator) ||
			filters.AuthUser.Role == string(constants.UserRoleAdmin) ||
			filters.AuthUser.Role == string(constants.UserRoleSuperAdmin)
		if !allowed {
			return errors.New(constants.ErrPostDeleteDenied.String())
		}

		var parentAggregate *models.Engagement
		var commentDetail *models.EngagementDetail
		if target.ParentID != nil {
			if tx.Name() == "postgres" {
				lockKey := engagementAggregateLockKey(models.EngagementContentableTypePost, *target.ParentID)
				if err := lockViewAggregate(tx, lockKey).Error; err != nil {
					return err
				}
			}

			var err error
			parentAggregate, err = loadOrCreateEngagementAggregate(tx, *target.ParentID, models.EngagementContentableTypePost)
			if err != nil {
				return err
			}
			commentDetail, err = lockCommentEngagementDetailInTransaction(tx, parentAggregate.ID, &target)
			if err != nil {
				return err
			}
		}

		// The post state and parent aggregate are one unit of work. Any detail or
		// counter failure below rolls this soft delete back.
		if err := tx.Delete(&target).Error; err != nil {
			return err
		}
		if target.ParentID == nil {
			return nil
		}
		if commentDetail != nil {
			if err := tx.Delete(&models.EngagementDetail{}, "id = ?", commentDetail.ID).Error; err != nil {
				return err
			}
		}
		return syncPostCommentCountInTransaction(tx, parentAggregate.ID, *target.ParentID)
	})
	if err != nil {
		if err.Error() == constants.ErrPostNotFound.String() || err.Error() == constants.ErrPostDeleteDenied.String() {
			return err
		}
		return fmt.Errorf("%s: %w", constants.ErrPostDeleteFailed.String(), err)
	}
	return nil
}

// lockCommentEngagementDetailInTransaction locks the exact, deduplicated
// detail for new comments. The NULL-key fallback supports comments created
// before comment identity became part of the aggregate without risking the
// removal of another modern comment by the same author.
func lockCommentEngagementDetailInTransaction(tx *gorm.DB, aggregateID uuid.UUID, comment *post.Post) (*models.EngagementDetail, error) {
	if tx == nil || aggregateID == uuid.Nil || comment == nil || comment.ID == uuid.Nil {
		return nil, errors.New("comment engagement lookup identifiers are required")
	}

	var detail models.EngagementDetail
	dedupeKey := commentEngagementDedupeKey(comment.ID)
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("engagement_id = ? AND dedupe_key = ? AND kind = ?", aggregateID, dedupeKey, models.EngagementKindComment).
		First(&detail).Error
	if err == nil {
		return &detail, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("engagement_id = ? AND engager_id = ? AND kind = ? AND dedupe_key IS NULL", aggregateID, comment.AuthorID, models.EngagementKindComment).
		Order("created_at ASC, id ASC").
		First(&detail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

const tipResultingBalanceDetailKey = "resulting_balance"

func scopedTipDedupeKey(payerID uuid.UUID, idempotencyKey domainwallet.IdempotencyKey) string {
	return fmt.Sprintf("post-tip:%s:%s", payerID, idempotencyKey.String())
}

type persistedTipDetails struct {
	Amount           string `json:"amount"`
	ResultingBalance string `json:"resulting_balance"`
	PostPublicID     string `json:"post_public_id"`
}

func loadTipReplay(
	db *gorm.DB,
	dedupeKey string,
	payerID uuid.UUID,
	postPublicID int64,
	requestedAmount decimal.Decimal,
) (decimal.Decimal, bool, error) {
	var detail models.EngagementDetail
	err := db.Where("dedupe_key = ?", dedupeKey).First(&detail).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return decimal.Zero, false, nil
	}
	if err != nil {
		return decimal.Zero, false, err
	}

	var aggregate models.Engagement
	if err := db.Select("id", "contentable_id", "contentable_type").First(&aggregate, "id = ?", detail.EngagementID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return decimal.Zero, false, domainwallet.ErrIdempotencyConflict
		}
		return decimal.Zero, false, err
	}

	if detail.EngagerID != payerID ||
		detail.EngageeID == uuid.Nil ||
		detail.Kind != models.EngagementKindTip ||
		aggregate.ContentableID == uuid.Nil ||
		aggregate.ContentableType != models.EngagementContentableTypePost {
		return decimal.Zero, false, domainwallet.ErrIdempotencyConflict
	}

	var persisted persistedTipDetails
	if err := json.Unmarshal(detail.Details, &persisted); err != nil {
		return decimal.Zero, false, domainwallet.ErrIdempotencyConflict
	}
	persistedAmount, err := decimal.NewFromString(persisted.Amount)
	if err != nil || !persistedAmount.Equal(requestedAmount) {
		return decimal.Zero, false, domainwallet.ErrIdempotencyConflict
	}
	persistedPostPublicID, err := strconv.ParseInt(persisted.PostPublicID, 10, 64)
	if err != nil || persistedPostPublicID != postPublicID {
		return decimal.Zero, false, domainwallet.ErrIdempotencyConflict
	}
	resultingBalance, err := decimal.NewFromString(persisted.ResultingBalance)
	if err != nil || domainwallet.ValidateMoneyRepresentation(resultingBalance) != nil || resultingBalance.IsNegative() {
		return decimal.Zero, false, domainwallet.ErrIdempotencyConflict
	}
	return resultingBalance, true, nil
}

func (r *PostRepository) Tip(ctx context.Context, postId int64, authUser *models.User, amount decimal.Decimal, idempotencyKey domainwallet.IdempotencyKey) (*decimal.Decimal, error) {
	if authUser == nil || authUser.ID == uuid.Nil {
		return nil, errors.New(constants.ErrUnauthorized.String())
	}
	if r.userRepo == nil || r.userRepo.engagementRepo == nil {
		return &authUser.Balance, errors.New("tip repository dependencies are not configured")
	}
	if err := domainwallet.ValidateTipAmount(amount); err != nil {
		return &authUser.Balance, err
	}
	validatedKey, err := domainwallet.NewIdempotencyKey(idempotencyKey.String())
	if err != nil {
		return &authUser.Balance, err
	}
	dedupeKey := scopedTipDedupeKey(authUser.ID, validatedKey)
	replayedBalance, replayed, err := loadTipReplay(
		r.db.WithContext(ctx),
		dedupeKey,
		authUser.ID,
		postId,
		amount,
	)
	if err != nil {
		return &authUser.Balance, err
	}
	if replayed {
		authUser.Balance = replayedBalance
		return &authUser.Balance, nil
	}

	var payerBalance decimal.Decimal
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target post.Post
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("public_id = ?", postId).
			Where("published = TRUE").
			Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
			Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'").
			First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(constants.ErrPostNotFound.String())
			}
			return err
		}
		if target.AuthorID == authUser.ID {
			return errors.New(constants.ErrCannotTipOwnPost.String())
		}
		var lockedUsers []models.User
		if err := tx.Clauses(clause.Locking{Strength: "NO KEY UPDATE"}).
			Where("id IN ?", []uuid.UUID{authUser.ID, target.AuthorID}).
			Order("id ASC").
			Find(&lockedUsers).Error; err != nil {
			return err
		}
		if len(lockedUsers) != 2 {
			return errors.New(constants.ErrUserNotFound.String())
		}

		var payer, payee *models.User
		for i := range lockedUsers {
			switch lockedUsers[i].ID {
			case authUser.ID:
				payer = &lockedUsers[i]
			case target.AuthorID:
				payee = &lockedUsers[i]
			}
		}
		if payer == nil || payee == nil {
			return errors.New(constants.ErrUserNotFound.String())
		}

		replayedBalance, replayed, err := loadTipReplay(
			tx,
			dedupeKey,
			payer.ID,
			postId,
			amount,
		)
		if err != nil {
			return err
		}
		if replayed {
			payerBalance = replayedBalance
			return nil
		}

		transfer, err := domainwallet.NewTransfer(
			payer.ID,
			payee.ID,
			amount,
			domainwallet.MinimumTipAmount(),
			payer.Balance,
		)
		if err != nil {
			switch {
			case errors.Is(err, domainwallet.ErrSelfTransfer):
				return errors.New(constants.ErrCannotTipOwnPost.String())
			case errors.Is(err, domainwallet.ErrInsufficientFunds):
				return errors.New(constants.ErrInsufficientBalance.String())
			case errors.Is(err, domainwallet.ErrInvalidAmount),
				errors.Is(err, domainwallet.ErrAmountBelowMinimum),
				errors.Is(err, domainwallet.ErrAmountOutOfRange):
				return err
			default:
				return errors.New(constants.ErrInvalidAmount.String())
			}
		}

		newPayerBalance, newPayeeBalance := transfer.Apply(payer.Balance, payee.Balance)
		payerUpdate := tx.Model(&models.User{}).
			Where("id = ?", payer.ID).
			Update("balance", newPayerBalance)
		if payerUpdate.Error != nil {
			return payerUpdate.Error
		}
		if payerUpdate.RowsAffected != 1 {
			return errors.New(constants.ErrUserNotFound.String())
		}
		payeeUpdate := tx.Model(&models.User{}).
			Where("id = ?", payee.ID).
			Update("balance", newPayeeBalance)
		if payeeUpdate.Error != nil {
			return payeeUpdate.Error
		}
		if payeeUpdate.RowsAffected != 1 {
			return errors.New(constants.ErrUserNotFound.String())
		}

		if err := addTipInTransaction(
			tx,
			payer.ID,
			payee.ID,
			transfer.Amount(),
			target.ID,
			models.EngagementContentableTypePost,
			models.EngagementKindTip,
			withAmountEngagementDedupeKey(dedupeKey),
			withAmountEngagementDetail(tipResultingBalanceDetailKey, newPayerBalance.String()),
			withAmountEngagementDetail("post_public_id", strconv.FormatInt(postId, 10)),
		); err != nil {
			return err
		}

		payerBalance = newPayerBalance
		return nil
	})
	if errors.Is(err, errEngagementDetailAlreadyExists) {
		replayedBalance, replayed, replayErr := loadTipReplay(
			r.db.WithContext(ctx),
			dedupeKey,
			authUser.ID,
			postId,
			amount,
		)
		if replayErr != nil {
			return &authUser.Balance, replayErr
		}
		if !replayed {
			return &authUser.Balance, domainwallet.ErrIdempotencyConflict
		}
		payerBalance = replayedBalance
		err = nil
	}
	if err != nil {
		return &authUser.Balance, err
	}

	authUser.Balance = payerBalance
	return &authUser.Balance, nil
}

func (r *PostRepository) CreatePillar(ctx context.Context, pillar *taxonomy.Pillar) error {
	var existing taxonomy.Pillar

	pillar.Slug = helpers.GenerateSlug(pillar.Slug)

	err := r.db.WithContext(ctx).
		Where("slug = ?", pillar.Slug).
		First(&existing).Error

	if err == nil {
		return fmt.Errorf("pillar already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if pillar.ID == uuid.Nil {
		pillar.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(pillar).Error
}

func normalizeTaxonomySlug(raw string) string {
	return helpers.GenerateSlug(strings.TrimSpace(raw))
}

func normalizeTaxonomySearch(raw string) (string, string) {
	slug := normalizeTaxonomySlug(raw)
	return slug, strings.ToLower(helpers.SlugifyStrict(slug))
}

func activeTaxonomyPreload(db *gorm.DB) *gorm.DB {
	return db.Where("is_active = ?", true).
		Where("deleted_at IS NULL")
}

func orderedSynonymPreload(db *gorm.DB) *gorm.DB {
	return db.Order("is_primary DESC").
		Order("search_weight DESC").
		Order("slug ASC")
}

func rootClusterPreload(db *gorm.DB) *gorm.DB {
	return activeTaxonomyPreload(db).
		Where("parent_id IS NULL").
		Order("slug ASC")
}

func childClusterPreload(db *gorm.DB) *gorm.DB {
	return activeTaxonomyPreload(db).
		Order("slug ASC")
}

func taxonomyPillarTreeQuery(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Clusters", rootClusterPreload).
		Preload("Clusters.Synonyms", orderedSynonymPreload).
		Preload("Clusters.Children", childClusterPreload).
		Preload("Clusters.Children.Synonyms", orderedSynonymPreload)
}

func exactClusterLookupQuery(tx *gorm.DB, pillarID uuid.UUID, parentID *uuid.UUID, slug string) *gorm.DB {
	slug = normalizeTaxonomySlug(slug)

	query := tx.Where("pillar_id = ?", pillarID).
		Where("slug = ?", slug).
		Where("deleted_at IS NULL")

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	return query
}

func taxonomyPillarMatchQuery(tx *gorm.DB, slug string) *gorm.DB {
	normalizedSlug, strictSlug := normalizeTaxonomySearch(slug)
	if normalizedSlug == "" {
		return activeTaxonomyPreload(tx)
	}

	likePattern := "%" + strictSlug + "%"

	return activeTaxonomyPreload(tx).
		Where(`
			pillars.slug = ?
			OR EXISTS (
				SELECT 1
				FROM clusters
				WHERE clusters.pillar_id = pillars.id
				  AND clusters.is_active = true
				  AND clusters.deleted_at IS NULL
				  AND (
					clusters.slug = ?
					OR clusters.search_vector ILIKE ?
					OR EXISTS (
						SELECT 1
						FROM synonyms
						WHERE synonyms.cluster_id = clusters.id
						  AND synonyms.slug = ?
					)
				  )
			)
		`, normalizedSlug, normalizedSlug, likePattern, normalizedSlug)
}

func applyTaxonomyCategoryFilter(query *gorm.DB, category *string) *gorm.DB {
	if category == nil {
		return query
	}

	normalizedSlug, strictSlug := normalizeTaxonomySearch(*category)
	if normalizedSlug == "" {
		return query
	}

	likePattern := "%" + strictSlug + "%"

	return query.Where(`
		EXISTS (
			SELECT 1
			FROM post_clusters
			JOIN clusters ON clusters.id = post_clusters.cluster_id
			JOIN pillars ON pillars.id = clusters.pillar_id
			LEFT JOIN synonyms ON synonyms.cluster_id = clusters.id
			WHERE post_clusters.post_id = posts.id
			  AND clusters.is_active = true
			  AND clusters.deleted_at IS NULL
			  AND pillars.is_active = true
			  AND pillars.deleted_at IS NULL
			  AND (
				pillars.slug = ?
				OR clusters.slug = ?
				OR clusters.search_vector ILIKE ?
				OR synonyms.slug = ?
			  )
		)
	`, normalizedSlug, normalizedSlug, likePattern, normalizedSlug)
}

func (r *PostRepository) CreateCluster(ctx context.Context, cluster *taxonomy.Cluster) error {
	var existing taxonomy.Cluster

	cluster.Slug = normalizeTaxonomySlug(cluster.Slug)

	err := exactClusterLookupQuery(r.db.WithContext(ctx), cluster.PillarID, cluster.ParentID, cluster.Slug).
		First(&existing).Error
	if err == nil {
		return fmt.Errorf("cluster already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if cluster.ID == uuid.Nil {
		cluster.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *PostRepository) CreateSynonym(ctx context.Context, synonym *taxonomy.Synonym) error {
	var existing taxonomy.Synonym
	synonym.Slug = normalizeTaxonomySlug(synonym.Slug)
	err := r.db.WithContext(ctx).
		Where("cluster_id = ?", synonym.ClusterID).
		Where("slug = ?", synonym.Slug).
		First(&existing).Error

	if err == nil {
		return fmt.Errorf("synonym already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if synonym.ID == uuid.Nil {
		synonym.ID = uuid.New()
	}

	if synonym.SearchWeight <= 0 {
		synonym.SearchWeight = 1
	}

	if synonym.IsPrimary {
		var primaryCount int64
		if err := r.db.WithContext(ctx).
			Model(&taxonomy.Synonym{}).
			Where("cluster_id = ? AND is_primary = ?", synonym.ClusterID, true).
			Count(&primaryCount).Error; err != nil {
			return err
		}

		if primaryCount > 0 {
			return fmt.Errorf("primary synonym already exists for this cluster")
		}
	}

	synonym.CreatedAt = time.Now()

	return r.db.WithContext(ctx).Create(synonym).Error
}

func (r *PostRepository) PillarExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	slug = helpers.GenerateSlug(slug)

	err := r.db.WithContext(ctx).
		Model(&taxonomy.Pillar{}).
		Where("slug = ?", slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostRepository) GetPillarBySlug(ctx context.Context, slug string) (*taxonomy.Pillar, error) {
	var pillar taxonomy.Pillar
	slug = helpers.GenerateSlug(slug)

	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&pillar).Error

	if err != nil {
		return nil, err
	}

	return &pillar, nil
}

func (r *PostRepository) GetOrCreatePillar(ctx context.Context, slug string, name utils.LocalizedString) (*taxonomy.Pillar, error) {
	var pillar taxonomy.Pillar
	slug = helpers.GenerateSlug(slug)

	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&pillar).Error

	if err == nil {
		return &pillar, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	pillar = taxonomy.Pillar{
		ID:   uuid.New(),
		Slug: slug,
		Name: name,
	}

	if err := r.db.WithContext(ctx).Create(&pillar).Error; err != nil {
		return nil, err
	}

	return &pillar, nil
}

func (r *PostRepository) ClusterExists(ctx context.Context, pillarID uuid.UUID, parentID *uuid.UUID, slug string) (bool, error) {
	_, err := r.GetCluster(ctx, pillarID, parentID, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *PostRepository) FindClusterBySlug(ctx context.Context, pillarID uuid.UUID, slug string) (*taxonomy.Cluster, error) {
	var clusters []taxonomy.Cluster

	err := r.db.WithContext(ctx).
		Where("pillar_id = ?", pillarID).
		Where("slug = ?", normalizeTaxonomySlug(slug)).
		Where("deleted_at IS NULL").
		Limit(2).
		Find(&clusters).Error
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(clusters) > 1 {
		return nil, fmt.Errorf("cluster slug %q is ambiguous within pillar %s", slug, pillarID)
	}

	return &clusters[0], nil
}

func (r *PostRepository) GetCluster(ctx context.Context, pillarID uuid.UUID, parentID *uuid.UUID, slug string) (*taxonomy.Cluster, error) {
	var cluster taxonomy.Cluster
	if err := exactClusterLookupQuery(r.db.WithContext(ctx), pillarID, parentID, slug).
		First(&cluster).Error; err != nil {
		return nil, err
	}

	return &cluster, nil
}

func (r *PostRepository) GetOrCreateCluster(ctx context.Context, pillarID uuid.UUID, parentID *uuid.UUID, slug string, name utils.LocalizedString) (*taxonomy.Cluster, error) {

	var cluster taxonomy.Cluster
	slug = normalizeTaxonomySlug(slug)

	err := exactClusterLookupQuery(r.db.WithContext(ctx), pillarID, parentID, slug).
		First(&cluster).Error
	if err == nil {
		return &cluster, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cluster = taxonomy.Cluster{
		ID:       uuid.New(),
		PillarID: pillarID,
		ParentID: parentID,
		Slug:     slug,
		Name:     name,
		IsActive: true,
	}

	if err := r.db.WithContext(ctx).Create(&cluster).Error; err != nil {
		return nil, err
	}

	return &cluster, nil
}

func (r *PostRepository) SynonymExists(ctx context.Context, clusterID uuid.UUID, slug string) (bool, error) {
	var count int64
	slug = normalizeTaxonomySlug(slug)

	err := r.db.WithContext(ctx).
		Model(&taxonomy.Synonym{}).
		Where("cluster_id = ? AND slug = ?", clusterID, slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostRepository) GetSynonym(ctx context.Context, clusterID uuid.UUID, slug string) (*taxonomy.Synonym, error) {
	var synonym taxonomy.Synonym
	slug = normalizeTaxonomySlug(slug)

	if err := r.db.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		Where("slug = ?", slug).
		First(&synonym).Error; err != nil {
		return nil, err
	}

	return &synonym, nil
}

func (r *PostRepository) GetOrCreateSynonym(ctx context.Context, clusterID uuid.UUID, slug string, word utils.LocalizedString, isPrimary bool, weight int) (*taxonomy.Synonym, error) {

	var synonym taxonomy.Synonym
	slug = normalizeTaxonomySlug(slug)

	err := r.db.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		Where("slug = ?", slug).
		First(&synonym).Error

	if err == nil {
		return &synonym, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	synonym = taxonomy.Synonym{
		ID:           uuid.New(),
		ClusterID:    clusterID,
		Slug:         slug,
		Word:         word,
		IsPrimary:    isPrimary,
		SearchWeight: weight,
	}

	if err := r.db.WithContext(ctx).Create(&synonym).Error; err != nil {
		return nil, err
	}

	return &synonym, nil
}

func (r *PostRepository) GetPillars(ctx context.Context) ([]taxonomy.Pillar, error) {
	var pillars []taxonomy.Pillar

	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("updated_at DESC").
		Find(&pillars).Error

	return pillars, err
}

func (r *PostRepository) GetClusters(ctx context.Context) ([]taxonomy.Cluster, error) {
	var clusters []taxonomy.Cluster

	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("updated_at DESC").
		Find(&clusters).Error

	return clusters, err
}

func (r *PostRepository) GetPillarsWithClusters(filters types.Filter) ([]taxonomy.Pillar, error) {
	var pillars []taxonomy.Pillar

	err := taxonomyPillarTreeQuery(
		activeTaxonomyPreload(
			r.db.WithContext(filters.Context).Model(&taxonomy.Pillar{}),
		),
	).
		Where("domain = ?", *filters.Domain).
		Order("pillars.slug ASC").
		Find(&pillars).Error

	if err != nil {
		return nil, err
	}

	return pillars, nil
}

func (r *PostRepository) GetPillarsWithClustersWithSlug(ctx context.Context, slug string) ([]taxonomy.Pillar, error) {
	var pillars []taxonomy.Pillar

	err := taxonomyPillarTreeQuery(
		taxonomyPillarMatchQuery(
			r.db.WithContext(ctx).Model(&taxonomy.Pillar{}),
			slug,
		),
	).
		Order("pillars.slug ASC").
		Find(&pillars).Error

	if err != nil {
		return nil, err
	}

	return pillars, nil
}
