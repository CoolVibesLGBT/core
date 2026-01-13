package repositories

import (
	"context"
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/post"
	"coolvibes/models/utils"
	"coolvibes/types"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlaceRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	postRepo         *PostRepository
	notificationRepo *NotificationRepository
}

func (r *PlaceRepository) DB() *gorm.DB {
	return r.db
}

func (r *PlaceRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func (r *PlaceRepository) MediaRepo() *MediaRepository {
	return r.mediaRepo
}

func (r *PlaceRepository) UserRepo() *UserRepository {
	return r.userRepo
}

func (r *PlaceRepository) NotificationRepo() *NotificationRepository {
	return r.notificationRepo
}

func (r *PlaceRepository) PostRepo() *PostRepository {
	return r.postRepo
}

func NewPlaceRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository, postRepo *PostRepository) *PlaceRepository {
	return &PlaceRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo, postRepo: postRepo}
}

func (r *PlaceRepository) Create(place *post.Post) error {
	return r.postRepo.CreatePost(place)
}

func (r *PlaceRepository) GetPlaceByID(id uuid.UUID) (*post.Post, error) {
	return r.postRepo.GetPostByID(id)
}

func (r *PlaceRepository) ExistsBySourceAndPlaceSourceID(
	source string,
	placeSourceID string,
) (bool, error) {

	var exists bool

	err := r.db.Raw(`
        SELECT EXISTS (
            SELECT 1
            FROM posts
            WHERE  extras -> 'place' ->> 'Source' = ?
              AND extras -> 'place' ->> 'SourceID' = ?
        )
    `, source, placeSourceID).Scan(&exists).Error

	return exists, err
}

func (r *PlaceRepository) GetNearByPlaces(ctx context.Context, authUser *models.User, lat *float64, lon *float64, cursor *int64, limit int) ([]*post.Post, types.Cursor, error) {
	var posts []*post.Post
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}

	if lat != nil && lon != nil {
		fmt.Println("Latitude", *lat, "-- Longitude", *lon)
	} else {
		fmt.Println("Latitude or Longitude is nil")
	}

	query := r.db.Model(&post.Post{}).
		Where("posts.contentable_type IN ?", []string{string(post.PostKindPost), string(post.PostKindPlace)}).
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
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Hashtags").
		Preload("Mentions").
		Preload("Attachments").
		Preload("Attachments.File")

	if cursor != nil {
		query = query.Where("public_id < ?", *cursor)
	}

	if lat != nil && lon != nil {
		userPoint := fmt.Sprintf("POINT(%f %f)", *lon, *lat)

		query = query.
			Joins(`
            LEFT JOIN locations 
            ON locations.contentable_id = posts.id
            AND locations.contentable_type = ?
        `, utils.LocationOwnerPost).
			Order(fmt.Sprintf(`
            ST_Distance(
                locations.location_point,
                ST_SetSRID(ST_GeomFromText('%s'), 4326)
            ) ASC,
            posts.public_id DESC
        `, userPoint))
	} else {
		query = query.Order("posts.public_id DESC")
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, types.Cursor{}, err
	}

	var prevCursor *string
	if cursor != nil {
		s := strconv.FormatInt(*cursor, 10)
		prevCursor = &s
	}

	var nextCursor *string
	if len(posts) > 0 {
		s := strconv.FormatInt(int64(posts[len(posts)-1].PublicID), 10)
		nextCursor = &s
	}

	resultCursor := types.Cursor{
		Prev: prevCursor,
		Next: nextCursor,
	}

	return posts, resultCursor, nil
}
