// Package groups seeds one public "group" chat per country/territory, using
// the pre-rendered avatar/cover/hero AVIF images shipped under
// static/groups/images. Country names and descriptions are localized across
// every language the product supports (constants.Languages).
package groups

import (
	"core/constants"
	"core/helpers"
	"core/models"
	chatmodel "core/models/chat"
	"core/models/media"
	modelutils "core/models/utils"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

//go:embed countries_data.json
var countriesDataJSON []byte

type countryGroupData struct {
	Code           string            `json:"code"`
	PublicUsername string            `json:"public_username"`
	Titles         map[string]string `json:"titles"`
	Descriptions   map[string]string `json:"descriptions"`
	Avatar         string            `json:"avatar"`
	Cover          string            `json:"cover"`
	Hero           string            `json:"hero"`
}

// imageSpec describes the fixed dimensions of each pre-rendered asset kind,
// per static/groups/README.txt.
type imageSpec struct {
	role          media.MediaRole
	width, height int
	// isPublic must be false for roles the DB's chk_medias_protected_visibility
	// constraint treats as protected (private_photo, chat_image, chat_media,
	// chat_video) — the avatar keeps the same role real group-avatar uploads
	// use, so it inherits that rule even though it's served from an ungated
	// static path.
	isPublic bool
}

var imageSpecs = map[string]imageSpec{
	"avatar": {role: media.RoleChatImage, width: 512, height: 512, isPublic: true},
	"cover":  {role: media.RoleChatCover, width: 1500, height: 500, isPublic: true},
	"hero":   {role: media.RoleChatHero, width: 1920, height: 1280, isPublic: true},
}

const groupsStaticDir = "static/groups"

func SeedGroups(db *gorm.DB, node *helpers.Node) error {
	if node == nil {
		return errors.New("snowflake node is required")
	}

	var countries []countryGroupData
	if err := json.Unmarshal(countriesDataJSON, &countries); err != nil {
		return fmt.Errorf("parse countries_data.json: %w", err)
	}

	var admin models.User
	adminEmail := fmt.Sprintf("%s@coolvibes.lgbt", constants.SystemUserAdmin)
	if err := db.Unscoped().
		Where("LOWER(user_name) = ? AND LOWER(email) = ?", strings.ToLower(constants.SystemUserAdmin), strings.ToLower(adminEmail)).
		First(&admin).Error; err != nil {
		return fmt.Errorf("find system admin user for group seeding: %w", err)
	}

	created, skipped := 0, 0
	for i := range countries {
		entry := &countries[i]

		var existing chatmodel.Chat
		err := db.Unscoped().Where("public_username = ?", entry.PublicUsername).First(&existing).Error
		if err == nil {
			skipped++
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("check existing group %s: %w", entry.PublicUsername, err)
		}

		if err := createCountryGroup(db, node, admin.ID, entry); err != nil {
			return fmt.Errorf("create group %s: %w", entry.PublicUsername, err)
		}
		created++
	}

	fmt.Printf("groups seeder: created %d country groups, skipped %d already seeded\n", created, skipped)
	return nil
}

func createCountryGroup(db *gorm.DB, node *helpers.Node, creatorID uuid.UUID, entry *countryGroupData) error {
	return db.Transaction(func(tx *gorm.DB) error {
		chatID := uuid.New()
		now := time.Now().UTC()

		avatarMedia, err := buildGroupMedia(node, chatID, creatorID, "avatar", entry.Avatar)
		if err != nil {
			return err
		}
		coverMedia, err := buildGroupMedia(node, chatID, creatorID, "cover", entry.Cover)
		if err != nil {
			return err
		}
		heroMedia, err := buildGroupMedia(node, chatID, creatorID, "hero", entry.Hero)
		if err != nil {
			return err
		}
		for _, m := range []*media.Media{avatarMedia, coverMedia, heroMedia} {
			if err := tx.Create(&m.File).Error; err != nil {
				return fmt.Errorf("create media file: %w", err)
			}
			if err := tx.Create(m).Error; err != nil {
				return fmt.Errorf("create media: %w", err)
			}
		}

		title := modelutils.LocalizedString(cloneStringMap(entry.Titles))
		description := modelutils.LocalizedString(cloneStringMap(entry.Descriptions))
		username := entry.PublicUsername

		groupChat := chatmodel.Chat{
			ID:             chatID,
			Type:           chatmodel.ChatTypeGroup,
			Visibility:     "public",
			PublicUsername: &username,
			Title:          &title,
			Description:    &description,
			AvatarID:       &avatarMedia.ID,
			CoverID:        &coverMedia.ID,
			HeroID:         &heroMedia.ID,
			CreatorID:      creatorID,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := tx.Create(&groupChat).Error; err != nil {
			return fmt.Errorf("create chat: %w", err)
		}

		participant := chatmodel.ChatParticipant{
			ID:       uuid.New(),
			ChatID:   chatID,
			UserID:   creatorID,
			Role:     chatmodel.ParticipantRoleOwner,
			JoinedAt: now,
		}
		if err := tx.Create(&participant).Error; err != nil {
			return fmt.Errorf("create owner participant: %w", err)
		}

		return nil
	})
}

func buildGroupMedia(node *helpers.Node, chatID, userID uuid.UUID, kind, relPath string) (*media.Media, error) {
	spec, ok := imageSpecs[kind]
	if !ok {
		return nil, fmt.Errorf("unknown image kind %q", kind)
	}

	storagePath := groupsStaticDir + "/" + relPath
	info, err := os.Stat(storagePath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", storagePath, err)
	}

	width, height := spec.width, spec.height
	fileName := relPath[strings.LastIndex(relPath, "/")+1:]

	fileMeta := modelutils.FileMetadata{
		ID:          uuid.New(),
		URL:         "/" + storagePath,
		StoragePath: storagePath,
		MimeType:    "image/avif",
		Size:        info.Size(),
		Name:        fileName,
		Width:       &width,
		Height:      &height,
		CreatedAt:   time.Now().UTC(),
	}

	return &media.Media{
		ID:               uuid.New(),
		PublicID:         node.Generate().Int64(),
		FileID:           fileMeta.ID,
		OwnerID:          chatID,
		OwnerType:        media.OwnerChat,
		UserID:           userID,
		Role:             spec.role,
		IsPublic:         spec.isPublic,
		ProcessingStatus: media.ProcessingStatusReady,
		File:             fileMeta,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}, nil
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
