package default_users

import (
	"core/constants"
	"core/helpers"
	"core/models"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedDefaultSystemUsers(db *gorm.DB, node *helpers.Node) error {
	if node == nil {
		return errors.New("snowflake node is required")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, defaultUserName := range constants.DefaultSystemUsers {
			defaultUserEmail := fmt.Sprintf("%s@coolvibes.lgbt", defaultUserName)
			desiredRole := defaultSystemRole(defaultUserName)

			var existing models.User
			err := tx.Unscoped().
				Where("LOWER(user_name) = ? AND LOWER(email) = ?", strings.ToLower(defaultUserName), strings.ToLower(defaultUserEmail)).
				First(&existing).Error
			if err == nil {
				// Never elevate an existing account implicitly. An operator can
				// grant a real account with -grant-admin/-grant-moderator.
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("find system user %s: %w", defaultUserName, err)
			}

			var collisionCount int64
			if err := tx.Unscoped().Model(&models.User{}).
				Where("LOWER(user_name) = ? OR LOWER(email) = ?", strings.ToLower(defaultUserName), strings.ToLower(defaultUserEmail)).
				Count(&collisionCount).Error; err != nil {
				return fmt.Errorf("check system user %s: %w", defaultUserName, err)
			}
			if collisionCount != 0 {
				return fmt.Errorf("canonical system identity %s conflicts with an existing account", defaultUserName)
			}

			hashedPassword, err := generateSystemPassword()
			if err != nil {
				return fmt.Errorf("generate password for system user %s: %w", defaultUserName, err)
			}

			userObj := &models.User{
				ID:          uuid.New(),
				PublicID:    node.Generate().Int64(),
				UserName:    defaultUserName,
				DisplayName: defaultUserName,
				Email:       defaultUserEmail,
				Password:    hashedPassword,
				Domain:      models.CoolVibes,
				UserRole:    desiredRole,
			}
			if err := tx.Create(userObj).Error; err != nil {
				return fmt.Errorf("create system user %s: %w", defaultUserName, err)
			}
		}
		return nil
	})
}

func defaultSystemRole(userName string) constants.UserRole {
	switch userName {
	case constants.SystemUserAdmin:
		return constants.UserRoleAdmin
	case constants.SystemUserModerator:
		return constants.UserRoleModerator
	default:
		return constants.UserRoleUser
	}
}

func generateSystemPassword() (string, error) {
	rawPassword, err := helpers.GenerateRandomPassword(16)
	if err != nil {
		return "", err
	}
	return helpers.HashPasswordArgon2id(rawPassword)
}
