package handlers

import (
	"core/constants"
	"core/models"
	"core/models/post"
	"core/types"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func ParseFilters(c fiber.Ctx, authUser *models.User) (types.Filter, error) {
	filter := types.Filter{
		AuthUser: authUser,
		Context:  c.Context(),
		Limit:    constants.DEFAULT_LIMIT,
	}

	// user_id
	if userIdStr := c.FormValue("user_id"); userIdStr != "" {
		if userUUID, err := uuid.Parse(userIdStr); err == nil {
			filter.UserUUID = userUUID
		} else if userId, err := strconv.ParseInt(userIdStr, 10, 64); err == nil {
			filter.UserID = userId
		} else {
			return filter, fmt.Errorf("invalid user id: %s", userIdStr)
		}
	}

	// post_id
	postIdStr := c.FormValue("post_id")
	if postIdStr != "" {
		if postUUID, err := uuid.Parse(postIdStr); err == nil {
			filter.PostUUID = postUUID
		} else if postId, err := strconv.ParseInt(postIdStr, 10, 64); err == nil {
			filter.PostID = postId
		} else {
			return filter, fmt.Errorf("invalid post id: %s", postIdStr)
		}
	}

	// limit
	if limitStr := c.FormValue("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > constants.MAXIMUM_LIMIT {
				l = constants.MAXIMUM_LIMIT
			}
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
	}

	//search
	search := c.Query("search")
	if search == "" {
		search = c.Query("q")
	}
	if search == "" {
		search = c.FormValue("search")
	}
	if search == "" {
		search = c.FormValue("q")
	}
	if search != "" {
		filter.Search = &search
	}

	//slug
	slug := c.Query("slug")
	if slug == "" {
		slug = c.Query("slug")
	}
	if slug == "" {
		slug = c.FormValue("slug")
	}
	if slug == "" {
		slug = c.FormValue("slug")
	}
	if slug != "" {
		filter.Slug = &slug
	}

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

	// Distance
	if distStr := c.FormValue("distance"); distStr != "" {
		dist, err := strconv.ParseFloat(distStr, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid distance")
		}
		filter.Distance = &dist
	}

	// Post Kind
	if pk := c.FormValue("post_kind"); pk != "" {
		if post.IsValidPostKind(pk) {
			v := post.PostKind(pk)
			filter.PostKind = v
		} else {
			filter.PostKind = post.PostKindPost
		}
	}

	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Hostname()
	}

	if host != "" {
		kind := string(models.GetDomainKind(host))
		filter.Domain = &kind
	}

	return filter, nil
}
