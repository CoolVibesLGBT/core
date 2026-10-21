package test

import (
	"context"
	upload "core/adapters/media"
	usecases "core/application/usecases"
	"core/helpers"
	"core/infrastructure/repositories"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func testImage(db *gorm.DB, snowFlakeNode *helpers.Node, t *testing.T) {

	notificationRepo := repositories.NewNotificationRepository(db, snowFlakeNode)
	// repository ve service oluştur
	engagementRepo := repositories.NewEngagementRepository(db)
	userRepo := repositories.NewUserRepository(db, nil, snowFlakeNode, engagementRepo, notificationRepo)
	mediaRepo := repositories.NewMediaRepository(db, snowFlakeNode)
	postRepo := repositories.NewPostRepository(db, snowFlakeNode, mediaRepo, userRepo, notificationRepo)
	userService := usecases.NewUserService(userRepo, postRepo, mediaRepo, engagementRepo, notificationRepo)

	user, err := userService.GetUserByID(uuid.MustParse("e6717509-ad80-433b-bf7e-8f1b4d338c8e"))
	if err != nil {
		fmt.Println("TEST:USER_NOT_FOUND")
	}

	fmt.Println("USER", user.UserName)

	currentDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	testAvatarFile := currentDirectory + "/static/samples/test2.jpeg"
	fmt.Println("FULL_FILE_PATH", testAvatarFile)
	testAvatarUpload, err := upload.FromPath(testAvatarFile)
	if err != nil {
		fmt.Println("TEST:AVATAR_FILE_ERR", err)
		return
	}

	avatarMedia, avatarErr := userService.UpdateAvatar(context.Background(), testAvatarUpload, user)
	if avatarErr != nil {
		fmt.Println("TEST:AVATAR_FILE_ERR", avatarErr)
		return
	}
	fmt.Printf("AVATAR_SAMPLE\n%+v\n", avatarMedia)

	coverMedia, coverErr := userService.UpdateCover(context.Background(), testAvatarUpload, user)
	if coverErr != nil {
		fmt.Println("TEST:AVATAR_FILE_ERR", avatarErr)
		return
	}
	fmt.Printf("COVER_SAMPLE\n%+v\n", coverMedia)
	fmt.Println("Test Excuted")
}

func StartImageTest(db *gorm.DB, snowFlakeNode *helpers.Node, t *testing.T) {
	testImage(db, snowFlakeNode, t)
}
