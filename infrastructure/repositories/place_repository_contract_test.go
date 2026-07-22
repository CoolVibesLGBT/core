package repositories

import (
	"core/application/types"
	"core/constants"
	"core/models/post"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNearByPlacesQueryWithoutCoordinatesUsesPublicIDCursor(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &PlaceRepository{db: db}
	cursor := int64(1001)

	var posts []post.Post
	tx := repo.nearByPlacesQuery(types.Filter{Cursor: &cursor, Limit: 12}, 12).Find(&posts)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"posts.contentable_type",
		"posts.published",
		"parent_id IS NULL",
		"posts.public_id <",
		"ORDER BY posts.public_id DESC",
		"LIMIT",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
	if strings.Contains(sql, "ST_Distance") {
		t.Fatalf("expected non-location query without ST_Distance, got %s", sql)
	}
}

func TestNearByPlaceLimitIsBounded(t *testing.T) {
	for _, test := range []struct {
		requested int
		want      int
	}{
		{requested: -1, want: constants.DEFAULT_LIMIT},
		{requested: 0, want: constants.DEFAULT_LIMIT},
		{requested: 12, want: 12},
		{requested: constants.MAXIMUM_LIMIT + 1, want: constants.MAXIMUM_LIMIT},
	} {
		if got := nearByPlaceLimit(test.requested); got != test.want {
			t.Fatalf("nearByPlaceLimit(%d) = %d, want %d", test.requested, got, test.want)
		}
	}
}

func TestNearByPlacesQueryWithCoordinatesUsesDistanceCursor(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &PlaceRepository{db: db}
	cursor := int64(1001)
	distance := 250.5
	lat := 41.0082
	lon := 28.9784
	domain := "coolvibes"

	var ranks []nearByPlaceRank
	tx := repo.nearByPlaceRanksQuery(types.Filter{
		Cursor:    &cursor,
		Distance:  &distance,
		Latitude:  &lat,
		Longitude: &lon,
		Domain:    &domain,
		Limit:     12,
	}, 12).Find(&ranks)
	if tx.Error != nil {
		t.Fatalf("query error = %v", tx.Error)
	}

	sql := tx.Statement.SQL.String()
	for _, fragment := range []string{
		"JOIN posts ON posts.id = locations.contentable_id",
		"locations.contentable_type",
		"locations.deleted_at IS NULL",
		"locations.location_point IS NOT NULL",
		"posts.deleted_at IS NULL",
		"posts.domain",
		"<->",
		"ST_MakePoint",
		"posts.public_id <",
		"ORDER BY distance ASC",
		"posts.public_id DESC",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("expected SQL to contain %q, got %s", fragment, sql)
		}
	}
	for _, forbidden := range []string{"LEFT JOIN locations", "ST_Distance", "ST_GeomFromText"} {
		if strings.Contains(sql, forbidden) {
			t.Fatalf("expected indexable spatial SQL without %q, got %s", forbidden, sql)
		}
	}
	vars := stringifyVars(tx.Statement.Vars)
	for _, value := range []string{"post", "coolvibes", "250.5", "1001", "41.0082", "28.9784"} {
		if !strings.Contains(vars, value) {
			t.Fatalf("expected query vars to contain %q, got %s", value, vars)
		}
	}
}

func TestNearByPlaceDetailsQueryOnlyPreloadsCardRelations(t *testing.T) {
	db := newDryRunTaxonomyDB(t)
	repo := &PlaceRepository{db: db}

	query := repo.nearByPlaceDetailsQuery(types.Filter{})

	for _, relation := range []string{
		"Clusters",
		"Location",
		"Engagements",
		"Author",
		"Author.Avatar",
		"Author.Avatar.File",
		"Hashtags",
		"Attachments",
		"Attachments.File",
	} {
		if _, ok := query.Statement.Preloads[relation]; !ok {
			t.Fatalf("expected %q preload", relation)
		}
	}

	for _, relation := range []string{
		"Poll",
		"Event",
		"Mentions",
		"Author.Cover",
		"Engagements.EngagementDetails",
		"Clusters.Parent",
		"Clusters.Children",
		"Clusters.Synonyms",
	} {
		if _, ok := query.Statement.Preloads[relation]; ok {
			t.Fatalf("did not expect expensive place-card preload %q", relation)
		}
	}

	attachments := query.Statement.Preloads["Attachments"]
	if len(attachments) != 1 || attachments[0] != "is_public = TRUE" {
		t.Fatalf("expected public attachment filter, got %#v", attachments)
	}
}

func TestOrderRankedPlacesPreservesSpatialOrderAndCursorDistance(t *testing.T) {
	nearID := uuid.New()
	farID := uuid.New()
	missingID := uuid.New()
	near := &post.Post{ID: nearID, PublicID: 101}
	far := &post.Post{ID: farID, PublicID: 99}

	ordered, lastDistance := orderRankedPlaces(
		[]*post.Post{far, near},
		[]nearByPlaceRank{
			{ID: nearID, PublicID: near.PublicID, Distance: 12.5},
			{ID: missingID, PublicID: 100, Distance: 25},
			{ID: farID, PublicID: far.PublicID, Distance: 40.25},
		},
	)

	if len(ordered) != 2 || ordered[0] != near || ordered[1] != far {
		t.Fatalf("unexpected ranked order: %#v", ordered)
	}
	if lastDistance == nil || *lastDistance != 40.25 {
		t.Fatalf("last distance = %#v, want 40.25", lastDistance)
	}
}
