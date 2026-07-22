package db

import (
	"core/constants"
	"core/models"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestUserIdentityIndexDefinitionsAreCaseInsensitiveAndSoftDeleteAware(t *testing.T) {
	definitions := userIdentityIndexDefinitions()
	if len(definitions) != 2 {
		t.Fatalf("user identity index count = %d, want 2", len(definitions))
	}

	byName := make(map[string]IndexDefinition, len(definitions))
	for _, definition := range definitions {
		byName[definition.Name] = definition
	}

	username := byName["uidx_users_active_user_name_ci"]
	if !username.Unique || strings.Join(username.Columns, ",") != "LOWER(user_name)" {
		t.Fatalf("unexpected username identity index: %#v", username)
	}
	if username.Table != "users" || username.Condition != "deleted_at IS NULL" {
		t.Fatalf("username identity index is not active-user scoped: %#v", username)
	}

	email := byName["uidx_users_active_email_ci"]
	if !email.Unique || strings.Join(email.Columns, ",") != "LOWER(email)" {
		t.Fatalf("unexpected email identity index: %#v", email)
	}
	for _, fragment := range []string{"deleted_at IS NULL", "NULLIF(BTRIM(email), '') IS NOT NULL"} {
		if !strings.Contains(email.Condition, fragment) {
			t.Fatalf("email identity index condition is missing %q: %q", fragment, email.Condition)
		}
	}
}

func TestMigrateUserIdentityIndexesEnforcesActiveCaseInsensitiveUniquenessIntegration(t *testing.T) {
	database := migrationIntegrationDB(t).Session(&gorm.Session{SkipDefaultTransaction: true})
	if !database.Migrator().HasTable(&models.User{}) {
		t.Skip("users table is not migrated in TEST_DATABASE_URL")
	}

	if err := MigrateUserIdentityIndexes(database); err != nil {
		t.Fatalf("first user identity index migration: %v", err)
	}
	if err := MigrateUserIdentityIndexes(database); err != nil {
		t.Fatalf("idempotent user identity index migration: %v", err)
	}

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	basePublicID := time.Now().UTC().UnixNano()
	newUser := func(offset int64, username, email string) models.User {
		return models.User{
			ID:          uuid.New(),
			PublicID:    basePublicID + offset,
			Domain:      models.CoolVibes,
			UserName:    username,
			DisplayName: "Identity Index",
			Email:       email,
			UserRole:    constants.UserRoleUser,
		}
	}

	username := "CaseUser-" + suffix
	email := "CaseEmail-" + suffix + "@Example.COM"
	original := newUser(1, username, email)
	if err := database.Omit(clause.Associations).Create(&original).Error; err != nil {
		t.Fatalf("create original user: %v", err)
	}

	duplicateUsername := newUser(2, strings.ToUpper(username), "other-"+suffix+"@example.com")
	expectUserCreateConstraintFailure(t, database, "duplicate_username", &duplicateUsername)

	duplicateEmail := newUser(3, "other-"+suffix, strings.ToLower(email))
	expectUserCreateConstraintFailure(t, database, "duplicate_email", &duplicateEmail)

	if err := database.Delete(&original).Error; err != nil {
		t.Fatalf("soft-delete original user: %v", err)
	}
	replacement := newUser(4, strings.ToUpper(username), strings.ToLower(email))
	if err := database.Omit(clause.Associations).Create(&replacement).Error; err != nil {
		t.Fatalf("reuse soft-deleted username and email: %v", err)
	}

	firstWithoutEmail := newUser(5, "no-email-a-"+suffix, "")
	secondWithoutEmail := newUser(6, "no-email-b-"+suffix, "   ")
	if err := database.Omit(clause.Associations).Create(&firstWithoutEmail).Error; err != nil {
		t.Fatalf("create first email-less user: %v", err)
	}
	if err := database.Omit(clause.Associations).Create(&secondWithoutEmail).Error; err != nil {
		t.Fatalf("create second email-less user: %v", err)
	}
}

func expectUserCreateConstraintFailure(t *testing.T, database *gorm.DB, savepoint string, user *models.User) {
	t.Helper()
	if err := database.SavePoint(savepoint).Error; err != nil {
		t.Fatalf("create savepoint %q: %v", savepoint, err)
	}
	createErr := database.Omit(clause.Associations).Create(user).Error
	if rollbackErr := database.RollbackTo(savepoint).Error; rollbackErr != nil {
		t.Fatalf("rollback savepoint %q after error %v: %v", savepoint, createErr, rollbackErr)
	}
	if createErr == nil {
		t.Fatalf("create user at savepoint %q unexpectedly bypassed unique identity index", savepoint)
	}
}
