package lgbt

import (
	payloads "core/constants"
	helpers "core/helpers"
	"core/models"
	"core/models/utils"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

func FetchInterests() ([]models.PreferenceCategory, error) {
	var attributes []models.PreferenceCategory

	// JSON dosyasını aç — interests_i18n.json, interests.json'un yerini alır:
	// her kategori/öge artık tek bir "en" yerine core.Languages'teki her dil
	// için bir çeviri taşıyor (bkz. docs/interface-audit.md, web reposu).
	file, err := os.Open("static/data/interests_i18n.json")
	if err != nil {
		return nil, fmt.Errorf("cannot open JSON file: %w", err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// JSON yapısını Go struct'larıyla eşliyoruz

	type Group struct {
		GroupID          int               `json:"group_id"`
		ItemPreviewCount int               `json:"item_preview_count"`
		Titles           map[string]string `json:"titles"`
	}

	type Interest struct {
		InterestID int               `json:"interest_id"`
		GroupID    int               `json:"group_id"`
		Emoji      string            `json:"emoji"`
		Titles     map[string]string `json:"titles"`
	}

	type Data struct {
		Groups    []Group    `json:"groups"`
		Interests []Interest `json:"interests"`
	}

	var data Data
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("cannot decode JSON: %w", err)
	}

	for _, group := range data.Groups {
		// Slug'lar (ve onlardan türeyen deterministik ID'ler) kasıtlı olarak
		// hep İngilizce başlıktan üretiliyor, böylece bu çeviri geçişi mevcut
		// kullanıcı tercih seçimlerinin referans verdiği UUID'leri değiştirmiyor.
		category_title := group.Titles["en"]
		category_slug := helpers.GenerateSlug(category_title)
		category_tag := payloads.UserAttributeInterests

		category_id := uuid.NewSHA1(payloads.NameSpace, []byte(category_slug))
		category_description := ""
		category_titles := utils.LocalizedString(group.Titles)

		category := models.PreferenceCategory{
			ID:            category_id,
			Tag:           &category_tag,
			Slug:          &category_slug,
			Title:         &category_titles,
			Description:   utils.MakeLocalizedString("en", category_description),
			Icon:          nil,
			AllowMultiple: true,
			Items:         []models.PreferenceItem{},
		}

		for interestId, item := range data.Interests {
			if item.GroupID == group.GroupID {
				title := item.Titles["en"]
				icon := item.Emoji
				slug := helpers.GenerateSlug(title)
				item_id := uuid.NewSHA1(payloads.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
				slugPtr := &slug
				item_titles := utils.LocalizedString(item.Titles)
				prefItem := models.PreferenceItem{
					ID:           item_id,
					DisplayOrder: interestId,
					Slug:         slugPtr,
					Title:        &item_titles,
					Description:  nil,
					Icon:         &icon,
					Visible:      true,
				}
				category.Items = append(category.Items, prefItem)
			}
		}
		attributes = append(attributes, category)
	}
	return attributes, nil
}
