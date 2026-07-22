package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"core/application/ports"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type sessionTokenDecoder struct {
	publicID int64
	err      error
}

func (d sessionTokenDecoder) DecodeUserPublicID(string) (int64, error) {
	return d.publicID, d.err
}

type sessionUserRepository struct {
	user              *ports.SessionUser
	err               error
	requestedPublicID int64
	locationUpdates   int
	updatedUserID     uuid.UUID
}

func (r *sessionUserRepository) GetSessionUserByPublicID(_ context.Context, publicID int64) (*ports.SessionUser, error) {
	r.requestedPublicID = publicID
	return r.user, r.err
}

func (r *sessionUserRepository) UpdateLocation(_ context.Context, userID uuid.UUID, _ string) error {
	r.locationUpdates++
	r.updatedUserID = userID
	return nil
}

var _ ports.SessionRepository = (*sessionUserRepository)(nil)

func TestSessionResolveUsesLightweightUserQuery(t *testing.T) {
	want := &ports.SessionUser{
		ID:               uuid.New(),
		PublicID:         77,
		Domain:           domainuser.CoolVibes,
		UserName:         "session-user",
		DisplayName:      "Session User",
		DefaultLanguage:  "tr",
		PreferencesFlags: "101",
		Role:             string(constants.UserRoleModerator),
		Balance:          decimal.RequireFromString("12.50"),
		HasLocation:      true,
	}
	repository := &sessionUserRepository{user: want}
	service := NewSessionService(repository, sessionTokenDecoder{publicID: 77})

	got, err := service.ResolveUser(context.Background(), "token")
	if err != nil {
		t.Fatalf("ResolveUser() error = %v", err)
	}
	if got == nil || got.User == nil || repository.requestedPublicID != 77 {
		t.Fatalf("ResolveUser() = %#v, requested %d", got, repository.requestedPublicID)
	}
	if got.User.ID != want.ID || got.User.PublicID != want.PublicID || got.User.Domain != models.CoolVibes ||
		got.User.UserName != want.UserName || got.User.DisplayName != want.DisplayName ||
		got.User.DefaultLanguage != want.DefaultLanguage || got.User.PreferencesFlags != want.PreferencesFlags ||
		got.User.UserRole != constants.UserRoleModerator || !got.User.Balance.Equal(want.Balance) {
		t.Fatalf("session mapper lost a hot-path field: %#v", got.User)
	}
	if got.User.Password != "" || got.User.Email != "" || got.User.Location != nil || !got.hasLocation {
		t.Fatalf("session mapper exposed a sensitive field or location marker: user=%#v has_location=%v", got.User, got.hasLocation)
	}
	encoded, err := json.Marshal(got.User)
	if err != nil {
		t.Fatalf("marshal session user: %v", err)
	}
	if strings.Contains(string(encoded), "has_location") || strings.Contains(string(encoded), `"location"`) {
		t.Fatalf("internal location marker leaked into session JSON: %s", encoded)
	}
}

func TestSessionResolveRejectsMissingRepositoryRecord(t *testing.T) {
	service := NewSessionService(&sessionUserRepository{}, sessionTokenDecoder{publicID: 77})
	resolved, err := service.ResolveUser(context.Background(), "token")
	if resolved != nil || !errors.Is(err, ErrSessionUserNotFound) {
		t.Fatalf("ResolveUser() = %#v, %v; want nil, ErrSessionUserNotFound", resolved, err)
	}
}

func TestSessionResolveRejectsBotsAndSuspendedAccountRoles(t *testing.T) {
	tests := []ports.SessionUser{
		{ID: uuid.New(), PublicID: 77, Role: string(constants.UserRoleUser), IsBot: true},
		{ID: uuid.New(), PublicID: 77, Role: string(constants.UserRoleBanned)},
		{ID: uuid.New(), PublicID: 77, Role: string(constants.UserRoleDeleted)},
		{ID: uuid.New(), PublicID: 77, Role: string(constants.UserRolePending)},
	}

	for _, account := range tests {
		name := account.Role
		if account.IsBot {
			name = "bot"
		}
		t.Run(name, func(t *testing.T) {
			repository := &sessionUserRepository{user: &account}
			service := NewSessionService(repository, sessionTokenDecoder{publicID: account.PublicID})

			resolved, err := service.ResolveUser(context.Background(), "previously-issued-token")
			if resolved != nil || !errors.Is(err, ErrSessionUserNotFound) {
				t.Fatalf("ResolveUser(%s) = %#v, %v; want rejected session", name, resolved, err)
			}
		})
	}
}

func TestSessionTrackLocationSkipsKnownOrMissingUser(t *testing.T) {
	repository := &sessionUserRepository{}
	service := NewSessionService(repository, sessionTokenDecoder{})

	if err := service.TrackLocation(context.Background(), nil, "127.0.0.1"); err != nil {
		t.Fatalf("TrackLocation(nil) error = %v", err)
	}
	withLocation := &ResolvedSession{User: &models.User{ID: uuid.New()}, hasLocation: true}
	if err := service.TrackLocation(context.Background(), withLocation, "127.0.0.1"); err != nil {
		t.Fatalf("TrackLocation(known) error = %v", err)
	}
	if repository.locationUpdates != 0 {
		t.Fatalf("known location caused %d writes", repository.locationUpdates)
	}

	withoutLocation := &ResolvedSession{User: &models.User{ID: uuid.New()}}
	if err := service.TrackLocation(context.Background(), withoutLocation, "127.0.0.1"); err != nil {
		t.Fatalf("TrackLocation(new) error = %v", err)
	}
	if repository.locationUpdates != 1 {
		t.Fatalf("new location caused %d writes, want 1", repository.locationUpdates)
	}
	if repository.updatedUserID != withoutLocation.User.ID || !withoutLocation.hasLocation {
		t.Fatalf("location update did not use/mark resolved session: %#v", withoutLocation)
	}
	if err := service.TrackLocation(context.Background(), withoutLocation, "127.0.0.1"); err != nil {
		t.Fatalf("TrackLocation(already updated) error = %v", err)
	}
	if repository.locationUpdates != 1 {
		t.Fatalf("resolved session repeated location write, got %d", repository.locationUpdates)
	}
}
