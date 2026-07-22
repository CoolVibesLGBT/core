package handlers

import (
	"core/adapters/inbound/http/middleware"
	"core/application/ports"
	"core/application/types"
	usecases "core/application/usecases"
	"core/constants"
	"core/utils"
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type BroadcastHandler struct {
	service *usecases.BroadcastService
}

func NewBroadcastHandler(service *usecases.BroadcastService) *BroadcastHandler {
	return &BroadcastHandler{service: service}
}

func HandleFetchBroadcastsH(service *usecases.BroadcastService) fiber.Handler {
	return handleExternalBroadcastFetch(service, ports.BroadcastProviderHornet)
}

func HandleFetchBroadcastsG(service *usecases.BroadcastService) fiber.Handler {
	return handleExternalBroadcastFetch(service, ports.BroadcastProviderGrowlr)
}

func handleExternalBroadcastFetch(service *usecases.BroadcastService, provider ports.BroadcastProvider) fiber.Handler {
	return func(c fiber.Ctx) error {
		if _, ok := authenticatedBroadcastPrincipal(c); !ok {
			return broadcastServiceError(c, ports.ErrBroadcastUnauthorized)
		}
		body, err := service.FetchTrending(c.Context(), provider, ports.BroadcastTrendingQuery{
			PageSize: 10_000,
			Gender:   "all",
			More:     true,
			Score:    "0",
		})
		if err != nil {
			return broadcastServiceError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, string(body), "Broadcasts fetched successfully")
	}
}

// HandleFetchBroadcasts returns locally imported live users. External provider
// access is intentionally kept in BroadcastService-backed handlers.
func HandleFetchBroadcasts(service *usecases.UserService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, _ := middleware.GetAuthenticatedUser(c)

		filters, err := ParseFilters(c, authUser)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}

		isLive := true
		filters.IsLive = &isLive
		users, err := service.FetchLiveUsers(filters)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrDatabaseError)
		}

		var cursor types.Cursor
		if len(users) > 0 {
			last := users[len(users)-1]
			nextCursor, cursorErr := types.NewPublicIDDistanceCursor(last.PublicID, filters.Distance)
			if cursorErr != nil {
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
			cursor.Next = nextCursor
			cursor.Distance = filters.Distance
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, map[string]interface{}{
			"users":  users,
			"cursor": cursor,
		}, "Broadcasts fetched successfully")
	}
}

func HandleCreateBroadcast(service *usecases.BroadcastService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := authenticatedBroadcastPrincipal(c)
		if !ok {
			return broadcastServiceError(c, ports.ErrBroadcastUnauthorized)
		}
		body, err := service.CreateBroadcast(c.Context(), principal, broadcastRequestField(c, "streamDescription", "stream_description"))
		if err != nil {
			return broadcastServiceError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, string(body), "Broadcast created")
	}
}

func HandleViewBroadcast(service *usecases.BroadcastService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := authenticatedBroadcastPrincipal(c)
		if !ok {
			return broadcastServiceError(c, ports.ErrBroadcastUnauthorized)
		}
		broadcastID := strings.TrimSpace(broadcastRequestField(c, "broadcastId", "broadcast_id"))
		if broadcastID == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "broadcastId is required")
		}

		body, err := service.ViewBroadcast(c.Context(), principal, broadcastID)
		if err != nil {
			return broadcastServiceError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, string(body), "Viewer registered")
	}
}

func HandleBroadcastsJoinRequest(service *usecases.BroadcastService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := authenticatedBroadcastPrincipal(c)
		if !ok {
			return broadcastServiceError(c, ports.ErrBroadcastUnauthorized)
		}
		broadcastID := strings.TrimSpace(broadcastRequestField(c, "broadcastId", "broadcast_id"))
		streamClientID := strings.TrimSpace(broadcastRequestField(c, "streamClientId", "stream_client_id"))
		if broadcastID == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "broadcastId is required")
		}
		if streamClientID == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "streamClientId is required")
		}

		body, err := service.RequestGuestBroadcast(c.Context(), principal, broadcastID, streamClientID)
		if err != nil {
			return broadcastServiceError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusCreated, string(body), "Guest request sent")
	}
}

func HandleLikeBroadcast(service *usecases.BroadcastService) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, ok := authenticatedBroadcastPrincipal(c)
		if !ok {
			return broadcastServiceError(c, ports.ErrBroadcastUnauthorized)
		}
		broadcastID := strings.TrimSpace(broadcastRequestField(c, "broadcastId", "broadcast_id"))
		viewerID := strings.TrimSpace(broadcastRequestField(c, "viewerId", "viewer_id"))
		numLikesRaw := strings.TrimSpace(broadcastRequestField(c, "numLikes", "num_likes"))
		if broadcastID == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "broadcastId is required")
		}
		if viewerID == "" {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "viewerId is required")
		}
		numLikes, err := strconv.Atoi(numLikesRaw)
		if err != nil || numLikes <= 0 {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "numLikes must be a positive number")
		}

		body, err := service.LikeBroadcast(c.Context(), principal, broadcastID, viewerID, numLikes)
		if err != nil {
			return broadcastServiceError(c, err)
		}
		return utils.SendSuccessWithMessage(c, fiber.StatusOK, string(body), "Like sent")
	}
}

func broadcastServiceError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ports.ErrBroadcastUnauthorized):
		return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
	case errors.Is(err, ports.ErrBroadcastInvalidInput), errors.Is(err, ports.ErrBroadcastProviderUnsupported):
		return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
	case errors.Is(err, ports.ErrBroadcastIntegrationDisabled):
		return utils.SendErrorWithMessage(c, fiber.StatusServiceUnavailable, constants.ErrNetworkError, "Broadcast integration is unavailable")
	case errors.Is(err, ports.ErrBroadcastUpstream):
		return utils.SendErrorWithMessage(c, fiber.StatusBadGateway, constants.ErrNetworkError, "Broadcast provider request failed")
	default:
		return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
	}
}

func authenticatedBroadcastPrincipal(c fiber.Ctx) (ports.BroadcastPrincipal, bool) {
	user, ok := middleware.GetAuthenticatedUser(c)
	if !ok || user == nil || user.ID == uuid.Nil {
		return ports.BroadcastPrincipal{}, false
	}
	return ports.BroadcastPrincipal{UserID: user.ID.String()}, true
}

func broadcastRequestField(c fiber.Ctx, names ...string) string {
	for _, name := range names {
		if value := requestField(c, name); value != "" {
			return value
		}
	}
	return ""
}
