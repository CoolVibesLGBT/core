package repositories

import (
	"container/heap"
	"core/application/types"
	domainuser "core/domain/user"
	"core/models"
	modelutils "core/models/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"math"
	"sort"
	"strings"
	"time"
)

type rankedMatch struct {
	id       uuid.UUID
	publicID int64
	online   bool
	score    float64
}
type matchHeap []rankedMatch

func (h matchHeap) Len() int           { return len(h) }
func (h matchHeap) Less(i, j int) bool { return matchIsWorse(h[i], h[j]) }
func (h matchHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *matchHeap) Push(v any)        { *h = append(*h, v.(rankedMatch)) }
func (h *matchHeap) Pop() any          { last := len(*h) - 1; v := (*h)[last]; *h = (*h)[:last]; return v }
func matchIsWorse(first, second rankedMatch) bool {
	if first.score != second.score {
		return first.score < second.score
	}
	if first.online != second.online {
		return !first.online
	}
	return first.publicID < second.publicID
}
func matchingInterests(user models.User) []string {
	groups := []struct {
		name   string
		values []string
	}{
		{"hobbies", user.Hobbies}, {"movies_genres", user.MoviesGenres}, {"tv_shows_genres", user.TVShowsGenres}, {"theater_genres", user.TheaterGenres}, {"cinema_genres", user.CinemaGenres}, {"art_interests", user.ArtInterests}, {"entertainment", user.Entertainment},
	}
	var values []string
	for _, group := range groups {
		for _, value := range group.values {
			if value = strings.TrimSpace(value); value != "" {
				values = append(values, group.name+":"+value)
			}
		}
	}
	return values
}
func matchingCoordinates(location *modelutils.Location) *domainuser.Coordinates {
	if location == nil || location.Latitude == nil || location.Longitude == nil {
		return nil
	}
	coordinates, err := domainuser.NewCoordinates(*location.Latitude, *location.Longitude)
	if err != nil {
		return nil
	}
	return &coordinates
}
func matchingDistance(first, second *domainuser.Coordinates) *float64 {
	if first == nil || second == nil {
		return nil
	}
	lat := math.Pi / 180
	deltaLat := (second.Latitude - first.Latitude) * lat
	deltaLon := (second.Longitude - first.Longitude) * lat
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) + math.Cos(first.Latitude*lat)*math.Cos(second.Latitude*lat)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	distance := 6371.0 * 2 * math.Asin(math.Sqrt(min(1.0, max(0.0, a))))
	return &distance
}
func matchingAgeAllowed(birth *time.Time, criteria domainuser.MatchCriteria, now time.Time) bool {
	if criteria.MinAge == nil && criteria.MaxAge == nil {
		return true
	}
	if birth == nil {
		return false
	}
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return (criteria.MinAge == nil || age >= *criteria.MinAge) && (criteria.MaxAge == nil || age <= *criteria.MaxAge)
}

// Scan score inputs in bounded GORM batches and retain only the best page.
// The score comes from the domain policy, not a second SQL implementation.
func rankMatchingUsers(query *gorm.DB, principal models.User, criteria domainuser.MatchCriteria, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		return []uuid.UUID{}, nil
	}
	best := &matchHeap{}
	heap.Init(best)
	principalInterests := matchingInterests(principal)
	principalCoordinates := matchingCoordinates(principal.Location)
	now := time.Now().UTC()
	scoreQuery := query.Select("users.id", "users.public_id", "users.is_online", "users.date_of_birth", "users.languages", "users.hobbies", "users.movies_genres", "users.tv_shows_genres", "users.theater_genres", "users.cinema_genres", "users.art_interests", "users.entertainment")
	var batch []models.User
	err := scoreQuery.FindInBatches(&batch, 200, func(tx *gorm.DB, _ int) error {
		ids := make([]uuid.UUID, len(batch))
		for i := range batch {
			ids[i] = batch[i].ID
		}
		var locations []modelutils.Location
		if err := repositorySubquery(tx).Model(&modelutils.Location{}).Select("id", "contentable_id", "latitude", "longitude").Where("contentable_type = ? AND contentable_id IN ?", modelutils.LocationOwnerUser, ids).Order("id ASC").Find(&locations).Error; err != nil {
			return err
		}
		coordinates := make(map[uuid.UUID]*domainuser.Coordinates, len(locations))
		for i := range locations {
			if _, exists := coordinates[locations[i].ContentableID]; !exists {
				coordinates[locations[i].ContentableID] = matchingCoordinates(&locations[i])
			}
		}
		for _, candidate := range batch {
			if !matchingAgeAllowed(candidate.DateOfBirth, criteria, now) {
				continue
			}
			distance := matchingDistance(principalCoordinates, coordinates[candidate.ID])
			if criteria.MaxDistanceKM != nil && (distance == nil || *distance > *criteria.MaxDistanceKM) {
				continue
			}
			rank := rankedMatch{id: candidate.ID, publicID: candidate.PublicID, online: candidate.IsOnline, score: domainuser.MatchCompatibility(principalInterests, matchingInterests(candidate), principal.Languages, candidate.Languages, distance)}
			if best.Len() < limit {
				heap.Push(best, rank)
			} else if matchIsWorse((*best)[0], rank) {
				heap.Pop(best)
				heap.Push(best, rank)
			}
		}
		return nil
	}).Error
	if err != nil {
		return nil, err
	}
	sort.Slice(*best, func(i, j int) bool { return matchIsWorse((*best)[j], (*best)[i]) })
	ids := make([]uuid.UUID, len(*best))
	for i, entry := range *best {
		ids[i] = entry.id
	}
	return ids, nil
}

func matchingSummaryPage(db *gorm.DB, actor uuid.UUID, ids []uuid.UUID) ([]types.PublicUserSummary, error) {
	if len(ids) == 0 {
		return []types.PublicUserSummary{}, nil
	}
	var rows []publicUserProjectionRow
	query := applyMatchPublicUserJoins(db.Table("users").Select(unseenPublicUserProjection).Where("users.id IN ?", ids).Where("users.domain = (?)", userDomainSubquery(db, actor)))
	if err := publicUserVisibilityScope(userBlockScope(query, actor, "users")).Scan(&rows).Error; err != nil {
		return nil, err
	}
	// Reapply the computed order after loading only the public response fields.
	var order []models.User
	if err := db.Model(&models.User{}).Select("id", "public_id").Where("id IN ?", ids).Find(&order).Error; err != nil {
		return nil, err
	}
	publicIDs := make(map[uuid.UUID]int64, len(order))
	for _, user := range order {
		publicIDs[user.ID] = user.PublicID
	}
	summaries := make(map[int64]types.PublicUserSummary, len(rows))
	for _, row := range rows {
		summaries[row.PublicID] = row.summaryView()
	}
	result := make([]types.PublicUserSummary, 0, len(ids))
	for _, id := range ids {
		if row, ok := summaries[publicIDs[id]]; ok {
			result = append(result, row)
		}
	}
	return result, nil
}
