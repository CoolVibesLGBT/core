package middleware

import (
	usecases "core/application/usecases"
	"core/types"

	"github.com/gofiber/fiber/v3"
)

func WebMiddleware(s *usecases.PostService) fiber.Handler {

	var (
		cachedCategories  any
		cachedStars       any
		cachedPreferences any
	)

	return func(c fiber.Ctx) error {

		if cachedCategories == nil || cachedStars == nil || cachedPreferences == nil {
			domain := "coolvies"

			categories, err := s.GetPillarsWithClusters(types.Filter{
				Domain: &domain,
			})

			stars, _, userErr := s.FetchNearbyUsers(types.Filter{
				Domain: &domain,
				Limit:  20,
			})

			preferences, prefErr := s.GetPreferences()

			if err == nil {
				cachedCategories = categories
			}

			if userErr == nil {
				cachedStars = stars
			}

			if prefErr == nil {
				cachedPreferences = preferences
			}
		}

		c.Locals("Categories", cachedCategories)
		c.Locals("Stars", cachedStars)
		c.Locals("Preferences", cachedPreferences)
		return c.Next()
	}
}
