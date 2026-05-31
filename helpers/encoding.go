package helpers

import (
	"html"
	"strings"
	"unicode"
	"unicode/utf8"
)

var windows1252Reverse = map[rune]byte{
	0x20AC: 0x80,
	0x201A: 0x82,
	0x0192: 0x83,
	0x201E: 0x84,
	0x2026: 0x85,
	0x2020: 0x86,
	0x2021: 0x87,
	0x02C6: 0x88,
	0x2030: 0x89,
	0x0160: 0x8A,
	0x2039: 0x8B,
	0x0152: 0x8C,
	0x017D: 0x8E,
	0x2018: 0x91,
	0x2019: 0x92,
	0x201C: 0x93,
	0x201D: 0x94,
	0x2022: 0x95,
	0x2013: 0x96,
	0x2014: 0x97,
	0x02DC: 0x98,
	0x2122: 0x99,
	0x0161: 0x9A,
	0x203A: 0x9B,
	0x0153: 0x9C,
	0x017E: 0x9E,
	0x0178: 0x9F,
}

func NormalizeByLang(input string, lang string) string {
	normalized := strings.TrimSpace(html.UnescapeString(input))
	if normalized == "" {
		return ""
	}

	best := normalized
	bestScore := scoreNormalizedText(best, lang)

	for range 3 {
		candidate, ok := reinterpretWindows1252AsUTF8(best)
		if !ok {
			break
		}

		candidateScore := scoreNormalizedText(candidate, lang)
		if candidateScore <= bestScore {
			break
		}

		best = candidate
		bestScore = candidateScore
	}

	return strings.TrimSpace(stripUnsafeControlRunes(best))
}

func reinterpretWindows1252AsUTF8(input string) (string, bool) {
	raw, ok := toWindows1252Bytes(input)
	if !ok || len(raw) == 0 || !utf8.Valid(raw) {
		return "", false
	}

	decoded := string(raw)
	if decoded == input {
		return "", false
	}

	return decoded, true
}

func toWindows1252Bytes(input string) ([]byte, bool) {
	raw := make([]byte, 0, len(input))
	convertedAny := false

	for _, r := range input {
		if b, ok := windows1252Reverse[r]; ok {
			raw = append(raw, b)
			convertedAny = true
			continue
		}

		if r <= 0xFF {
			raw = append(raw, byte(r))
			if r >= 0x80 {
				convertedAny = true
			}
			continue
		}

		return nil, false
	}

	return raw, convertedAny
}

func stripUnsafeControlRunes(input string) string {
	var builder strings.Builder
	builder.Grow(len(input))

	for _, r := range input {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			builder.WriteRune(r)
		case unicode.IsControl(r):
			continue
		default:
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func scoreNormalizedText(input string, lang string) int {
	score := 0
	language := strings.ToLower(strings.TrimSpace(lang))

	for _, r := range input {
		switch {
		case r == utf8.RuneError:
			score -= 60
		case r == '\ufffd':
			score -= 30
		case unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t':
			score -= 30
		case unicode.Is(unicode.Hiragana, r):
			score += languageWeight(language, "ja", 18, 10)
		case unicode.Is(unicode.Katakana, r):
			score += languageWeight(language, "ja", 16, 9)
		case unicode.Is(unicode.Hangul, r):
			score += languageWeight(language, "ko", 18, 10)
		case unicode.Is(unicode.Han, r):
			score += hanWeight(language)
		case unicode.Is(unicode.Cyrillic, r):
			score += languageWeight(language, "ru", 15, 8)
		case unicode.IsLetter(r):
			score += 3
		case unicode.IsDigit(r):
			score += 2
		case strings.ContainsRune(" \n\r\t/-:,.!?()[]'\"&", r):
			score += 1
		}
	}

	for _, marker := range suspiciousMarkers {
		score -= strings.Count(input, marker) * 40
	}

	return score
}

func languageWeight(language, target string, preferred, fallback int) int {
	if language == target {
		return preferred
	}
	return fallback
}

func hanWeight(language string) int {
	switch language {
	case "ja", "zh":
		return 14
	case "ko":
		return 6
	default:
		return 8
	}
}

var suspiciousMarkers = []string{
	"Ã", "Â", "â€", "â€™", "â€œ", "\u00e2\u20ac\u009d", "â€¦", "ãƒ", "ã‚", "ã€", "Ð", "Ñ", "ï¿½", "锟", "嚙",
}
