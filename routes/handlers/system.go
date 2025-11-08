package handlers

import (
	payloads "coolvibes/models/user_payloads"
	"encoding/json"
	"log"
	"net/http"

	"gorm.io/gorm"
)

// TranslationMap map[string]Translation

// CountryResponse tek ülke objesi
type CountryResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type LanguageResponse struct {
	Code string `json:"code"`
	Flag string `json:"flag"`
	Name string `json:"name"`
}

type OrientationData struct {
	ID           string            `json:"id"`
	Key          string            `json:"key"`
	Translations map[string]string `json:"translations"`
}

type GroupedAttributes struct {
	Category   string          `json:"category"`
	Attributes json.RawMessage `json:"attributes"` // JSON array olarak döner
}

// InitialData dönecek ana struct
type InitialData struct {
	Fantasies  []payloads.Fantasy         `json:"fantasies"`
	Countries  map[string]CountryResponse `json:"countries"`
	Interests  []payloads.Interest        `json:"interests"`
	Attributes []GroupedAttributes        `json:"attributes"` // key -> {lang -> label}

	Languages map[string]LanguageResponse `json:"languages"`

	GenderIdentities   []payloads.GenderIdentity    `json:"gender_identities"`
	SexualOrientations []payloads.SexualOrientation `json:"sexual_orientations"`
	SexRoles           []payloads.SexualRole        `json:"sexual_roles"`
	Status             string                       `json:"status"`
}

func HandleInitialSync(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Tüm fantezileri çek
		var fantasies []payloads.Fantasy
		if err := db.Order("display_order DESC").Find(&fantasies).Error; err != nil {
			http.Error(w, "Failed to fetch fantasies", http.StatusInternalServerError)
			return
		}

		var interests []payloads.Interest
		if err := db.Preload("Items").Find(&interests).Error; err != nil {
			http.Error(w, "Failed to fetch interests", http.StatusInternalServerError)
			return
		}

		var genderIdentities []payloads.GenderIdentity
		var sexualOrientations []payloads.SexualOrientation
		var sexRoles []payloads.SexualRole

		// Gender Identities
		if err := db.Find(&genderIdentities).Error; err != nil {
			http.Error(w, "Failed to fetch gender identities", http.StatusInternalServerError)
			return
		}

		// Sexual Orientations
		if err := db.Find(&sexualOrientations).Error; err != nil {
			http.Error(w, "Failed to fetch Sexual Orientations", http.StatusInternalServerError)
			return
		}

		// Sex Roles
		if err := db.Find(&sexRoles).Error; err != nil {
			http.Error(w, "Failed to fetch sex roles", http.StatusInternalServerError)
			return
		}

		var attributes []GroupedAttributes

		err := db.Model(&payloads.Attribute{}).
			Select(`
			category,
			json_agg(
				jsonb_build_object(
					'id', id,
					'display_order', display_order,
					'name', name
				) ORDER BY display_order
			) AS attributes
		`).
			Group("category").
			Order("category ASC").
			Scan(&attributes).Error

		if err != nil {
			log.Fatalf("query error: %v", err)
		}

		// 3. Ülkeleri çek
		// Örneğin countries tablosu veya sabit listeden
		countries := map[string]CountryResponse{
			"TR": {Code: "TR", Name: "Turkey"},
			"US": {Code: "US", Name: "United States"},
			// dilediğin kadar ekle
		}

		// Languages
		languages := map[string]LanguageResponse{
			"en": {Code: "en", Flag: "🇺🇸", Name: "English"},
			"tr": {Code: "tr", Flag: "🇹🇷", Name: "Türkçe"},
			"es": {Code: "es", Flag: "🇪🇸", Name: "Español"},
			"he": {Code: "he", Flag: "🇮🇱", Name: "עברית"},
			"ar": {Code: "ar", Flag: "🇸🇦", Name: "العربية"},
			"zh": {Code: "zh", Flag: "🇨🇳", Name: "中文"},
			"ja": {Code: "ja", Flag: "🇯🇵", Name: "日本語"},
			"hi": {Code: "hi", Flag: "🇮🇳", Name: "हिन्दी"},
			"de": {Code: "de", Flag: "🇩🇪", Name: "Deutsch"},
			"th": {Code: "th", Flag: "🇹🇭", Name: "ไทย"},
			"ru": {Code: "ru", Flag: "🇷🇺", Name: "Русский"},          // Rusça
			"pl": {Code: "pl", Flag: "🇵🇱", Name: "Polski"},           // Lehçe
			"fr": {Code: "fr", Flag: "🇫🇷", Name: "Français"},         // Fransızca
			"pt": {Code: "pt", Flag: "🇵🇹", Name: "Português"},        // Portekizce
			"id": {Code: "id", Flag: "🇮🇩", Name: "Bahasa Indonesia"}, // Endonezce
			"bn": {Code: "bn", Flag: "🇧🇩", Name: "বাংলা"},            // Bengalce
		}

		// 5. InitialData hazırla
		initialData := InitialData{
			Fantasies:          fantasies,
			Countries:          countries,
			Interests:          interests,
			Attributes:         attributes,
			Languages:          languages,
			GenderIdentities:   genderIdentities,
			SexualOrientations: sexualOrientations,
			SexRoles:           sexRoles,
			Status:             "ok",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(initialData)
	}
}
