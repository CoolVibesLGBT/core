package handlers

import (
	"core/constants"
	"core/models"
	"core/models/post"
	"core/types"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/paginate"
	"github.com/google/uuid"
)

func getParam(c fiber.Ctx, key string) string {
	v := c.Params(key)
	if v == "" {
		v = c.Query(key)
	}
	if v == "" {
		v = c.FormValue(key)
	}
	return v
}

func ParseFilters(c fiber.Ctx, authUser *models.User) (types.Filter, error) {
	filter := types.Filter{
		AuthUser: authUser,
		Context:  c.Context(),
		Limit:    constants.DEFAULT_LIMIT,
	}
	if pageInfo, ok := paginate.FromContext(c); ok {
		if pageInfo.Limit > 0 {
			filter.Limit = pageInfo.Limit
		}
		applyCursorValues(&filter, pageInfo.CursorValues())
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
	if cursorStr := getParam(c, "cursor"); cursorStr != "" {
		if values, ok := types.DecodePaginationCursor(cursorStr); ok {
			applyCursorValues(&filter, values)
		} else {
			cVal, err := strconv.ParseInt(cursorStr, 10, 64)
			if err != nil {
				return filter, fmt.Errorf("invalid cursor")
			}
			filter.Cursor = &cVal
		}
	}

	// Search supports both the historical `search` key and the shorter `q`
	// alias. Trim it here so every handler receives the same value regardless
	// of whether the request came from a form body or a query string.
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
	search = strings.TrimSpace(search)
	if search != "" {
		filter.Search = &search
	}

	if slug := getParam(c, "slug"); slug != "" {
		filter.Slug = &slug
	}

	if pillar := getParam(c, "pillar"); pillar != "" {
		filter.Pillar = &pillar
	}

	if cluster := getParam(c, "cluster"); cluster != "" {
		filter.Cluster = &cluster
	}

	if category := getParam(c, "category"); category != "" {
		filter.Category = &category
	}

	if name := getParam(c, "name"); name != "" {
		filter.Name = &name
	}

	if city := getParam(c, "city"); city != "" {
		filter.City = &city
	}

	if country := getParam(c, "country"); country != "" {
		filter.Country = &country
	}

	// Latitude (float64 pointer)
	if latStr := getParam(c, "latitude"); latStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid latitude")
		}
		filter.Latitude = &lat
	}

	// Longitude (float64 pointer)
	if longStr := getParam(c, "longitude"); longStr != "" {
		long, err := strconv.ParseFloat(longStr, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid longitude")
		}
		filter.Longitude = &long
	}

	// Distance
	if distStr := getParam(c, "distance"); distStr != "" {
		dist, err := strconv.ParseFloat(distStr, 64)
		if err != nil {
			return filter, fmt.Errorf("invalid distance")
		}
		filter.Distance = &dist
	}

	// Post Kind
	if pk := getParam(c, "post_kind"); pk != "" {
		if post.IsValidPostKind(pk) {
			v := post.PostKind(pk)
			filter.PostKind = v
		} else {
			filter.PostKind = post.PostKindPost
		}
	}

	// The web client sends the public app domain because the API host can be a
	// different hostname (for example api.coolvibes.lgbt). Prefer that value,
	// then fall back to the request host. Unknown local/proxy hosts are treated
	// as the default CoolVibes domain instead of producing an unusable
	// `domain = unknown` predicate.
	domainInput := getParam(c, "domain")
	if strings.EqualFold(strings.TrimSpace(domainInput), string(models.AllDomains)) {
		kind := string(models.AllDomains)
		filter.Domain = &kind
	} else if domainInput != "" {
		if domainKind := models.GetDomainKind(domainInput); domainKind != models.UnknownDomain {
			kind := string(domainKind)
			filter.Domain = &kind
		}
	}

	if filter.Domain == nil {
		host := c.Get("X-Forwarded-Host")
		if host == "" {
			host = c.Hostname()
		}
		kind := models.GetDomainKind(host)
		if kind == models.UnknownDomain {
			kind = models.CoolVibes
		}
		kindValue := string(kind)
		filter.Domain = &kindValue
	}

	return filter, nil
}

func applyCursorValues(filter *types.Filter, values map[string]any) {
	if len(values) == 0 {
		return
	}
	if publicID, ok := types.CursorPublicID(values); ok {
		filter.Cursor = &publicID
	}
	if distance, ok := types.CursorDistance(values); ok {
		filter.Distance = &distance
	}
}
