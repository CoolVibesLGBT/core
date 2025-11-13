package lgbt

import (
	payloads "coolvibes/constants"
	helpers "coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/utils"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
)

func FetchInterests() ([]models.PreferenceCategory, error) {
	var attributes []models.PreferenceCategory

	// JSON dosyasını aç
	file, err := os.Open("static/data/interests.json")
	if err != nil {
		return nil, fmt.Errorf("Cannot open JSON file: %w", err)
	}
	defer file.Close()

	// JSON yapısını Go struct'larıyla eşliyoruz

	type Group struct {
		Gpb              string `json:"$gpb"`
		GroupID          int    `json:"group_id"`
		Name             string `json:"name"`
		ItemPreviewCount int    `json:"item_preview_count"`
	}

	type Interest struct {
		Gpb         string `json:"$gpb"`
		InterestID  int    `json:"interest_id"`
		Name        string `json:"name"`
		GroupID     int    `json:"group_id"`
		Emoji       string `json:"emoji"`
		HpElementID int    `json:"hp_element_id"`
	}

	type Data struct {
		Groups    []Group    `json:"groups"`
		Interests []Interest `json:"interests"`
	}

	var data Data
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("Cannot decode JSON: %w", err)
	}

	for _, group := range data.Groups {
		fmt.Printf("Group: %s\n", group.Name)

		category_title := group.Name
		category_slug := helpers.GenerateSlug(category_title)
		category_tag := payloads.UserAttributeInterests

		category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
		category_description := ""

		category := models.PreferenceCategory{
			ID:            category_id,
			Tag:           &category_tag,
			Slug:          &category_slug,
			Title:         utils.MakeLocalizedString("en", category_title),
			Description:   utils.MakeLocalizedString("en", category_description),
			Icon:          nil,
			AllowMultiple: true,
			Items:         []models.PreferenceItem{},
		}

		for interestId, item := range data.Interests {
			if item.GroupID == group.GroupID {
				title := item.Name
				icon := item.Emoji
				slug := helpers.GenerateSlug(title)
				item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
				slugPtr := &slug
				item := models.PreferenceItem{
					ID:           item_id,
					DisplayOrder: interestId,
					Slug:         slugPtr,
					Title:        utils.MakeLocalizedString("en", title),
					Description:  nil,
					Icon:         &icon,
					Visible:      true,
				}
				category.Items = append(category.Items, item)
			}
		}
		attributes = append(attributes, category)
	}
	return attributes, nil
}
