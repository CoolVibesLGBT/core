package news

import (
	"fmt"
	"net/url"
	"strings"
)

type RSSSource struct {
	ID      string
	Name    string
	URL     string
	Enabled bool
}

type Language struct {
	Code           string   // ex: "tr"
	CountryCode    string   // ex: "TR"
	SearchKeywords []string // LGBT ile ilgili kelimeler listesi
	HL             string
	GL             string
	CEID           string
}

var Languages = []Language{
	{
		Code: "tr", CountryCode: "TR",
		SearchKeywords: []string{"gay", "homoseksüel", "lezbiyen", "biseksüel", "transseksüel", "travesti", "queer", "interseks", "nonbinary", "panseksüel", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "tr", GL: "TR", CEID: "TR:tr",
	},
	{
		Code: "en", CountryCode: "US",
		SearchKeywords: []string{"gay", "homosexual", "lesbian", "bisexual", "transgender", "queer", "intersex", "nonbinary", "pansexual", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "en-US", GL: "US", CEID: "US:en",
	},
	{
		Code: "es", CountryCode: "ES",
		SearchKeywords: []string{"gay", "homosexual", "lesbiana", "bisexual", "transgénero", "travesti", "queer", "intersexual", "no binario", "pansexual", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "es", GL: "ES", CEID: "ES:es",
	},
	{
		Code: "fr", CountryCode: "FR",
		SearchKeywords: []string{"gay", "homosexuel", "lesbienne", "bisexuel", "transgenre", "travesti", "queer", "intersexe", "non binaire", "pansexuel", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "fr", GL: "FR", CEID: "FR:fr",
	},
	{
		Code: "de", CountryCode: "DE",
		SearchKeywords: []string{"schwul", "homosexuell", "lesbisch", "bisexuell", "transgender", "travesti", "queer", "intersexuell", "nichtbinär", "pansexuell", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "de", GL: "DE", CEID: "DE:de",
	},
	{
		Code: "ru", CountryCode: "RU",
		SearchKeywords: []string{"гей", "гомосексуалист", "лесбиянка", "бисексуал", "трансгендер", "квир", "интерсекс", "небинарный", "пансексуал", "ЛГБТ", "ЛГБТК", "ЛГБТКИА"},
		HL:             "ru", GL: "RU", CEID: "RU:ru",
	},
	{
		Code: "pt", CountryCode: "PT",
		SearchKeywords: []string{"gay", "homossexual", "lésbica", "bissexual", "transgênero", "travesti", "queer", "intersexo", "não binário", "pansexual", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "pt", GL: "PT", CEID: "PT:pt",
	},
	{
		Code: "it", CountryCode: "IT",
		SearchKeywords: []string{"gay", "omosessuale", "lesbica", "bisessuale", "transgender", "travestito", "queer", "intersessuale", "non binario", "pansessuale", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "it", GL: "IT", CEID: "IT:it",
	},
	{
		Code: "ja", CountryCode: "JP",
		SearchKeywords: []string{"ゲイ", "レズビアン", "バイセクシャル", "トランスジェンダー", "クィア", "インターセックス", "ノンバイナリー", "パンセクシュアル", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "ja", GL: "JP", CEID: "JP:ja",
	},
	{
		Code: "ko", CountryCode: "KR",
		SearchKeywords: []string{"게이", "레즈비언", "양성애자", "트랜스젠더", "퀴어", "인터섹스", "논바이너리", "범성애자", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "ko", GL: "KR", CEID: "KR:ko",
	},
	{
		Code: "zh", CountryCode: "CN",
		SearchKeywords: []string{"同性恋", "双性恋", "跨性别", "酷儿", "间性人", "非二元", "泛性恋", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "zh-CN", GL: "CN", CEID: "CN:zh-Hans",
	},
	{
		Code: "ar", CountryCode: "SA",
		SearchKeywords: []string{"مثلي", "مثلية", "مزدوج", "متحول", "كوير", "بين الجنسين", "بانسكسوال", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "ar", GL: "SA", CEID: "SA:ar",
	},
	{
		Code: "he", CountryCode: "IL",
		SearchKeywords: []string{"הומו", "לסבית", "ביסקסואל", "טרנסג'נדר", "קוויר", "אינטרסקס", "נון-בינארי", "פנסקסואל", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "he", GL: "IL", CEID: "IL:he",
	},
	{
		Code: "hi", CountryCode: "IN",
		SearchKeywords: []string{"गे", "समलैंगिक", "लेस्बियन", "बाइसेक्शुअल", "ट्रांसजेंडर", "क्वीर", "इंटरसेक्स", "नॉनबाइनरी", "पैनसेक्सुअल", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "hi", GL: "IN", CEID: "IN:hi",
	},
	{
		Code: "id", CountryCode: "ID",
		SearchKeywords: []string{"gay", "lesbian", "biseksual", "transgender", "queer", "interseks", "nonbiner", "panseksual", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "id", GL: "ID", CEID: "ID:id",
	},
	{
		Code: "nl", CountryCode: "NL",
		SearchKeywords: []string{"homo", "lesbisch", "biseksueel", "transgender", "queer", "intersekse", "non-binair", "panseksueel", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "nl", GL: "NL", CEID: "NL:nl",
	},
	{
		Code: "pl", CountryCode: "PL",
		SearchKeywords: []string{"gej", "homoseksualista", "lesbijka", "biseksualista", "transpłciowy", "queer", "interseksualny", "niebinarny", "panseksualny", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "pl", GL: "PL", CEID: "PL:pl",
	},
	{
		Code: "sv", CountryCode: "SE",
		SearchKeywords: []string{"gay", "homosexuell", "lesbisk", "bisexuell", "transgender", "queer", "intersexuell", "ickebinär", "pansexuell", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "sv", GL: "SE", CEID: "SE:sv",
	},
	{
		Code: "ro", CountryCode: "RO",
		SearchKeywords: []string{"gay", "homosexual", "lesbiană", "bisexual", "transgender", "queer", "intersexual", "non-binar", "pansexual", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "ro", GL: "RO", CEID: "RO:ro",
	},
	{
		Code: "vi", CountryCode: "VN",
		SearchKeywords: []string{"đồng tính", "đồng tính nữ", "lưỡng tính", "chuyển giới", "queer", "liên giới tính", "phi nhị phân", "đa giới tính", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "vi", GL: "VN", CEID: "VN:vi",
	},
	{
		Code: "ne", CountryCode: "NP",
		SearchKeywords: []string{"gay", "लेस्बियन", "बाइसेक्सुअल", "ट्रान्सजेन्डर", "क्वियर", "इन्टरसेक्स", "ननबाइनरी", "प्यानसेक्सुअल", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "ne", GL: "NP", CEID: "NP:ne",
	},
	{
		Code: "tw", CountryCode: "TW",
		SearchKeywords: []string{"同志", "同性戀", "跨性別", "酷兒", "變性", "性少數", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "zh-TW", GL: "TW", CEID: "TW:zh-Hant",
	},
	{
		Code: "th", CountryCode: "TH",
		SearchKeywords: []string{"เกย์", "เลสเบี้ยน", "ไบเซ็กชวล", "ทรานส์เจนเดอร์", "ควียร์", "อินเตอร์เซ็กซ์", "โนบินารี", "แพนเซ็กชวล", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "th", GL: "TH", CEID: "TH:th",
	},
	{
		Code: "tl", CountryCode: "PH",
		SearchKeywords: []string{"bakla", "lesbiyana", "biysekswal", "transgender", "queer", "intersex", "nonbinary", "pansexual", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "tl", GL: "PH", CEID: "PH:tl",
	},
	{
		Code: "km", CountryCode: "KH",
		SearchKeywords: []string{"កាម៉េរ៉ូ", "ស្រឡាញ់ផ្លូវភេទដូចគ្នា", "បីភេទ", "បំលែងភេទ", "គួ", "អន្តរភេទ", "អតិបរិមា", "LGBT", "LGBTQ", "LGBTQIA"},
		HL:             "km", GL: "KH", CEID: "KH:km",
	},
}

var GoogleNewsSource = RSSSource{
	ID:      "gnews",
	Name:    "Google News (Dynamic)",
	URL:     "https://news.google.com/rss/search?q=(%s) when:7d&hl=%s&gl=%s&ceid=%s",
	Enabled: true,
}

var StaticRSSSources = []RSSSource{
	{ID: "pinknews", Name: "PinkNews", URL: "https://www.pinknews.co.uk/feed/", Enabled: true},
	{ID: "scenemag", Name: "SceneMag", URL: "https://www.scenemag.co.uk/rss/", Enabled: true},
	{ID: "hrw", Name: "hrw", URL: "https://www.hrw.org/rss/news", Enabled: true},

	{ID: "spod", Name: "SpoD", URL: "https://spod.org.tr/feed/", Enabled: true},
	{ID: "lgbtqnation", Name: "LGBTQ Nation", URL: "https://www.lgbtqnation.com/feed/", Enabled: true},
	{ID: "guardian-lgbt", Name: "The Guardian - LGBT rights", URL: "https://www.theguardian.com/world/lgbt-rights/rss", Enabled: true},
	{ID: "out-mag", Name: "OUT (尝试)", URL: "https://www.out.com/rss.xml", Enabled: true},
}

func GenerateGoogleNewsRSSURL(lang Language) string {
	baseURL := "https://news.google.com/rss/search"
	query := "(" + strings.Join(lang.SearchKeywords, " OR ") + ") when:7d"
	encodedQuery := url.QueryEscape(query)

	return fmt.Sprintf("%s?q=%s&hl=%s&gl=%s&ceid=%s", baseURL, encodedQuery, lang.HL, lang.GL, lang.CEID)
}

func GenerateBingNewsRSSURL(lang Language) string {
	baseURL := "https://www.bing.com/news/search"

	// Her arama kelimesini feed:"kelime" olarak yap, sonra OR ile bağla
	quotedTerms := make([]string, len(lang.SearchKeywords))
	for i, w := range lang.SearchKeywords {
		quotedTerms[i] = fmt.Sprintf("feed:%q", w) // örn: feed:"gay"
	}

	query := strings.Join(quotedTerms, " OR ")
	encoded := url.QueryEscape(query)

	return fmt.Sprintf("%s?q=%s&format=rss&setlang=%s&mkt=%s", baseURL, encoded, lang.HL, lang.HL)

}

func GenerateAllRSSSources() []RSSSource {
	var rssSources []RSSSource

	// Önce statik kaynakları ekle
	rssSources = append(rssSources, StaticRSSSources...)

	// Sonra dillerin Google News kaynaklarını ekle
	for _, lang := range Languages {
		rssSources = append(rssSources, RSSSource{
			ID:      "gnews-" + lang.Code + "-" + strings.ToLower(lang.CountryCode),
			Name:    fmt.Sprintf("Google News (%s, %s)", lang.CountryCode, lang.Code),
			URL:     GenerateGoogleNewsRSSURL(lang),
			Enabled: true,
		})
	}

	for _, lang := range Languages {
		rssSources = append(rssSources, RSSSource{
			ID:      "bingnews-" + lang.Code + "-" + strings.ToLower(lang.CountryCode),
			Name:    fmt.Sprintf("Bing News (%s, %s)", lang.CountryCode, lang.Code),
			URL:     GenerateBingNewsRSSURL(lang),
			Enabled: false,
		})
	}

	return rssSources
}

var DefaultRSSSources = GenerateAllRSSSources()
