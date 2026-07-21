package repositories

import (
	"context"
	"core/constants"
	"core/helpers"
	"core/models/post"
	"core/models/taxonomy"
	"core/models/utils"
	"core/types"
	"fmt"

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

func (r *PlaceRepository) GetPlacesCategories(filters types.Filter) ([]taxonomy.Pillar, error) {
	filters.Search = utils.StringPtr("places")
	return r.postRepo.GetPillarsWithClustersWithSlug(context.Background(), *filters.Search)
}

func (r *PlaceRepository) ExistsBySourceAndPlaceSourceID(source string, placeSourceID string) (bool, error) {
	var exists bool
	err := r.db.Raw(`
        SELECT EXISTS (
            SELECT 1
            FROM posts
            WHERE post_kind = ?
              AND extras -> 'place' ->> 'source' = ?
              AND extras -> 'place' ->> 'source_id' = ?
        )
    `, post.PostKindPlace, source, placeSourceID).Scan(&exists).Error
	return exists, err
}

func (r *PlaceRepository) GetNearByPlaces(filters types.Filter) ([]*post.Post, types.Cursor, error) {
	var posts []*post.Post
	limit := filters.Limit
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}

	lat := filters.Latitude
	lon := filters.Longitude

	query := r.nearByPlacesQuery(filters, limit)

	if err := query.Find(&posts).Error; err != nil {
		return nil, types.Cursor{}, err
	}

	var nextCursor *string
	var nextDistance *float64

	if len(posts) > 0 {

		last := posts[len(posts)-1]

		if lat != nil && lon != nil {
			var dist float64
			r.db.Raw(`
				SELECT ST_Distance(
					location_point::geography,
					ST_SetSRID(ST_GeomFromText(?), 4326)::geography
				)
				FROM locations
				WHERE contentable_id = ?
				AND contentable_type = ?
			`, fmt.Sprintf("POINT(%f %f)", *lon, *lat),
				last.ID,
				utils.LocationOwnerPost,
			).Scan(&dist)

			nextDistance = &dist
		}

		var cursorErr error
		nextCursor, cursorErr = types.NewPublicIDDistanceCursor(last.PublicID, nextDistance)
		if cursorErr != nil {
			return nil, types.Cursor{}, cursorErr
		}
	}

	var prevCursor *string
	if filters.Cursor != nil {
		var cursorErr error
		prevCursor, cursorErr = types.NewPublicIDDistanceCursor(*filters.Cursor, filters.Distance)
		if cursorErr != nil {
			return nil, types.Cursor{}, cursorErr
		}
	}

	return posts, types.Cursor{
		Prev:     prevCursor,
		Next:     nextCursor,
		Distance: nextDistance,
	}, nil
}

func (r *PlaceRepository) nearByPlacesQuery(filters types.Filter, limit int) *gorm.DB {
	cursorID := filters.Cursor
	cursorDistance := filters.Distance
	lat := filters.Latitude
	lon := filters.Longitude

	query := r.db.Model(&post.Post{}).
		Where("posts.contentable_type = ?", string(post.PostKindPlace)).
		Where("posts.published = TRUE").
		Where("COALESCE(NULLIF(posts.audience, ''), 'public') = 'public'").
		Where("parent_id IS NULL").
		Limit(limit).
		Preload("Clusters").
		Preload("Clusters.Pillar").
		Preload("Clusters.Synonyms").
		Preload("Clusters.Parent").
		Preload("Clusters.Children").
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

	if lat != nil && lon != nil {

		userPoint := fmt.Sprintf("POINT(%f %f)", *lon, *lat)

		distanceSQL := fmt.Sprintf(`
			ST_Distance(
				locations.location_point::geography,
				ST_SetSRID(ST_GeomFromText('%s'), 4326)::geography
			)
		`, userPoint)

		query = query.Joins(`
			LEFT JOIN locations 
			ON locations.contentable_id = posts.id
			AND locations.contentable_type = ?
		`, utils.LocationOwnerPost)

		// Cursor kesme (distance + id)
		if cursorID != nil && cursorDistance != nil {
			query = query.Where(fmt.Sprintf(`
				(%s > ?) OR
				(%s = ? AND posts.public_id < ?)
			`, distanceSQL, distanceSQL),
				*cursorDistance,
				*cursorDistance,
				*cursorID,
			)
		}

		query = query.
			Order(distanceSQL + " ASC").
			Order("posts.public_id DESC")

	} else {

		if cursorID != nil {
			query = query.Where("posts.public_id < ?", *cursorID)
		}

		query = query.Order("posts.public_id DESC")
	}

	return query
}
