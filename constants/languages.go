package constants

type LanguageResponse struct {
	Code string `json:"code"`
	Flag string `json:"flag"`
	Name string `json:"name"`
}

// Languages
var Languages = map[string]LanguageResponse{
	"en": {Code: "en", Flag: "/images/countries/1x1/us.svg", Name: "English"},
	"tr": {Code: "tr", Flag: "/images/countries/1x1/tr.svg", Name: "Türkçe"},
	"es": {Code: "es", Flag: "/images/countries/1x1/es.svg", Name: "Español"},
	"he": {Code: "he", Flag: "/images/countries/1x1/il.svg", Name: "עברית"},
	"ar": {Code: "ar", Flag: "/images/countries/1x1/sa.svg", Name: "العربية"},

	// Mandarin / Chinese
	"zh": {Code: "zh", Flag: "/images/countries/1x1/cn.svg", Name: "中文"},       // Simplified Chinese
	"tw": {Code: "tw", Flag: "/images/countries/1x1/tw.svg", Name: "繁體中文"},     // Traditional Chinese (Taiwan)
	"hk": {Code: "hk", Flag: "/images/countries/1x1/hk.svg", Name: "繁體中文（香港）"}, // Traditional Chinese (Hong Kong)

	"ja": {Code: "ja", Flag: "/images/countries/1x1/jp.svg", Name: "日本語"},
	"hi": {Code: "hi", Flag: "/images/countries/1x1/in.svg", Name: "हिन्दी"},
	"de": {Code: "de", Flag: "/images/countries/1x1/de.svg", Name: "Deutsch"},
	"th": {Code: "th", Flag: "/images/countries/1x1/th.svg", Name: "ไทย"},

	"ru": {Code: "ru", Flag: "/images/countries/1x1/ru.svg", Name: "Русский"},
	"pl": {Code: "pl", Flag: "/images/countries/1x1/pl.svg", Name: "Polski"},
	"fr": {Code: "fr", Flag: "/images/countries/1x1/fr.svg", Name: "Français"},
	"pt": {Code: "pt", Flag: "/images/countries/1x1/pt.svg", Name: "Português"},
	"id": {Code: "id", Flag: "/images/countries/1x1/id.svg", Name: "Bahasa Indonesia"},
	"bn": {Code: "bn", Flag: "/images/countries/1x1/bd.svg", Name: "বাংলা"},

	// Iran – Persian (Farsi)
	"fa": {Code: "fa", Flag: "/images/countries/1x1/ir.svg", Name: "فارسی"},

	"kr": {Code: "kr", Flag: "/images/countries/1x1/kr.svg", Name: "한국어 (South Korea)"},
	"kp": {Code: "kp", Flag: "/images/countries/1x1/kp.svg", Name: "조선말 (North Korea)"},
}
