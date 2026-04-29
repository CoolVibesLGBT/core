package handlers

import (
	"core/middleware"

	"github.com/gofiber/fiber/v3"
)

func BasePage(c fiber.Ctx, data fiber.Map) fiber.Map {

	base := fiber.Map{
		"Categories": c.Locals("Categories"),
		"Stars":      c.Locals("Stars"),
		"Path":       c.Path(),
	}

	for k, v := range data {
		base[k] = v
	}

	return base
}

func HandleHomePage() fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)
		filters, _ := ParseFilters(c, authUser)
		return c.Render("pages/home",
			BasePage(c, fiber.Map{
				"Title":   "Home",
				"Filters": filters,
			}),
			"layouts/main",
		)
	}
}

func HandleVideoPage() fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)
		filters, _ := ParseFilters(c, authUser)
		return c.Render("pages/video",
			BasePage(c, fiber.Map{
				"Title":   "Video",
				"Filters": filters,
			}),
			"layouts/main",
		)
	}
}

func HandleCategoriesPage() fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)
		filters, _ := ParseFilters(c, authUser)

		return c.Render("pages/categories/index",
			BasePage(c, fiber.Map{
				"Title":   filters.Slug,
				"Filters": filters,
			}),
			"layouts/main",
		)
	}
}

func HandleCategoriesDetailPage() fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)
		filters, _ := ParseFilters(c, authUser)

		return c.Render("pages/categories/detail",
			BasePage(c, fiber.Map{
				"Title":   filters.Slug,
				"Filters": filters,
			}),
			"layouts/main",
		)
	}
}

func HandleModelsPage() fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)
		filters, _ := ParseFilters(c, authUser)
		slug := c.Params("slug")

		return c.Render("pages/models/index",
			BasePage(c, fiber.Map{
				"Title":   slug,
				"Filters": filters,
			}),
			"layouts/main",
		)
	}
}

func HandleModelDetailsPage() fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)
		filters, _ := ParseFilters(c, authUser)
		slug := c.Params("slug")

		return c.Render("pages/models/detail",
			BasePage(c, fiber.Map{
				"Title":   slug,
				"Filters": filters,
			}),
			"layouts/main",
		)
	}
}

func HandleProfilePage() fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)
		filters, _ := ParseFilters(c, authUser)
		slug := c.Params("slug")
		return c.Render("pages/profile",
			BasePage(c, fiber.Map{
				"Title":   slug,
				"Filters": filters,
			}),
			"layouts/main",
		)
	}
}
