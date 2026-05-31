package default_users

import (
	"core/constants"
	"core/helpers"
	"core/infrastructure/repositories"
	"core/models"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func SeedDefaultSystemUsers(db *gorm.DB, node *helpers.Node) error {
	engagementRepo := repositories.NewEngagementRepository(db)
	userRepo := repositories.NewUserRepository(db, nil, node, engagementRepo, nil)

	for _, defaultUserName := range constants.DefaultSystemUsers {
		defaultUserEmail := fmt.Sprintf("%s@coolvibes.lgbt", defaultUserName) // Email'i uygun şekilde ayarla
		userID := uuid.New()

		exists, err := userRepo.ExistsByNameOrMail(defaultUserName)
		if err != nil {
			return fmt.Errorf("error checking if user %s exists: %w", defaultUserName, err)
		}

		if exists {
			fmt.Printf("Info: Default system user '%s' already exists\n", defaultUserName)
			continue
		}

		password, err := helpers.GenerateRandomPassword(16)
		if err != nil {
			return fmt.Errorf("failed to generate random password for user %s: %w", defaultUserName, err)
		}

		userObj := &models.User{
			ID:          userID,
			PublicID:    node.Generate().Int64(),
			UserName:    defaultUserName,
			DisplayName: defaultUserName,
			Email:       defaultUserEmail,
			Password:    password,
			Domain:      models.CoolVibes,
			UserRole:    constants.UserRoleUser,
		}

		if err := userRepo.Create(userObj); err != nil {
			return fmt.Errorf("failed to create default system user %s: %w", defaultUserName, err)
		}

		fmt.Printf("Info: Default system user '%s' created\n", defaultUserName)
	}

	return nil
}
