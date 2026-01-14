package handlers

import (
	"coolvibes/middleware"
	services "coolvibes/services/user"
	"coolvibes/types"
	"mime/multipart"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type PlaceHandler struct {
	service *services.PlaceService
}

func NewPlaceHandler(service *services.PlaceService) *PlaceHandler {
	return &PlaceHandler{service: service}
}

func HandleCreatePlace(s *services.PlaceService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Could not parse multipart form: " + err.Error(),
			})
		}

		formParams := form.Value
		images := form.File["images[]"]
		videos := form.File["videos[]"]

		files := append([]*multipart.FileHeader{}, images...)
		files = append(files, videos...)

		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "User not authenticated",
			})
		}

		post, err := s.CreatePlace(formParams, files, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to create post: " + err.Error(),
			})
		}

		return c.Status(fiber.StatusCreated).JSON(post)
	}
}

func HandleGetNearByPlaces(s *services.PlaceService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, _ := middleware.GetAuthenticatedUser(c)

		limitStr := c.FormValue("limit")
		limit := 10
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		var lat *float64
		latStr := c.FormValue("latitude")
		if latStr != "" {
			if v, err := strconv.ParseFloat(latStr, 64); err == nil {
				lat = &v
			}
		}

		var lon *float64
		lonStr := c.FormValue("longitude")
		if lonStr != "" {
			if v, err := strconv.ParseFloat(lonStr, 64); err == nil {
				lon = &v
			}
		}

		var cursor *int64
		cursorStr := c.FormValue("cursor")
		if cursorStr != "" {
			cVal, err := strconv.ParseInt(cursorStr, 10, 64)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "invalid cursor",
				})
			}
			cursor = &cVal
		}

		category := c.FormValue("category")
		name := c.FormValue("name")
		city := c.FormValue("city")
		country := c.FormValue("country")

		filters := types.PlaceFilters{
			Latitude:  lat,
			Longitude: lon,
			Cursor:    cursor,
			Limit:     limit,
			Category:  &category,
			Name:      &name,
			City:      &city,
			Country:   &country,
		}

		places, cursorInfo, err := s.GetNearByPlaces(c.Context(), authUser, filters)

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"success": err == nil,
			"error":   err,
			"places":  places,
			"cursor":  cursorInfo,
		})
	}
}
