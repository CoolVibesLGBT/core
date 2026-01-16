package default_users

import (
	"coolvibes/application"
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/repositories"
	"fmt"

	"github.com/google/uuid"
)

func SeedDefaultSystemUsers(application *application.App) error {
	engagementRepo := repositories.NewEngagementRepository(application.DB)
	userRepo := repositories.NewUserRepository(application.DB, application.SnowFlakeNode, engagementRepo)

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
			PublicID:    application.SnowFlakeNode.Generate().Int64(),
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
