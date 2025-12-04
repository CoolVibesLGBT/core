package constants

type LanguageResponse struct {
	Code string `json:"code"`
	Flag string `json:"flag"`
	Name string `json:"name"`
}

// Languages
var Languages = map[string]LanguageResponse{
	"en": {Code: "en", Flag: "🇺🇸", Name: "English"},
	"tr": {Code: "tr", Flag: "🇹🇷", Name: "Türkçe"},
	"es": {Code: "es", Flag: "🇪🇸", Name: "Español"},
	"he": {Code: "he", Flag: "🇮🇱", Name: "עברית"},
	"ar": {Code: "ar", Flag: "🇸🇦", Name: "العربية"},

	// Mandarin / Chinese
	"zh": {Code: "zh", Flag: "🇨🇳", Name: "中文"},       // Simplified Chinese
	"tw": {Code: "tw", Flag: "🇹🇼", Name: "繁體中文"},     // Traditional Chinese (Taiwan)
	"hk": {Code: "hk", Flag: "🇭🇰", Name: "繁體中文（香港）"}, // Traditional Chinese (Hong Kong)

	"ja": {Code: "ja", Flag: "🇯🇵", Name: "日本語"},
	"hi": {Code: "hi", Flag: "🇮🇳", Name: "हिन्दी"},
	"de": {Code: "de", Flag: "🇩🇪", Name: "Deutsch"},
	"th": {Code: "th", Flag: "🇹🇭", Name: "ไทย"},

	"ru": {Code: "ru", Flag: "🇷🇺", Name: "Русский"},
	"pl": {Code: "pl", Flag: "🇵🇱", Name: "Polski"},
	"fr": {Code: "fr", Flag: "🇫🇷", Name: "Français"},
	"pt": {Code: "pt", Flag: "🇵🇹", Name: "Português"},
	"id": {Code: "id", Flag: "🇮🇩", Name: "Bahasa Indonesia"},
	"bn": {Code: "bn", Flag: "🇧🇩", Name: "বাংলা"},

	// Iran – Persian (Farsi)
	"fa": {Code: "fa", Flag: "🇮🇷", Name: "فارسی"},
}
