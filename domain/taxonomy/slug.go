package taxonomy

import (
	"strings"
	"unicode"

	"github.com/gosimple/slug"
)

func NormalizeSlug(value string) string {
	return slug.Make(strings.ReplaceAll(value, "_", "-"))
}

func StrictSlug(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
