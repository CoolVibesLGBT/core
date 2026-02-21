package helpers

import (
	"strings"
	"unicode"

	"github.com/gosimple/slug"
)

func GenerateSlug(s string) string {
	return slug.Make(s)
}

func SlugifyStrict(input string) string {
	s := strings.ToLower(input)
	var builder strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
