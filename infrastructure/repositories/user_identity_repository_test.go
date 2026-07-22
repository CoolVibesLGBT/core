package repositories

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	"core/models"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestActiveUserIdentityQueryIsCaseInsensitiveAndExcludesSoftDeletedUsers(t *testing.T) {
	database := newDryRunTaxonomyDB(t)

	tests := []struct {
		column string
		query  func(*gorm.DB, string) *gorm.DB
	}{
		{column: "user_name", query: activeUsernameIdentityQuery},
		{column: "email", query: activeEmailIdentityQuery},
	}
	for _, test := range tests {
		t.Run(test.column, func(t *testing.T) {
			var count int64
			query := test.query(database, "  MiXeD  ").Count(&count)
			if query.Error != nil {
				t.Fatalf("activeUserIdentityQuery() error = %v", query.Error)
			}

			sql := strings.ToLower(query.Statement.SQL.String())
			for _, fragment := range []string{
				"from \"users\"",
				"deleted_at is null",
				"lower(" + test.column + ") = lower(",
			} {
				if !strings.Contains(sql, fragment) {
					t.Fatalf("identity query is missing %q: %s", fragment, query.Statement.SQL.String())
				}
			}
			if len(query.Statement.Vars) != 1 || query.Statement.Vars[0] != "MiXeD" {
				t.Fatalf("identity query vars = %#v, want trimmed input", query.Statement.Vars)
			}
		})
	}
}

func TestExistsByEmailSkipsBlankOptionalIdentity(t *testing.T) {
	repository := NewUserRepository(nil, nil, nil, nil, nil)
	exists, err := repository.ExistsByEmail("   ")
	if err != nil || exists {
		t.Fatalf("ExistsByEmail(blank) = %v, %v; want false, nil", exists, err)
	}
}

func TestUserIdentityLookupMapsOnlyMissingRowsToPortNotFoundIntegration(t *testing.T) {
	database := engagementViewIntegrationDB(t)
	if !database.Migrator().HasTable(&models.User{}) {
		t.Skip("users table is not migrated in TEST_DATABASE_URL")
	}
	repository := NewUserRepository(database, nil, nil, nil, nil)

	if _, err := repository.GetUserByPublicIdWithoutRelations(types.Filter{Context: context.Background(), UserID: time.Now().UTC().UnixNano()}); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("public ID lookup error = %v, want ports.ErrNotFound", err)
	}
	if _, err := repository.GetUserByUUIDdWithoutRelations(types.Filter{Context: context.Background(), UserUUID: uuid.New()}); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("UUID lookup error = %v, want ports.ErrNotFound", err)
	}
}

func TestUserIdentityLookupsAreCaseInsensitiveAndIgnoreSoftDeletedUsersIntegration(t *testing.T) {
	database := engagementViewIntegrationDB(t)
	if !database.Migrator().HasTable(&models.User{}) {
		t.Skip("users table is not migrated in TEST_DATABASE_URL")
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	user := models.User{
		ID:          uuid.New(),
		PublicID:    time.Now().UTC().UnixNano(),
		Domain:      models.CoolVibes,
		UserName:    "Identity-" + suffix,
		DisplayName: "Identity Lookup",
		Email:       "Identity-" + suffix + "@Example.COM",
		UserRole:    constants.UserRoleUser,
	}
	if err := database.Omit(clause.Associations).Create(&user).Error; err != nil {
		t.Fatalf("create identity lookup user: %v", err)
	}

	repository := NewUserRepository(database, nil, nil, nil, nil)
	usernameExists, err := repository.ExistsByUsername(strings.ToUpper(user.UserName))
	if err != nil || !usernameExists {
		t.Fatalf("ExistsByUsername(case variant) = %v, %v; want true, nil", usernameExists, err)
	}
	emailExists, err := repository.ExistsByEmail(strings.ToLower(user.Email))
	if err != nil || !emailExists {
		t.Fatalf("ExistsByEmail(case variant) = %v, %v; want true, nil", emailExists, err)
	}

	if err := database.Delete(&user).Error; err != nil {
		t.Fatalf("soft-delete identity lookup user: %v", err)
	}
	usernameExists, err = repository.ExistsByUsername(user.UserName)
	if err != nil || usernameExists {
		t.Fatalf("ExistsByUsername(soft-deleted) = %v, %v; want false, nil", usernameExists, err)
	}
	emailExists, err = repository.ExistsByEmail(user.Email)
	if err != nil || emailExists {
		t.Fatalf("ExistsByEmail(soft-deleted) = %v, %v; want false, nil", emailExists, err)
	}
}
