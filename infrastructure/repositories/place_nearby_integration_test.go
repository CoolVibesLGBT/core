package repositories

import (
	"context"
	"core/application/types"
	"core/models"
	"core/models/post"
	"core/models/utils"
	"testing"
)

func TestGetNearByPlacesUsesStableSpatialCursorIntegration(t *testing.T) {
	database := eventRSVPIntegrationDB(t)
	for _, table := range []interface{}{&post.Post{}, &utils.Location{}} {
		if !database.Migrator().HasTable(table) {
			t.Skip("nearby place schema is not migrated in TEST_DATABASE_URL")
		}
	}

	latitude := 41.0082
	longitude := 28.9784
	domain := string(models.CoolVibes)
	repo := &PlaceRepository{db: database}

	firstPage, cursor, err := repo.GetNearByPlaces(types.Filter{
		Context:   context.Background(),
		Latitude:  &latitude,
		Longitude: &longitude,
		Domain:    &domain,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("first spatial page: %v", err)
	}
	if len(firstPage) == 0 {
		t.Skip("TEST_DATABASE_URL has no published place fixtures")
	}
	if len(firstPage) > 5 || cursor.Next == nil || cursor.Distance == nil {
		t.Fatalf("first page size/cursor = %d, %#v", len(firstPage), cursor)
	}

	firstIDs := make(map[int64]struct{}, len(firstPage))
	for _, place := range firstPage {
		firstIDs[place.PublicID] = struct{}{}
		if place.Location == nil || place.Location.DeletedAt.Valid {
			t.Fatalf("place %d has no active location: %#v", place.PublicID, place.Location)
		}
		for _, attachment := range place.Attachments {
			if !attachment.IsPublic {
				t.Fatalf("place %d exposed private attachment %s", place.PublicID, attachment.ID)
			}
		}
	}

	values, ok := types.DecodePaginationCursor(*cursor.Next)
	if !ok {
		t.Fatalf("decode next cursor %q", *cursor.Next)
	}
	publicID, ok := types.CursorPublicID(values)
	if !ok {
		t.Fatalf("cursor has no public id: %#v", values)
	}
	distance, ok := types.CursorDistance(values)
	if !ok {
		t.Fatalf("cursor has no distance: %#v", values)
	}

	secondPage, _, err := repo.GetNearByPlaces(types.Filter{
		Context:   context.Background(),
		Latitude:  &latitude,
		Longitude: &longitude,
		Domain:    &domain,
		Cursor:    &publicID,
		Distance:  &distance,
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("second spatial page: %v", err)
	}
	for _, place := range secondPage {
		if _, duplicate := firstIDs[place.PublicID]; duplicate {
			t.Fatalf("place %d was repeated across spatial pages", place.PublicID)
		}
	}
}
