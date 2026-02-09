package handlers

import (
	"core/models"
	"core/types"
	"fmt"
	"math"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func ParseFilters(c *fiber.Ctx, authUser *models.User) (types.Filter, error) {
	filter := types.Filter{
		AuthUser: authUser,
		Context:  c.Context(),
		Limit:    20, // default
	}

	// limit
	if limitStr := c.FormValue("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filter.Limit = l
		} else {
			return filter, fmt.Errorf("invalid limit")
		}
	}

	// cursor
	if cursorStr := c.FormValue("cursor"); cursorStr != "" {
		cVal, err := strconv.ParseInt(cursorStr, 10, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid cursor")
		}
		filter.Cursor = &cVal
	} else {
		maxInt64 := int64(math.MaxInt64)
		filter.Cursor = &maxInt64
	}

	// Search (optional string pointer)
	if search := c.FormValue("search"); search != "" {
		filter.Search = &search
	}

	// Category
	if category := c.FormValue("category"); category != "" {
		filter.Category = &category
	}

	// Name
	if name := c.FormValue("name"); name != "" {
		filter.Name = &name
	}

	// City
	if city := c.FormValue("city"); city != "" {
		filter.City = &city
	}

	// Country
	if country := c.FormValue("country"); country != "" {
		filter.Country = &country
	}

	// Latitude (float64 pointer)
	if latStr := c.FormValue("latitude"); latStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid latitude")
		}
		filter.Latitude = &lat
	}

	// Longitude (float64 pointer)
	if longStr := c.FormValue("longitude"); longStr != "" {
		long, err := strconv.ParseFloat(longStr, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid longitude")
		}
		filter.Longitude = &long
	}

	return filter, nil
}
