package helpers

import "github.com/gosimple/slug"

func GenerateSlug(s string) string {
	return slug.Make(s)
}
