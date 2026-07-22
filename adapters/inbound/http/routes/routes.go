// routes/router.go
package routes

import (
	"core/adapters/inbound/http/middleware"
	"core/adapters/inbound/http/router"
	"core/adapters/inbound/http/routes/handlers"
	"core/application/ports"
	usecases "core/application/usecases"
	"core/constants"
	"core/mcp"
	"os"
	"strings"

	fiber "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/paginate"
	"github.com/gofiber/fiber/v3/middleware/static"
	html "github.com/gofiber/template/html/v2"
)

func requestBodyLimit() int {
	// Fiber and fasthttp represent BodyLimit as an int and replace zero with
	// their 4 MiB default. MaxInt-1 is therefore the largest safe no-limit
	// sentinel; the multipart parser may add one while constructing a reader.
	return int(^uint(0)>>1) - 1
}

func trustedProxySettings(raw string) (bool, fiber.TrustProxyConfig) {
	values := strings.Split(raw, ",")
	proxies := make([]string, 0, len(values))
	for _, value := range values {
		if proxy := strings.TrimSpace(value); proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	return len(proxies) > 0, fiber.TrustProxyConfig{Proxies: proxies}
}

type Router struct {
	fiber  *fiber.App
	action *router.ActionRouter
}

type Dependencies struct {
	MCPServer           *mcp.MCPServer
	UserService         *usecases.UserService
	PostService         *usecases.PostService
	PlaceService        *usecases.PlaceService
	NewsService         *usecases.NewsService
	ClassifiedService   *usecases.ClassifiedService
	MatchesService      *usecases.MatchesService
	ChatService         *usecases.ChatService
	NotificationService *usecases.NotificationsService
	PaymentService      *usecases.PaymentService
	SystemService       *usecases.SystemService
	ModerationService   *usecases.ModerationService
	BroadcastService    *usecases.BroadcastService
	PrivatePhotoService *usecases.PrivatePhotoService
	SessionService      *usecases.SessionService
	SitemapService      *usecases.SitemapService
	MediaAccessService  *usecases.MediaAccessService
	TokenDecoder        ports.UserTokenDecoder
	TelegramProcessor   ports.TelegramUpdateProcessor
}

func NewRouter(deps Dependencies) *Router {
	trustProxy, trustProxyConfig := trustedProxySettings(os.Getenv("TRUSTED_PROXIES"))
	mcpServer := deps.MCPServer
	userService := deps.UserService
	postService := deps.PostService
	placeService := deps.PlaceService
	newsService := deps.NewsService
	classifiedService := deps.ClassifiedService
	matchesService := deps.MatchesService
	chatService := deps.ChatService
	notificationService := deps.NotificationService
	paymentService := deps.PaymentService
	systemService := deps.SystemService
	moderationService := deps.ModerationService
	broadcastService := deps.BroadcastService
	privatePhotoService := deps.PrivatePhotoService
	sessionService := deps.SessionService
	sitemapService := deps.SitemapService
	tg := deps.TelegramProcessor

	engine := html.New("./views", ".html")
	r := &Router{
		action: router.NewActionRouter(),
		fiber: fiber.New(fiber.Config{
			Views:                        engine,
			ReadBufferSize:               8192,
			WriteBufferSize:              8192,
			BodyLimit:                    requestBodyLimit(),
			StreamRequestBody:            true,
			DisablePreParseMultipartForm: true,
			ProxyHeader:                  fiber.HeaderXForwardedFor,
			TrustProxy:                   trustProxy,
			TrustProxyConfig:             trustProxyConfig,
		}),
	}

	r.fiber.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders: []string{
			"Accept", "Authorization", "Content-Type", "Content-Length",
			"X-CSRF-Token", "Token", "session", "Origin", "Host", "Connection",
			"Accept-Encoding", "Accept-Language", "X-Requested-With", "Idempotency-Key", "X-Action",
		},
		AllowCredentials: false,
	}))
	// Register compression before API/packet routes. Fiber middleware only
	// wraps handlers registered after it; placing this near the web routes left
	// every mobile API response uncompressed. BestSpeed keeps CPU latency low.
	r.fiber.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	r.fiber.Use(paginate.New(
		paginate.Config{
			PageKey:      "page",
			LimitKey:     "limit",
			DefaultPage:  1,
			DefaultLimit: constants.DEFAULT_LIMIT,
			MaxLimit:     constants.MAXIMUM_LIMIT,
		},
	))

	r.fiber.Use("/static/uploads", handleMediaFile(deps.MediaAccessService, deps.TokenDecoder))
	r.fiber.Use("/static", static.New("./static"))

	r.action.Register(constants.CMD_INITIAL_SYNC, handlers.HandleInitialSync(systemService))
	r.action.Register(constants.CMD_LINK_METADATA, handlers.HandleLinkPreview())

	r.action.Register(constants.CMD_GET_VAPID_PUBLIC_KEY, handlers.HandleVapidGetKey(systemService))
	r.action.Register(constants.CMD_SET_VAPID_SUBSCRIBE, handlers.HandleVapidSubscribe(systemService), middleware.AuthMiddleware(sessionService))

	r.action.Register(constants.CMD_AUTH_CHECK, handlers.HandleAuthCheck(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_AUTH_REGISTER, handlers.HandleRegister(userService))
	r.action.Register(constants.CMD_AUTH_LOGIN, handlers.HandleLogin(userService))
	r.action.Register(constants.CMD_USER_FETCH_PROFILE, handlers.HandleFetchUserProfile(userService))
	r.action.Register(constants.CMD_USER_VIEW_PROFILE, handlers.HandleUserViewProfile(userService), middleware.AuthMiddleware(sessionService))

	r.action.Register(constants.CMD_SEARCH_LOOKUP_USER, handlers.HandleGetUsersStartingWith(userService))
	r.action.Register(constants.CMD_SEARCH_GLOBAL, handlers.HandleGlobalSearch(userService, postService))
	r.action.Register(constants.CMD_SEARCH_TRENDS, handlers.HandleGetTrends(postService))

	r.action.Register(constants.CMD_AUTH_USER_INFO, handlers.HandleUserInfo(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_PAYMENT_METHODS, handlers.HandleFetchPaymentMethods(systemService))
	r.action.Register(constants.CMD_GET_NOTIFICATIONS, handlers.HandleGetNotifications(notificationService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_GET_NOTIFICATIONS, handlers.HandleUserNotifications(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_UPDATE_PREFERENCES, handlers.HandleSetUserPreferences(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_UPDATE_USER_PROFILE, handlers.HandleUpdateUserProfile(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_FETCH_ENGAGEMENTS, handlers.HandleFetchUserEngagements(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_UPLOAD_AVATAR, handlers.HandleUploadAvatar(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_DELETE_PROFILE, handlers.HandleUserDelete(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_CHECK_IN, handlers.HandleUserCheckIn(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_CHECK_IN_FETCH, handlers.HandleFetchCheckIns(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_UPLOAD_COVER, handlers.HandleUploadCover(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_UPLOAD_STORY, handlers.HandleUploadStory(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_PRIVATE_PHOTOS_FETCH, handlers.HandleFetchPrivatePhotos(privatePhotoService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_PRIVATE_PHOTOS_UPLOAD, handlers.HandleUploadPrivatePhotos(privatePhotoService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_PRIVATE_PHOTOS_DELETE, handlers.HandleDeletePrivatePhoto(privatePhotoService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_REQUEST, handlers.HandleRequestPrivatePhotoAccess(privatePhotoService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_REQUESTS, handlers.HandleFetchPrivatePhotoAccessRequests(privatePhotoService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_RESPOND, handlers.HandleRespondPrivatePhotoAccess(privatePhotoService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_PRIVATE_PHOTOS_ACCESS_REVOKE, handlers.HandleRevokePrivatePhotoAccess(privatePhotoService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_POSTS, handlers.HandleGetPostsByUser(postService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_USER_POST_REPLIES, handlers.HandleGetRepliesByUser(postService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_USER_POST_MEDIA, handlers.HandleGetAllMediasByUser(postService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_USER_POST_LIKES, handlers.HandlePostCollectionNotImplemented(), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_USER_POST_BOOKMARKS, handlers.HandlePostCollectionNotImplemented(), middleware.AuthMiddlewareWithoutCheck(sessionService))

	//USER FOLLOW
	r.action.Register(constants.CMD_USER_FOLLOW, handlers.HandleFollow(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_UNFOLLOW, handlers.HandleUnfollow(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_TOGGLE_FOLLOW, handlers.HandleToggleFollow(userService), middleware.AuthMiddleware(sessionService))

	//USER LIKE
	r.action.Register(constants.CMD_USER_LIKE, handlers.HandleUserLike(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_DISLIKE, handlers.HandleUserDislike(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_TOGGLE_LIKE, handlers.HandleUserToggleLikeDislike(userService, true), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_TOGGLE_DISLIKE, handlers.HandleUserToggleLikeDislike(userService, false), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_BLOCK, handlers.HandleUserBlock(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_UNBLOCK, handlers.HandleUserUnblock(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_TOGGLE_BLOCK, handlers.HandleUserToggleBlock(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_REPORT, handlers.HandleUserReport(userService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_USER_TOGGLE_SUBSCRIBE, handlers.HandleUserToggleSubscribe(userService), middleware.AuthMiddleware(sessionService))

	// POST
	r.action.Register(constants.CMD_POST_CATEGORIES, handlers.HandleGetCategories(postService))
	r.action.Register(constants.CMD_POST_CREATE, handlers.HandleCreate(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_VOTE, handlers.HandleVote(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_BANANA, handlers.HandlePostBanana(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_LIKE, handlers.HandlePostLike(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_DISLIKE, handlers.HandlePostDislike(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_BOOKMARK, handlers.HandlePostBookmark(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_REPORT, handlers.HandlePostReport(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_VIEW, handlers.HandlePostView(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_EVENT_RSVP, handlers.HandlePostEventRSVP(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_FETCH, handlers.HandleFetchPost(postService))
	r.action.Register(constants.CMD_POST_GET, handlers.HandleGetBySlug(postService))

	r.action.Register(constants.CMD_MODERATION_REPORTS_FETCH, handlers.HandleModerationFetchReports(moderationService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_MODERATION_REPORT_RESOLVE, handlers.HandleModerationResolveReport(moderationService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_MODERATION_POST_HIDE, handlers.HandleModerationHidePost(moderationService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_MODERATION_POST_UNHIDE, handlers.HandleModerationUnhidePost(moderationService), middleware.AuthMiddleware(sessionService))

	r.action.Register(constants.CMD_POST_DELETE, handlers.HandlePostDelete(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_TIP, handlers.HandlePostTip(postService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_POST_TIMELINE, handlers.HandleTimeline(postService))
	r.action.Register(constants.CMD_POST_SEARCH, handlers.HandleSearchPost(postService))

	r.action.Register(constants.CMD_POST_VIBES, handlers.HandleTimelineVibes(postService))
	r.action.Register(constants.CMD_USER_FETCH_STORIES, handlers.HandleFetchStories(userService))
	r.action.Register(constants.CMD_USER_FETCH_NEARBY_USERS, handlers.HandleFetchNearbyUsers(userService), middleware.AuthMiddlewareWithoutCheck(sessionService))

	// BROADCAST
	r.action.Register(constants.CMD_USER_FETCH_BROADCASTERS, handlers.HandleFetchBroadcasts(userService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	//

	//MATCHES EKRANI ICIN
	r.action.Register(constants.CMD_MATCH_GET_UNSEEN, handlers.HandleGetUnseenUsers(matchesService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_MATCH_CREATE, handlers.HandleRecordView(matchesService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_MATCH_FETCH_MATCHED, handlers.HandleGetMatchesAfter(matchesService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_MATCH_FETCH_LIKED, handlers.HandleGetLikesAfter(matchesService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_MATCH_FETCH_PASSED, handlers.HandleGetPassesAfter(matchesService), middleware.AuthMiddleware(sessionService))

	//CHAT
	r.action.Register(constants.CMD_TYPING, handlers.HandleSendTypingEvent(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_CHAT_CREATE, handlers.HandleCreateChat(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_CHAT_MESSAGE_READ, handlers.HandleChatMessageRead(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_CHAT_MESSAGE_OPEN, handlers.HandleChatMessageOpen(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_SEND_MESSAGE, handlers.HandleSendMessage(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_FETCH_CHATS, handlers.HandleGetChatsByUserID(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_FETCH_MESSAGES, handlers.HandleGetMessagesByChatID(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_DELETE_CHAT, handlers.HandleDeleteChat(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_DELETE_MESSAGE, handlers.HandleDeleteMessage(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_DELETE_MESSAGE_FOR_USER, handlers.HandleDeleteMessageForUser(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_DELETE_MESSAGE_FOR_ALL, handlers.HandleDeleteMessageForAll(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_DELETE_CHAT_FOR_USER, handlers.HandleDeleteChatForUser(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_DELETE_CHAT_FOR_ALL, handlers.HandleDeleteChatForAll(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_PIN_MESSAGE, handlers.HandlePinMessage(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_UNPIN_MESSAGE, handlers.HandleUnpinMessage(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_CLEAR_CHAT_HISTORY_FOR_USER, handlers.HandleClearChatHistoryForUser(chatService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_CLEAR_CHAT_HISTORY_FOR_ALL, handlers.HandleClearChatHistoryForAll(chatService), middleware.AuthMiddleware(sessionService))

	//PLACE EKRANI ICIN
	r.action.Register(constants.CMD_PLACE_CREATE, handlers.HandleCreatePlace(placeService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_PLACE_FETCH, handlers.HandleGetNearByPlaces(placeService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_PLACE_CATEGORIES, handlers.HandleGetPlaceCategories(placeService))

	//NEWS EKRANI
	r.action.Register(constants.CMD_NEWS_CREATE, handlers.HandleCreateNews(newsService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_NEWS_FETCH, handlers.HandleFetchNews(newsService), middleware.AuthMiddlewareWithoutCheck(sessionService))

	//CLASSIFIELDS
	r.action.Register(constants.CMD_CLASSIFIEDS_CREATE, handlers.HandleCreateClassified(classifiedService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_FETCH_JOB_OFFERS, handlers.HandleFetchJobOffers(classifiedService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_FETCH_JOB_SEARCH, handlers.HandleFetchJobSearches(classifiedService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_CLASSIFIEDS_FETCH, handlers.HandleGetClassified(classifiedService), middleware.AuthMiddlewareWithoutCheck(sessionService))

	//BROADCAST
	r.action.Register(constants.CMD_BROADCASTS_FETCH, handlers.HandleFetchBroadcasts(userService), middleware.AuthMiddlewareWithoutCheck(sessionService))
	r.action.Register(constants.CMD_BROADCASTS_JOIN, handlers.HandleBroadcastsJoinRequest(broadcastService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_BROADCASTS_VIEW, handlers.HandleViewBroadcast(broadcastService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_BROADCASTS_CREATE, handlers.HandleCreateBroadcast(broadcastService), middleware.AuthMiddleware(sessionService))
	r.action.Register(constants.CMD_BROADCASTS_LIKE, handlers.HandleLikeBroadcast(broadcastService), middleware.AuthMiddleware(sessionService))

	//WEBHOOK

	r.fiber.All("/mcp", handlers.HandleMCPTransport(mcpServer))
	r.fiber.All("/webhook/bot/telegram/", handlers.HandleTelegramUpdates(tg))

	r.fiber.Post("/webhook/gateway/stripe/thin", handlers.HandleStripeThin(paymentService))
	r.fiber.Post("/webhook/gateway/stripe/snapshot", handlers.HandleStripeSnapshot(paymentService))
	r.fiber.All("/signin-oidc", r.handlePacket)
	r.fiber.All("/signout-callback-oidc", r.handlePacket)

	r.fiber.Get("/swagger", handlers.HandleSwaggerUI())
	r.fiber.Get("/swagger/", handlers.HandleSwaggerUI())
	r.fiber.Get("/docs", handlers.HandleSwaggerUI())
	r.fiber.Get("/docs/", handlers.HandleSwaggerUI())
	r.fiber.Get("/swagger/openapi.yaml", handlers.HandleOpenAPIYAML())
	r.fiber.Get("/swagger/openapi.json", handlers.HandleOpenAPIJSON())
	r.fiber.Get("/docs/openapi.yaml", handlers.HandleOpenAPIYAML())
	r.fiber.Get("/docs/openapi.json", handlers.HandleOpenAPIJSON())

	r.fiber.All("/test", r.handlePacket)
	r.fiber.All("/packet", r.handlePacket)

	//r.action.Register(constants.CMD_UNPIN_MESSAGE, handlers.HandleUnpinMessage(chatService), middleware.WebMiddleware(postService))

	// API ROUTERS
	api := r.fiber.Group("/api")
	api.All("/actions/:action", r.handleActionAlias)
	api.All("/", r.handlePacket)

	// WEB ROUTES

	r.fiber.Use(middleware.WebMiddleware(postService))
	r.fiber.All("/sitemap.xml", func(c fiber.Ctx) error {
		xml, err := sitemapService.Index(GetApiURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.SendString(xml)
	})

	r.fiber.All("/sitemap-posts.xml", func(c fiber.Ctx) error {
		xml, err := sitemapService.Posts(c.Context(), GetFrontendURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.Send(xml)
	})

	r.fiber.All("/sitemap-news.xml", func(c fiber.Ctx) error {
		xml, err := sitemapService.News(c.Context(), GetFrontendURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.Send(xml)
	})

	r.fiber.All("/sitemap-categories.xml", func(c fiber.Ctx) error {
		xml, err := sitemapService.Categories(c.Context(), GetFrontendURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.Send(xml)
	})

	r.fiber.Get("/sitemap-images.xml", func(c fiber.Ctx) error {
		xmlData, err := sitemapService.Images(c.Context(), GetFrontendURL(c), GetApiURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		c.Type("xml")
		return c.Send(xmlData)
	})

	r.fiber.Get("/sitemap-videos.xml", func(c fiber.Ctx) error {
		xmlData, err := sitemapService.Videos(c.Context(), GetFrontendURL(c), GetApiURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		c.Type("xml")
		return c.Send(xmlData)
	})

	r.fiber.Get("/", handlers.HandleHomePage())
	r.fiber.Get("/categories", handlers.HandleCategoriesPage())
	r.fiber.Get("/models", handlers.HandleModelsPage())
	r.fiber.Get("/models/:slug", handlers.HandleModelDetailsPage())
	r.fiber.Get("/video/:slug", handlers.HandleVideoPage())
	r.fiber.Get("/:slug", handlers.HandleCategoriesPage())
	r.fiber.Get("/:pillar/:cluster", handlers.HandleCategoriesDetailPage())

	return r
}

func GetFrontendURL(c fiber.Ctx) string {
	return "https://" + strings.TrimPrefix(c.Hostname(), "api.")
}

func GetApiURL(c fiber.Ctx) string {
	return "https://" + c.Hostname()
}

func (r *Router) handlePacket(c fiber.Ctx) error {
	var action string
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	c.Set("Access-Control-Allow-Headers", "Accept,Authorization,Content-Type,X-CSRF-Token,Token,session,Origin,Host,Connection,Accept-Encoding,Accept-Language,X-Requested-With,Idempotency-Key,X-Action")
	if c.Method() == fiber.MethodOptions {
		return c.SendStatus(fiber.StatusNoContent)
	}

	// Prefer metadata that can be read without parsing the request body. This
	// lets upload handlers start consuming the streamed multipart body without
	// first materializing it just to discover the action. Body action remains
	// the backward-compatible fallback.
	action = strings.TrimSpace(c.Get("X-Action"))
	if action == "" {
		action = strings.TrimSpace(c.Query("action"))
	}

	switch c.Method() {
	case fiber.MethodGet:
		// Header/query action has already been resolved above.

	case fiber.MethodPost:
		if action == "" {
			contentType := c.Get("Content-Type")
			if strings.Contains(contentType, "application/json") {
				var packet struct {
					Action string `json:"action"`
				}

				if err := c.Bind().JSON(&packet); err != nil {
					return c.Status(fiber.StatusBadRequest).SendString("invalid JSON body")
				}
				action = packet.Action
			} else {
				action = c.FormValue("action")
			}
		}

	default:
		return c.Status(fiber.StatusMethodNotAllowed).SendString("method not allowed")
	}

	if action == "" {
		return c.SendString("Default handler executed")
	}

	return r.dispatchAction(c, action)
}

func (r *Router) handleActionAlias(c fiber.Ctx) error {
	action := c.Params("action")
	if action == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Unknown action")
	}

	return r.dispatchAction(c, action)
}

func (r *Router) dispatchAction(c fiber.Ctx, action string) error {
	route, ok := r.action.GetHandler(action)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Unknown action")
	}

	handler := route.Handler
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	return handler(c)
}

func (r *Router) GetFiber() *fiber.App {
	return r.fiber
}
