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

var CountryToLanguage = map[string]string{
	"AF": "ps", // Afghanistan - Pashto (diğer: Dari)
	"AL": "sq", // Albania - Albanian
	"DZ": "ar", // Algeria - Arabic
	"AS": "en", // American Samoa - English, Samoan
	"AD": "ca", // Andorra - Catalan
	"AO": "pt", // Angola - Portuguese
	"AI": "en", // Anguilla - English
	"AQ": "",   // Antarctica - No official language
	"AG": "en", // Antigua and Barbuda - English
	"AR": "es", // Argentina - Spanish
	"AM": "hy", // Armenia - Armenian
	"AW": "nl", // Aruba - Dutch, Papiamento
	"AU": "en", // Australia - English
	"AT": "de", // Austria - German
	"AZ": "az", // Azerbaijan - Azerbaijani
	"BS": "en", // Bahamas - English
	"BH": "ar", // Bahrain - Arabic
	"BD": "bn", // Bangladesh - Bengali
	"BB": "en", // Barbados - English
	"BY": "be", // Belarus - Belarusian, Russian
	"BE": "nl", // Belgium - Dutch, French, German
	"BZ": "en", // Belize - English
	"BJ": "fr", // Benin - French
	"BM": "en", // Bermuda - English
	"BT": "dz", // Bhutan - Dzongkha
	"BO": "es", // Bolivia - Spanish (also Quechua, Aymara)
	"BA": "bs", // Bosnia and Herzegovina - Bosnian, Croatian, Serbian
	"BW": "en", // Botswana - English, Tswana
	"BR": "pt", // Brazil - Portuguese
	"IO": "en", // British Indian Ocean Territory - English
	"BN": "ms", // Brunei - Malay
	"BG": "bg", // Bulgaria - Bulgarian
	"BF": "fr", // Burkina Faso - French
	"BI": "fr", // Burundi - French, Kirundi
	"KH": "km", // Cambodia - Khmer
	"CM": "fr", // Cameroon - French, English
	"CA": "en", // Canada - English, French
	"CV": "pt", // Cape Verde - Portuguese
	"KY": "en", // Cayman Islands - English
	"CF": "fr", // Central African Republic - French, Sango
	"TD": "fr", // Chad - French, Arabic
	"CL": "es", // Chile - Spanish
	"CN": "zh", // China - Chinese (Mandarin)
	"CX": "en", // Christmas Island - English
	"CC": "en", // Cocos Islands - English
	"CO": "es", // Colombia - Spanish
	"KM": "ar", // Comoros - Arabic, French
	"CG": "fr", // Congo - French
	"CD": "fr", // Congo (DRC) - French
	"CK": "en", // Cook Islands - English
	"CR": "es", // Costa Rica - Spanish
	"CI": "fr", // Côte d'Ivoire - French
	"HR": "hr", // Croatia - Croatian
	"CU": "es", // Cuba - Spanish
	"CW": "nl", // Curaçao - Dutch, Papiamento
	"CY": "el", // Cyprus - Greek, Turkish
	"CZ": "cs", // Czechia - Czech
	"DK": "da", // Denmark - Danish
	"DJ": "fr", // Djibouti - French, Arabic
	"DM": "en", // Dominica - English
	"DO": "es", // Dominican Republic - Spanish
	"EC": "es", // Ecuador - Spanish
	"EG": "ar", // Egypt - Arabic
	"SV": "es", // El Salvador - Spanish
	"GQ": "es", // Equatorial Guinea - Spanish, French, Portuguese
	"ER": "ti", // Eritrea - Tigrinya, Arabic, English
	"EE": "et", // Estonia - Estonian
	"SZ": "en", // Eswatini - English, Swazi
	"ET": "am", // Ethiopia - Amharic
	"FK": "en", // Falkland Islands - English
	"FO": "fo", // Faroe Islands - Faroese
	"FJ": "en", // Fiji - English, Fijian, Hindi
	"FI": "fi", // Finland - Finnish, Swedish
	"FR": "fr", // France - French
	"GF": "fr", // French Guiana - French
	"PF": "fr", // French Polynesia - French
	"GA": "fr", // Gabon - French
	"GM": "en", // Gambia - English
	"GE": "ka", // Georgia - Georgian
	"DE": "de", // Germany - German
	"GH": "en", // Ghana - English
	"GI": "en", // Gibraltar - English
	"GR": "el", // Greece - Greek
	"GL": "kl", // Greenland - Greenlandic
	"GD": "en", // Grenada - English
	"GP": "fr", // Guadeloupe - French
	"GU": "en", // Guam - English, Chamorro
	"GT": "es", // Guatemala - Spanish
	"GG": "en", // Guernsey - English
	"GN": "fr", // Guinea - French
	"GW": "pt", // Guinea-Bissau - Portuguese
	"GY": "en", // Guyana - English
	"HT": "fr", // Haiti - French, Haitian Creole
	"HN": "es", // Honduras - Spanish
	"HK": "zh", // Hong Kong - Chinese, English
	"HU": "hu", // Hungary - Hungarian
	"IS": "is", // Iceland - Icelandic
	"IN": "hi", // India - Hindi (many other languages)
	"ID": "id", // Indonesia - Indonesian
	"IR": "fa", // Iran - Persian
	"IQ": "ar", // Iraq - Arabic, Kurdish
	"IE": "en", // Ireland - English, Irish
	"IM": "en", // Isle of Man - English, Manx
	"IL": "he", // Israel - Hebrew
	"IT": "it", // Italy - Italian
	"JM": "en", // Jamaica - English
	"JP": "ja", // Japan - Japanese
	"JE": "en", // Jersey - English
	"JO": "ar", // Jordan - Arabic
	"KZ": "kk", // Kazakhstan - Kazakh, Russian
	"KE": "en", // Kenya - English, Swahili
	"KI": "en", // Kiribati - English
	"KP": "ko", // North Korea - Korean
	"KR": "ko", // South Korea - Korean
	"KW": "ar", // Kuwait - Arabic
	"KG": "ky", // Kyrgyzstan - Kyrgyz, Russian
	"LA": "lo", // Laos - Lao
	"LV": "lv", // Latvia - Latvian
	"LB": "ar", // Lebanon - Arabic
	"LS": "en", // Lesotho - English, Sesotho
	"LR": "en", // Liberia - English
	"LY": "ar", // Libya - Arabic
	"LI": "de", // Liechtenstein - German
	"LT": "lt", // Lithuania - Lithuanian
	"LU": "fr", // Luxembourg - French, German, Luxembourgish
	"MO": "zh", // Macau - Chinese, Portuguese
	"MG": "mg", // Madagascar - Malagasy, French
	"MW": "en", // Malawi - English, Chichewa
	"MY": "ms", // Malaysia - Malay
	"MV": "dv", // Maldives - Dhivehi
	"ML": "fr", // Mali - French
	"MT": "mt", // Malta - Maltese, English
	"MH": "en", // Marshall Islands - English, Marshallese
	"MQ": "fr", // Martinique - French
	"MR": "ar", // Mauritania - Arabic
	"MU": "en", // Mauritius - English
	"YT": "fr", // Mayotte - French
	"MX": "es", // Mexico - Spanish
	"FM": "en", // Micronesia - English
	"MD": "ro", // Moldova - Romanian
	"MC": "fr", // Monaco - French
	"MN": "mn", // Mongolia - Mongolian
	"ME": "sr", // Montenegro - Montenegrin (Serbian)
	"MS": "en", // Montserrat - English
	"MA": "ar", // Morocco - Arabic, Berber
	"MZ": "pt", // Mozambique - Portuguese
	"MM": "my", // Myanmar - Burmese
	"NA": "en", // Namibia - English
	"NR": "na", // Nauru - Nauruan, English
	"NP": "ne", // Nepal - Nepali
	"NL": "nl", // Netherlands - Dutch
	"NC": "fr", // New Caledonia - French
	"NZ": "en", // New Zealand - English, Maori
	"NI": "es", // Nicaragua - Spanish
	"NE": "fr", // Niger - French
	"NG": "en", // Nigeria - English
	"NU": "en", // Niue - English, Niuean
	"NF": "en", // Norfolk Island - English
	"MP": "en", // Northern Mariana Islands - English, Chamorro
	"NO": "no", // Norway - Norwegian
	"OM": "ar", // Oman - Arabic
	"PK": "ur", // Pakistan - Urdu, English
	"PW": "en", // Palau - English, Palauan
	"PS": "ar", // Palestine - Arabic
	"PA": "es", // Panama - Spanish
	"PG": "en", // Papua New Guinea - English, Tok Pisin, Hiri Motu
	"PY": "es", // Paraguay - Spanish, Guarani
	"PE": "es", // Peru - Spanish, Quechua, Aymara
	"PH": "en", // Philippines - English, Filipino
	"PN": "en", // Pitcairn Islands - English
	"PL": "pl", // Poland - Polish
	"PT": "pt", // Portugal - Portuguese
	"PR": "es", // Puerto Rico - Spanish, English
	"QA": "ar", // Qatar - Arabic
	"RE": "fr", // Réunion - French
	"RO": "ro", // Romania - Romanian
	"RU": "ru", // Russia - Russian
	"RW": "rw", // Rwanda - Kinyarwanda, French, English
	"BL": "fr", // Saint Barthélemy - French
	"SH": "en", // Saint Helena - English
	"KN": "en", // Saint Kitts and Nevis - English
	"LC": "en", // Saint Lucia - English
	"MF": "fr", // Saint Martin - French
	"PM": "fr", // Saint Pierre and Miquelon - French
	"VC": "en", // Saint Vincent and the Grenadines - English
	"WS": "sm", // Samoa - Samoan, English
	"SM": "it", // San Marino - Italian
	"ST": "pt", // São Tomé and Príncipe - Portuguese
	"SA": "ar", // Saudi Arabia - Arabic
	"SN": "fr", // Senegal - French
	"RS": "sr", // Serbia - Serbian
	"SC": "en", // Seychelles - English, French, Seychellois Creole
	"SL": "en", // Sierra Leone - English
	"SG": "en", // Singapore - English, Malay, Mandarin, Tamil
	"SX": "nl", // Sint Maarten - Dutch
	"SK": "sk", // Slovakia - Slovak
	"SI": "sl", // Slovenia - Slovene
	"SB": "en", // Solomon Islands - English
	"SO": "so", // Somalia - Somali, Arabic
	"ZA": "en", // South Africa - 11 official languages (English common)
	"SS": "en", // South Sudan - English
	"ES": "es", // Spain - Spanish
	"LK": "si", // Sri Lanka - Sinhala, Tamil
	"SD": "ar", // Sudan - Arabic, English
	"SR": "nl", // Suriname - Dutch
	"SJ": "no", // Svalbard and Jan Mayen - Norwegian
	"SE": "sv", // Sweden - Swedish
	"CH": "de", // Switzerland - German, French, Italian, Romansh
	"SY": "ar", // Syria - Arabic
	"TW": "zh", // Taiwan - Chinese (Mandarin)
	"TJ": "tg", // Tajikistan - Tajik
	"TZ": "sw", // Tanzania - Swahili, English
	"TH": "th", // Thailand - Thai
	"TL": "pt", // Timor-Leste - Portuguese
	"TG": "fr", // Togo - French
	"TK": "en", // Tokelau - English
	"TO": "to", // Tonga - Tongan, English
	"TT": "en", // Trinidad and Tobago - English
	"TN": "ar", // Tunisia - Arabic
	"TR": "tr", // Turkey - Turkish
	"TM": "tk", // Turkmenistan - Turkmen
	"TC": "en", // Turks and Caicos Islands - English
	"TV": "en", // Tuvalu - Tuvaluan, English
	"UG": "en", // Uganda - English, Swahili
	"UA": "uk", // Ukraine - Ukrainian
	"AE": "ar", // United Arab Emirates - Arabic
	"GB": "en", // United Kingdom - English
	"US": "en", // United States - English
	"UY": "es", // Uruguay - Spanish
	"UZ": "uz", // Uzbekistan - Uzbek
	"VU": "bi", // Vanuatu - Bislama, English, French
	"VE": "es", // Venezuela - Spanish
	"VN": "vi", // Vietnam - Vietnamese
	"VG": "en", // British Virgin Islands - English
	"VI": "en", // U.S. Virgin Islands - English
	"WF": "fr", // Wallis and Futuna - French
	"EH": "ar", // Western Sahara - Arabic
	"YE": "ar", // Yemen - Arabic
	"ZM": "en", // Zambia - English
	"ZW": "en", // Zimbabwe - English
}
