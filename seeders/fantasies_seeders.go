package seeders

import (
	"coolvibes/models/post/shared"
	payloads "coolvibes/models/user_payloads"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gorm.io/gorm"
)

func newCategoryLocalizedString(en, tr, es, he, ar, zh, ja, hi, de, th, ru, pl, fr, pt, id, bn string) shared.LocalizedString {
	return shared.LocalizedString{
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

var categories = map[string]shared.LocalizedString{
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

func SeedFantasies(db *gorm.DB) error {
	// JSON dosyasını aç
	file, err := os.Open("static/data/sexual_preferences.json")
	if err != nil {
		return fmt.Errorf("cannot open JSON file: %w", err)
	}
	defer file.Close()

	var data []struct {
		Label       map[string]string `json:"label"`
		Description map[string]string `json:"description"`
		Category    string            `json:"category"`
	}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return fmt.Errorf("cannot decode JSON: %w", err)
	}

	for index, item := range data {
		labelLocalized := shared.LocalizedString(item.Label)
		descriptionLocalized := shared.LocalizedString(item.Description)
		var existing payloads.Fantasy
		slug := item.Category // ya da uygun slug oluştur

		categoryLocalized := categories[item.Category]
		categoryPtr := &categoryLocalized

		// Label["en"] alanına göre sorgula
		if err := db.Where("label->>'en' = ?", item.Label["en"]).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// Kayıt yok, yeni oluştur
				fantasy := payloads.Fantasy{
					DisplayOrder: index,
					Slug:         slug,
					Category:     categoryPtr,
					Label:        labelLocalized,
					Description:  descriptionLocalized,
				}
				if err := db.Create(&fantasy).Error; err != nil {
					return fmt.Errorf("failed to insert fantasy: %w", err)
				}
			} else {
				return fmt.Errorf("db error: %w", err)
			}
		} else {
			// Kayıt var, güncelle
			existing.Slug = slug
			existing.Description = descriptionLocalized
			existing.Label = labelLocalized
			existing.Category = categoryPtr
			existing.DisplayOrder = index
			// Category da güncellenmek istenirse buraya ekle

			if err := db.Save(&existing).Error; err != nil {
				return fmt.Errorf("failed to update fantasy: %w", err)
			}
		}

	}

	return nil
}
