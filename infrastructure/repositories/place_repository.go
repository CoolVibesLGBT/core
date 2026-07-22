package repositories

import (
	"context"
	"core/application/types"
	"core/constants"
	"core/helpers"
	"core/models"
	"core/models/post"
	"core/models/taxonomy"
	"core/models/utils"
	"fmt"
	"strings"

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
	limit := nearByPlaceLimit(filters.Limit)

	lat := filters.Latitude
	lon := filters.Longitude
	var places []*post.Post

	var nextCursor *string
	var nextDistance *float64

	if lat != nil && lon != nil {
		if filters.Cursor != nil && filters.Distance == nil {
			return nil, types.Cursor{}, fmt.Errorf("distance cursor is required for location-based pagination")
		}
		ranks, err := r.fetchNearByPlaceRanks(filters, limit)
		if err != nil {
			return nil, types.Cursor{}, err
		}

		places, nextDistance, err = r.loadRankedPlaces(filters, ranks)
		if err != nil {
			return nil, types.Cursor{}, err
		}
	} else {
		if err := r.nearByPlacesQuery(filters, limit).Find(&places).Error; err != nil {
			return nil, types.Cursor{}, err
		}
	}

	if len(places) > 0 {
		last := places[len(places)-1]

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

	return places, types.Cursor{
		Prev:     prevCursor,
		Next:     nextCursor,
		Distance: nextDistance,
	}, nil
}

func nearByPlaceLimit(requested int) int {
	if requested <= 0 {
		return constants.DEFAULT_LIMIT
	}
	if requested > constants.MAXIMUM_LIMIT {
		return constants.MAXIMUM_LIMIT
	}
	return requested
}

type nearByPlaceRank struct {
	ID       uuid.UUID `gorm:"column:id"`
	PublicID int64     `gorm:"column:public_id"`
	Distance float64   `gorm:"column:distance"`
}

const nearByPlaceDistanceSQL = `
	locations.location_point <->
	ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
`

func (r *PlaceRepository) fetchNearByPlaceRanks(filters types.Filter, limit int) ([]nearByPlaceRank, error) {
	var ranks []nearByPlaceRank
	query := r.nearByPlaceRanksQuery(filters, limit)
	if err := query.Scan(&ranks).Error; err != nil {
		return nil, err
	}
	return ranks, nil
}

func (r *PlaceRepository) nearByPlaceRanksQuery(filters types.Filter, limit int) *gorm.DB {
	db := r.db
	if filters.Context != nil {
		db = db.WithContext(filters.Context)
	}

	query := db.
		Table("locations").
		Select(
			"posts.id, posts.public_id, "+nearByPlaceDistanceSQL+" AS distance",
			*filters.Longitude,
			*filters.Latitude,
		).
		Joins("JOIN posts ON posts.id = locations.contentable_id").
		Where("locations.contentable_type = ?", utils.LocationOwnerPost).
		Where("locations.deleted_at IS NULL").
		Where("locations.location_point IS NOT NULL").
		Where("posts.deleted_at IS NULL").
		Limit(limit)

	query = applyNearByPlaceVisibility(query)
	query = applyNearByPlaceDomain(query, filters.Domain)
	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil && filters.Distance != nil {
		query = query.Where(
			fmt.Sprintf("(%s > ?) OR (%s = ? AND posts.public_id < ?)", nearByPlaceDistanceSQL, nearByPlaceDistanceSQL),
			*filters.Longitude,
			*filters.Latitude,
			*filters.Distance,
			*filters.Longitude,
			*filters.Latitude,
			*filters.Distance,
			*filters.Cursor,
		)
	}

	return query.
		Order("distance ASC").
		Order("posts.public_id DESC")
}

func (r *PlaceRepository) loadRankedPlaces(filters types.Filter, ranks []nearByPlaceRank) ([]*post.Post, *float64, error) {
	if len(ranks) == 0 {
		return []*post.Post{}, nil, nil
	}

	ids := make([]uuid.UUID, 0, len(ranks))
	for _, rank := range ranks {
		ids = append(ids, rank.ID)
	}

	var unordered []*post.Post
	if err := r.nearByPlaceDetailsQuery(filters).
		Where("posts.id IN ?", ids).
		Find(&unordered).Error; err != nil {
		return nil, nil, err
	}

	ordered, lastDistance := orderRankedPlaces(unordered, ranks)
	return ordered, lastDistance, nil
}

func orderRankedPlaces(unordered []*post.Post, ranks []nearByPlaceRank) ([]*post.Post, *float64) {
	byID := make(map[uuid.UUID]*post.Post, len(unordered))
	for _, place := range unordered {
		byID[place.ID] = place
	}

	ordered := make([]*post.Post, 0, len(unordered))
	var lastDistance *float64
	for _, rank := range ranks {
		place, ok := byID[rank.ID]
		if !ok {
			continue
		}
		ordered = append(ordered, place)
		distance := rank.Distance
		lastDistance = &distance
	}

	return ordered, lastDistance
}

func (r *PlaceRepository) nearByPlacesQuery(filters types.Filter, limit int) *gorm.DB {
	query := r.nearByPlaceDetailsQuery(filters).
		Limit(limit)

	if filters.Cursor != nil {
		query = query.Where("posts.public_id < ?", *filters.Cursor)
	}

	return query.Order("posts.public_id DESC")
}

func (r *PlaceRepository) nearByPlaceDetailsQuery(filters types.Filter) *gorm.DB {
	db := r.db
	if filters.Context != nil {
		db = db.WithContext(filters.Context)
	}

	query := db.Model(&post.Post{})
	query = applyNearByPlaceVisibility(query)
	query = applyNearByPlaceDomain(query, filters.Domain)
	query = applyTaxonomyCategoryFilter(query, filters.Category)

	// A place card only consumes the place taxonomy, location, public media,
	// author identity, hashtags and aggregate counts. Poll/event trees and
	// per-user engagement details belong to their dedicated detail endpoints;
	// loading them here made one small page fan out into dozens of queries.
	return query.
		Preload("Clusters").
		Preload("Location").
		Preload("Engagements").
		Preload("Author", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "public_id", "user_name", "display_name", "avatar_id")
		}).
		Preload("Author.Avatar", "is_public = TRUE").
		Preload("Author.Avatar.File").
		Preload("Hashtags").
		Preload("Attachments", "is_public = TRUE").
		Preload("Attachments.File")
}

func applyNearByPlaceVisibility(query *gorm.DB) *gorm.DB {
	return query.
		Where("posts.contentable_type = ?", string(post.PostKindPlace)).
		Where("posts.published = TRUE").
		Where("COALESCE(NULLIF(posts.audience, ''), 'public') = 'public'").
		Where("posts.parent_id IS NULL")
}

func applyNearByPlaceDomain(query *gorm.DB, requestedDomain *string) *gorm.DB {
	if requestedDomain == nil {
		return query
	}

	domain := strings.TrimSpace(*requestedDomain)
	if domain == "" || strings.EqualFold(domain, string(models.AllDomains)) || strings.EqualFold(domain, string(models.UnknownDomain)) {
		return query
	}

	return query.Where("posts.domain = ?", domain)
}
