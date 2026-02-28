package test

import (
	"bytes"
	"context"
	"core/helpers"
	"core/repositories"
	services "core/services/user"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func detectMimeType(path string, t *testing.T) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := f.Close(); err != nil {
			t.Fatalf("file close error: %v", err)
		}
	}()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	return http.DetectContentType(buf[:n]), nil
}

func testFileHeader(path string, t *testing.T) (*multipart.FileHeader, error) {
	// MIME tespit et
	mime, err := detectMimeType(path, t)
	if err != nil {
		return nil, err
	}

	// multipart body oluştur
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// form file oluştur
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	// Test request oluştur
	req := httptest.NewRequest("POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// FileHeader'ı al
	_, header, err := req.FormFile("file")
	if err != nil {
		return nil, err
	}

	// MIME TYPE EKLE (en önemli nokta)
	header.Header = textproto.MIMEHeader{}
	header.Header.Set("Content-Type", mime)

	return header, nil
}

func testImage(db *gorm.DB, snowFlakeNode *helpers.Node, t *testing.T) {

	notificationRepo := repositories.NewNotificationRepository(db, snowFlakeNode)
	// repository ve service oluştur
	engagementRepo := repositories.NewEngagementRepository(db)
	userRepo := repositories.NewUserRepository(db, nil, snowFlakeNode, engagementRepo, notificationRepo)
	mediaRepo := repositories.NewMediaRepository(db, snowFlakeNode)
	postRepo := repositories.NewPostRepository(db, snowFlakeNode, mediaRepo, userRepo, notificationRepo)
	userService := services.NewUserService(userRepo, postRepo, mediaRepo, engagementRepo, notificationRepo)

	user, err := userService.GetUserByID(uuid.MustParse("e6717509-ad80-433b-bf7e-8f1b4d338c8e"))
	if err != nil {
		fmt.Println("TEST:USER_NOT_FOUND")
	}

	fmt.Println("USER", user.UserName)

	currentDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	testAvatarFile := currentDirectory + "/static/samples/test2.jpeg"
	fmt.Println("FULL_FILE_PATH", testAvatarFile)
	testAvatarFileHader, err := testFileHeader(testAvatarFile, t)
	if err != nil {
		fmt.Println("TEST:AVATAR_FILE_ERR", err)
		return
	}

	avatarMedia, avatarErr := userService.UpdateAvatar(context.Background(), testAvatarFileHader, user)
	if avatarErr != nil {
		fmt.Println("TEST:AVATAR_FILE_ERR", avatarErr)
		return
	}
	fmt.Printf("AVATAR_SAMPLE\n%+v\n", avatarMedia)

	coverMedia, coverErr := userService.UpdateCover(context.Background(), testAvatarFileHader, user)
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
