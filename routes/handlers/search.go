package handlers

import (
	"core/application/usecases"
	"core/constants"
	"core/models"
	"core/models/post"
	"core/types"
	"core/utils"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const (
	searchScopeAll      = "all"
	searchScopePeople   = "people"
	searchScopeEvents   = "events"
	searchScopePosts    = "posts"
	searchScopePlaces   = "places"
	searchScopeLocation = "locations"
)

// HandleGlobalSearch serves the web search screen with one stable response.
// The underlying repositories remain responsible for their own result shape,
// while this handler keeps the public API grouped by the UI filters.
func HandleGlobalSearch(userService *usecases.UserService, postService *usecases.PostService) fiber.Handler {
	return func(c fiber.Ctx) error {
		query := strings.TrimSpace(getParam(c, "query"))
		if query == "" {
			query = strings.TrimSpace(getParam(c, "search"))
		}
		if query == "" {
			query = strings.TrimSpace(getParam(c, "q"))
		}
		if len([]rune(query)) < 2 {
			return utils.SendErrorWithMessage(
				c,
				fiber.StatusBadRequest,
				constants.ErrInvalidInput,
				"query must contain at least 2 characters",
			)
		}

		filters, err := ParseFilters(c, nil)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		if filters.Limit <= 0 {
			filters.Limit = constants.DEFAULT_LIMIT
		}
		// A global request can fan out to four collections. Keep one request
		// bounded even if a caller sends the maximum generic API limit.
		if filters.Limit > constants.DEFAULT_LIMIT {
			filters.Limit = constants.DEFAULT_LIMIT
		}
		filters.Search = &query

		scope := strings.ToLower(strings.TrimSpace(getParam(c, "scope")))
		if scope == "" {
			scope = strings.ToLower(strings.TrimSpace(getParam(c, "type")))
		}
		if scope == "" {
			scope = searchScopeAll
		}
		if scope == searchScopeLocation {
			scope = searchScopePlaces
		}
		if scope != searchScopeAll && scope != searchScopePeople && scope != searchScopeEvents && scope != searchScopePosts && scope != searchScopePlaces {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "invalid search scope")
		}

		result := types.GlobalSearchResult{
			Query:  query,
			Users:  []models.User{},
			Events: []post.Post{},
			Posts:  []post.Post{},
			Places: []*post.Post{},
		}

		if scope == searchScopeAll || scope == searchScopePeople {
			users, searchErr := userService.SearchUsers(filters)
			if searchErr != nil {
				return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, "failed to search users: "+searchErr.Error())
			}
			result.Users = users
		}

		if scope == searchScopeAll || scope == searchScopePosts {
			postFilters := filters
			postFilters.PostKind = ""
			posts, searchErr := postService.SearchPost(postFilters)
			if searchErr != nil {
				return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, "failed to search posts: "+searchErr.Error())
			}
			result.Posts = posts.Posts
		}

		if scope == searchScopeAll || scope == searchScopeEvents {
			eventFilters := filters
			eventFilters.PostKind = post.PostKindEvent
			events, searchErr := postService.SearchPost(eventFilters)
			if searchErr != nil {
				return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, "failed to search events: "+searchErr.Error())
			}
			result.Events = events.Posts
		}

		if scope == searchScopeAll || scope == searchScopePlaces {
			placeFilters := filters
			placeFilters.PostKind = post.PostKindPlace
			places, searchErr := postService.SearchPost(placeFilters)
			if searchErr != nil {
				return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrDatabaseError, "failed to search places: "+searchErr.Error())
			}
			for index := range places.Posts {
				result.Places = append(result.Places, &places.Posts[index])
			}
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, result, "Search results fetched successfully")
	}
}
