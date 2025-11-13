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

func newCategoryLocalizedString(en, tr, es, he, ar, zh, ja, hi, de, th, ru, pl, fr, pt, id, bn string) utils.LocalizedString {
	return utils.LocalizedString{
		"en": en,
		"tr": tr,
		"es": es,
		"he": he,
		"ar": ar,
		"zh": zh,
		"ja": ja,
		"hi": hi,
		"de": de,
		"th": th,
		"ru": ru,
		"pl": pl,
		"fr": fr,
		"pt": pt,
		"id": id,
		"bn": bn,
	}
}

var categories = map[string]utils.LocalizedString{
	"joy_or_tabu": newCategoryLocalizedString(
		"Joy or Taboo", "Zevk veya Tabu", "Alegría o Tabú", "שמחה או טאבו", "فرح أو تابو", "欢乐或禁忌", "喜びまたは禁忌", "आनंद या वर्जना", "Freude oder Tabu", "ความสุขหรือตาบู", "Радость или табу", "Radość lub tabu", "Joie ou tabou", "Alegria ou tabu", "Kegembiraan atau tabu", "আনন্দ বা ট্যাবু",
	),
	"sexual_adventure": newCategoryLocalizedString(
		"Sexual Adventure", "Cinsel Macera", "Aventura Sexual", "הרפתקה מינית", "مغامرة جنسية", "性冒险", "性的冒険", "यौन साहसिक", "Sexuelles Abenteuer", "การผจญภัยทางเพศ", "Сексуальное приключение", "Przygoda seksualna", "Aventure sexuelle", "Aventura sexual", "Petualangan Seksual", "যৌন অ্যাডভেঞ্চার",
	),
	"physical_pref": newCategoryLocalizedString(
		"Physical Preference", "Fiziksel Tercih", "Preferencia Física", "העדפה פיזית", "تفضيل جسدي", "身体偏好", "身体的な好み", "शारीरिक प्राथमिकता", "Physische Präferenz", "ความชอบทางร่างกาย", "Физические предпочтения", "Preferencja fizyczna", "Préférence physique", "Preferência física", "Preferensi Fisik", "শারীরিক পছন্দ",
	),
	"sexual_pref": newCategoryLocalizedString(
		"Sexual Preference", "Cinsel Tercih", "Preferencia Sexual", "העדפה מינית", "تفضيل جنسي", "性偏好", "性的嗜好", "यौन प्राथमिकता", "Sexuelle Präferenz", "ความชอบทางเพศ", "Сексуальные предпочтения", "Preferencja seksualna", "Préférence sexuelle", "Preferência sexual", "Preferensi Seksual", "যৌন পছন্দ",
	),
	"amusement": newCategoryLocalizedString(
		"Amusement", "Eğlence", "Diversión", "בידור", "تسلية", "娱乐", "娯楽", "मनोरंजन", "Vergnügen", "ความบันเทิง", "Развлечение", "Rozrywka", "Amusement", "Diversão", "Hiburan", "বিনোদন",
	),
}

func FetchFantasies() ([]models.PreferenceCategory, error) {

	var fantasies []models.PreferenceCategory

	// JSON dosyasını aç
	file, err := os.Open("static/data/sexual_preferences.json")
	if err != nil {
		return nil, fmt.Errorf("cannot open JSON file: %w", err)
	}
	defer file.Close()

	var data []struct {
		Label       map[string]string `json:"label"`
		Description map[string]string `json:"description"`
		Category    string            `json:"category"`
	}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, fmt.Errorf("cannot decode JSON: %w", err)
	}

	var displayOrder = 0
	for categorySlug, categoryItem := range categories {
		category_title := categoryItem
		category_slug := helpers.GenerateSlug(category_title["en"])
		category_tag := payloads.UserAttributeFantasies

		category_id := uuid.NewSHA1(helpers.NameSpace, []byte(category_slug))
		category_description := "Gender identity refers to a person’s deeply held sense of their own gender how they personally experience themselves as male, female, both, neither, or somewhere along the gender spectrum. It may or may not align with the sex they were assigned at birth, and it is an internal, individual understanding of who they are, rather than how others perceive them."

		category := models.PreferenceCategory{
			ID:            category_id,
			DisplayOrder:  displayOrder,
			Tag:           &category_tag,
			Slug:          &category_slug,
			Title:         &category_title,
			Description:   utils.MakeLocalizedString("en", category_description),
			Icon:          nil,
			AllowMultiple: true,
			Items:         []models.PreferenceItem{},
		}
		displayOrder += 1

		for index, item := range data {
			labelLocalized := utils.LocalizedString(item.Label)
			descriptionLocalized := utils.LocalizedString(item.Description)

			if item.Category == categorySlug {

				slug := helpers.GenerateSlug(labelLocalized["en"])
				item_id := uuid.NewSHA1(helpers.NameSpace, []byte(fmt.Sprintf("%s-%s", category_slug, slug)))
				item := models.PreferenceItem{
					ID:           item_id,
					DisplayOrder: index,
					Slug:         &slug,
					Title:        &labelLocalized,
					Description:  &descriptionLocalized,
					Icon:         nil,
					Visible:      true,
				}
				category.Items = append(category.Items, item)
			}

		}
		fantasies = append(fantasies, category)

	}

	return fantasies, nil
}
