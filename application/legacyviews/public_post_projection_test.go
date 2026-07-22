package legacyviews

import (
	"core/constants"
	"core/models"
	"core/models/media"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	modelutils "core/models/utils"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

var canonicalUUIDPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}`)

func TestProjectPublicPostRemovesPersistenceAndAccountSecrets(t *testing.T) {
	postID := uuid.New()
	authorID := uuid.New()
	mediaID := uuid.New()
	fileID := uuid.New()
	pollID := uuid.New()
	choiceID := uuid.New()
	voteID := uuid.New()
	reportDetailID := uuid.New()
	locationID := uuid.New()
	secretPath := "/srv/coolvibes/private/original.jpg"
	secretName := "passport-original.jpg"
	secretBroadcast := `{"token":"broadcast-secret"}`
	localized := modelutils.LocalizedString{"en": "hello"}

	author := models.User{
		ID:               authorID,
		PublicID:         9001,
		UserName:         "public-author",
		DisplayName:      "Public Author",
		Email:            "secret@example.test",
		Password:         "password-hash",
		Balance:          decimal.NewFromInt(999),
		PreferencesFlags: "private-bits",
		UserRole:         constants.UserRoleAdmin,
		BroadcastInfo:    datatypes.JSON([]byte(secretBroadcast)),
		Avatar: &media.Media{
			ID:       mediaID,
			PublicID: 7001,
			IsPublic: true,
			Role:     media.RoleAvatar,
			FileID:   fileID,
			File: modelutils.FileMetadata{
				ID: fileID, URL: "https://cdn.test/avatar.jpg", StoragePath: secretPath, Name: secretName,
			},
		},
	}
	item := post.Post{
		ID:        postID,
		PublicID:  5001,
		AuthorID:  authorID,
		Author:    author,
		PostKind:  post.PostKindPost,
		Title:     &localized,
		Published: true,
		Location:  &modelutils.Location{ID: locationID, ContentableID: postID, Latitude: floatPtr(41.0), Longitude: floatPtr(29.0)},
		Attachments: []*media.Media{{
			ID: mediaID, PublicID: 7002, FileID: fileID, OwnerID: postID, UserID: authorID,
			OwnerType: media.OwnerPost, Role: media.RolePost, IsPublic: true,
			File: modelutils.FileMetadata{ID: fileID, URL: "https://cdn.test/post.jpg", StoragePath: secretPath, Name: secretName},
		}},
		Poll: []*postpayloads.Poll{{
			ID: pollID, Question: localized, Kind: postpayloads.PollKindSingle, MaxSelectable: 1,
			Choices: []postpayloads.PollChoice{{
				ID: choiceID, Label: localized, VoteCount: 1,
				Votes: []postpayloads.PollVote{{ID: voteID, ChoiceID: choiceID, UserID: authorID, User: author}},
			}},
		}},
		Engagements: &models.Engagement{
			ID: postID, ContentableID: postID, Counts: datatypes.JSON([]byte(`{"like_received_count":2,"report_count":1,"deposit_amount":"999"}`)),
			EngagementDetails: []models.EngagementDetail{{
				ID: reportDetailID, EngagementID: postID, EngagerID: authorID, Engager: author,
				Kind: models.EngagementKindReport, Details: datatypes.JSON([]byte(`{"description":"reporter secret"}`)),
			}},
		},
		CreatedAt: time.Now().UTC(),
	}

	encoded, err := json.Marshal(ProjectPublicPost(item))
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	payload := string(encoded)
	for _, secret := range []string{
		postID.String(), authorID.String(), mediaID.String(), fileID.String(), pollID.String(), choiceID.String(), voteID.String(),
		reportDetailID.String(), locationID.String(), "secret@example.test", "password-hash", "private-bits", "broadcast-secret",
		secretPath, secretName, "reporter secret", "report_count", "deposit_amount",
	} {
		if strings.Contains(payload, secret) {
			t.Fatalf("public projection leaked %q: %s", secret, payload)
		}
	}
	if canonicalUUIDPattern.Match(encoded) {
		t.Fatalf("public projection contains a canonical UUID: %s", payload)
	}
	for _, forbiddenKey := range []string{`"balance"`, `"user_role"`, `"preferences_flags"`, `"broadcast_info"`, `"storage_path"`, `"file_id"`, `"owner_id"`} {
		if strings.Contains(payload, forbiddenKey) {
			t.Fatalf("public projection contains forbidden key %s: %s", forbiddenKey, payload)
		}
	}
	if !strings.Contains(payload, `"id":"5001"`) || !strings.Contains(payload, `"author_id":"9001"`) {
		t.Fatalf("expected compatibility ids to use public identifiers: %s", payload)
	}
	if !strings.Contains(payload, `"id":"pc_`) {
		t.Fatalf("expected poll choice to use an opaque token: %s", payload)
	}
	if !strings.Contains(payload, `"like_received_count":2`) {
		t.Fatalf("expected safe engagement count to remain: %s", payload)
	}
}

func TestProjectPublicPostDropsNonPublicMedia(t *testing.T) {
	item := post.Post{
		PublicID: 1,
		Attachments: []*media.Media{
			{PublicID: 2, IsPublic: false, File: modelutils.FileMetadata{URL: "https://cdn.test/private.jpg"}},
			{PublicID: 3, IsPublic: true, File: modelutils.FileMetadata{URL: "https://cdn.test/public.jpg"}},
		},
	}
	projected := ProjectPublicPost(item)
	if len(projected.Attachments) != 1 || projected.Attachments[0].PublicID != 3 {
		t.Fatalf("attachments = %#v", projected.Attachments)
	}
}

func floatPtr(value float64) *float64 { return &value }
