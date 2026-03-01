// routes/router.go
package routes

import (
	"core/constants"
	"core/helpers"
	"core/mcp"
	"core/middleware"
	"core/repositories"
	"core/router"
	"core/routes/handlers"
	telegramService "core/services/bot/telegram"
	services "core/services/user"
	"fmt"
	"strings"

	fiber "github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/google/wire"
	"github.com/oschwald/maxminddb-golang"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewRouter, GeoIPDBProvider)

type Router struct {
	fiber         *fiber.App
	action        *router.ActionRouter
	db            *gorm.DB
	snowFlakeNode *helpers.Node
	MCPServer     *mcp.MCPServer
	GEOIPDB       *maxminddb.Reader

	TelegramService *telegramService.Service

	AIService           *services.AIService
	NewsService         *services.NewsService
	PostService         *services.PostService
	UserService         *services.UserService
	NotificationService *services.NotificationsService
	PlaceService        *services.PlaceService
	ChatService         *services.ChatService
	PaymentService      *services.PaymentService
}

func GeoIPDBProvider() (*maxminddb.Reader, error) {
	paths := []string{
		"./static/data/GeoLite2-City.mmdb",
		"../static/data/GeoLite2-City.mmdb",
		"../../static/data/GeoLite2-City.mmdb",
	}
	var db *maxminddb.Reader
	var err error
	for _, p := range paths {
		db, err = maxminddb.Open(p)
		if err == nil {
			return db, nil
		}
	}
	return nil, fmt.Errorf("unable to load GeoLite2-City.mmdb from any path: last error: %w", err)
}

func NewRouter(
	db *gorm.DB,
	snowFlakeNode *helpers.Node,
	userService *services.UserService,
	postService *services.PostService,
	placeService *services.PlaceService,
	newsService *services.NewsService,
	matchesService *services.MatchesService,
	chatService *services.ChatService,
	notificationService *services.NotificationsService,
	paymentService *services.PaymentService,
	aiService *services.AIService,
	userRepo *repositories.UserRepository,
	notificationRepo *repositories.NotificationRepository,
	sitemapRepo *repositories.SitemapRepository,
	geoIPDB *maxminddb.Reader,

) *Router {

	tg, err := telegramService.New()
	if err != nil {
		helpers.Error("Telegram disabled: %v", err)
		tg = nil
	}

	r := &Router{
		action:          router.NewActionRouter(db),
		db:              db,
		TelegramService: tg,
		fiber: fiber.New(fiber.Config{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
			ProxyHeader:     fiber.HeaderXForwardedFor,
		}),
		GEOIPDB:             geoIPDB,
		MCPServer:           mcp.NewMCPServer(),
		snowFlakeNode:       snowFlakeNode,
		AIService:           aiService,
		NewsService:         newsService,
		PostService:         postService,
		UserService:         userService,
		NotificationService: notificationService,
		PlaceService:        placeService,
		ChatService:         chatService,
		PaymentService:      paymentService,
	}

	r.fiber.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders: []string{
			"Accept", "Authorization", "Content-Type", "Content-Length",
			"X-CSRF-Token", "Token", "session", "Origin", "Host", "Connection",
			"Accept-Encoding", "Accept-Language", "X-Requested-With",
		},
		AllowCredentials: false,
	}))

	r.fiber.Use("/static", static.New("./static"))
	r.action.Register(constants.CMD_AGENTS_INVOKE, handlers.HandleMCP(aiService))
	r.action.Register(constants.CMD_INITIAL_SYNC, handlers.HandleInitialSync(r.db))         // middleware yok
	r.action.Register(constants.CMD_GET_VAPID_PUBLIC_KEY, handlers.HandleVapidGetKey(r.db)) // middleware yok vapid
	r.action.Register(constants.CMD_SET_VAPID_SUBSCRIBE, handlers.HandleVapidSubscribe(r.db), middleware.AuthMiddleware(userRepo))

	// Action register
	r.action.Register(constants.CMD_AUTH_CHECK, handlers.HandleAuthCheck(userService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_AUTH_REGISTER, handlers.HandleRegister(userService))
	r.action.Register(constants.CMD_AUTH_LOGIN, handlers.HandleLogin(userService))
	r.action.Register(constants.CMD_USER_FETCH_PROFILE, handlers.HandleFetchUserProfile(userService))

	r.action.Register(constants.CMD_SEARCH_LOOKUP_USER, handlers.HandleGetUsersStartingWith(userService))
	r.action.Register(constants.CMD_SEARCH_TRENDS, handlers.HandleGetTrends(postService))

	r.action.Register( // access token'a gore user bilgisi
		constants.CMD_AUTH_USER_INFO,
		handlers.HandleUserInfo(userService),
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register( // access token'a gore user bilgisi
		constants.CMD_PAYMENT_METHODS,
		handlers.HandleFetchPaymentMethods(db),
		//middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register( // access token'a gore user bilgisi
		constants.CMD_GET_NOTIFICATIONS,
		handlers.HandleGetNotifications(notificationService),
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register( // access token'a gore user attributes guncelleme
		constants.CMD_USER_GET_NOTIFICATIONS,
		handlers.HandleUserNotifications(userService),
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register( // access token'a gore user attributes guncelleme
		constants.CMD_USER_UPDATE_PREFERENCES,
		handlers.HandleSetUserPreferences(userService),
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register( // access token'a gore user interestlerini guncelleme
		constants.CMD_UPDATE_USER_PROFILE,
		handlers.HandleUpdateUserProfile(userService),
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register( // access token'a gore user engagelentlerini guncelleme
		constants.CMD_USER_FETCH_ENGAGEMENTS,
		handlers.HandleFetchUserEngagements(userService),
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_USER_UPLOAD_AVATAR,
		handlers.HandleUploadAvatar(userService), // handler
		middleware.AuthMiddleware(userRepo),      // middleware
	)

	r.action.Register(
		constants.CMD_USER_DELETE_PROFILE,
		handlers.HandleUserDelete(userService),
		middleware.AuthMiddleware(userRepo),
	)

	r.action.Register(
		constants.CMD_USER_CHECK_IN,
		handlers.HandleUserCheckIn(userService),
		middleware.AuthMiddleware(userRepo),
	)

	r.action.Register(
		constants.CMD_USER_UPLOAD_COVER,
		handlers.HandleUploadCover(userService), // handler
		middleware.AuthMiddleware(userRepo),     // middleware
	)

	r.action.Register(
		constants.CMD_USER_UPLOAD_STORY,
		handlers.HandleUploadStory(userService), // handler
		middleware.AuthMiddleware(userRepo),     // middleware
	)

	r.action.Register(
		constants.CMD_USER_POSTS,
		handlers.HandleGetPostsByUser(postService),      // handler
		middleware.AuthMiddlewareWithoutCheck(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_USER_POST_REPLIES,
		handlers.HandleGetRepliesByUser(postService),    // handler
		middleware.AuthMiddlewareWithoutCheck(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_USER_POST_MEDIA,
		handlers.HandleGetAllMediasByUser(postService),  // handler
		middleware.AuthMiddlewareWithoutCheck(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_USER_POST_LIKES,
		handlers.HandleGetAllMediasByUser(postService),  // handler
		middleware.AuthMiddlewareWithoutCheck(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_USER_POST_BOOKMARKS,
		handlers.HandleGetAllMediasByUser(postService),  // handler
		middleware.AuthMiddlewareWithoutCheck(userRepo), // middleware
	)

	//

	//USER FOLLOW
	r.action.Register(
		constants.CMD_USER_FOLLOW,
		handlers.HandleFollow(userService),  // handler
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_USER_UNFOLLOW,
		handlers.HandleUnfollow(userService), // handler
		middleware.AuthMiddleware(userRepo),  // middleware
	)
	r.action.Register(
		constants.CMD_USER_TOGGLE_FOLLOW,
		handlers.HandleToggleFollow(userService), // handler
		middleware.AuthMiddleware(userRepo),      // middleware
	)

	//USER LIKE
	r.action.Register(
		constants.CMD_USER_LIKE,
		handlers.HandleUserLike(userService), // handler
		middleware.AuthMiddleware(userRepo),  // middleware
	)

	r.action.Register(
		constants.CMD_USER_DISLIKE,
		handlers.HandleUserDislike(userService), // handler
		middleware.AuthMiddleware(userRepo),     // middleware
	)

	r.action.Register(constants.CMD_USER_TOGGLE_LIKE,
		handlers.HandleUserToggleLikeDislike(userService, true), // handler
		middleware.AuthMiddleware(userRepo),                     // middleware
	)

	r.action.Register(constants.CMD_USER_TOGGLE_DISLIKE,
		handlers.HandleUserToggleLikeDislike(userService, false), // handler
		middleware.AuthMiddleware(userRepo),                      // middleware
	)

	r.action.Register(
		constants.CMD_USER_BLOCK,
		handlers.HandleUserBlock(userService), // handler
		middleware.AuthMiddleware(userRepo),   // middleware
	)

	r.action.Register(
		constants.CMD_USER_UNBLOCK,
		handlers.HandleUserUnblock(userService), // handler
		middleware.AuthMiddleware(userRepo),     // middleware
	)

	r.action.Register(
		constants.CMD_USER_TOGGLE_BLOCK,
		handlers.HandleUserToggleBlock(userService), // handler
		middleware.AuthMiddleware(userRepo),         // middleware
	)

	// POST

	r.action.Register(constants.CMD_POST_CATEGORIES, handlers.HandleGetCategories(postService))
	r.action.Register(constants.CMD_POST_CREATE, handlers.HandleCreate(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_VOTE, handlers.HandleVote(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_BANANA, handlers.HandlePostBanana(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_LIKE, handlers.HandlePostLike(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_DISLIKE, handlers.HandlePostDislike(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_BOOKMARK, handlers.HandlePostBookmark(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_REPORT, handlers.HandlePostReport(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_VIEW, handlers.HandlePostView(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_FETCH, handlers.HandleGetByID(postService))
	r.action.Register(constants.CMD_POST_DELETE, handlers.HandlePostDelete(postService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_POST_TIP, handlers.HandlePostTip(postService), middleware.AuthMiddleware(userRepo))

	r.action.Register(constants.CMD_POST_TIMELINE, handlers.HandleTimeline(postService))
	r.action.Register(constants.CMD_POST_VIBES, handlers.HandleTimelineVibes(postService))

	r.action.Register(constants.CMD_USER_FETCH_STORIES, handlers.HandleFetchStories(userService))
	r.action.Register(constants.CMD_USER_FETCH_NEARBY_USERS, handlers.HandleFetchNearbyUsers(userService), middleware.AuthMiddlewareWithoutCheck(userRepo))

	//MATCHES EKRANI ICIN
	r.action.Register(
		constants.CMD_MATCH_GET_UNSEEN,
		handlers.HandleGetUnseenUsers(matchesService), // handler
		middleware.AuthMiddleware(userRepo),           // middleware
	)

	r.action.Register(
		constants.CMD_MATCH_CREATE,
		handlers.HandleRecordView(matchesService), // handler
		middleware.AuthMiddleware(userRepo),       // middleware
	)

	r.action.Register(
		constants.CMD_MATCH_FETCH_MATCHED,
		handlers.HandleGetMatchesAfter(matchesService), // handler
		middleware.AuthMiddleware(userRepo),            // middleware
	)

	r.action.Register(
		constants.CMD_MATCH_FETCH_LIKED,
		handlers.HandleGetLikesAfter(matchesService), // handler
		middleware.AuthMiddleware(userRepo),          // middleware
	)

	r.action.Register(
		constants.CMD_MATCH_FETCH_PASSED,
		handlers.HandleGetPassesAfter(matchesService), // handler
		middleware.AuthMiddleware(userRepo),           // middleware
	)

	//CHAT
	r.action.Register(
		constants.CMD_TYPING,
		handlers.HandleSendTypingEvent(chatService), // handler
		middleware.AuthMiddleware(userRepo),         // middleware
	)

	r.action.Register(
		constants.CMD_CHAT_CREATE,
		handlers.HandleCreateChat(chatService), // handler
		middleware.AuthMiddleware(userRepo),    // middleware
	)

	r.action.Register(
		constants.CMD_SEND_MESSAGE,
		handlers.HandleSendMessage(chatService), // handler
		middleware.AuthMiddleware(userRepo),     // middleware
	)

	r.action.Register(
		constants.CMD_FETCH_CHATS,
		handlers.HandleGetChatsByUserID(chatService), // handler
		middleware.AuthMiddleware(userRepo),          // middleware
	)
	r.action.Register(
		constants.CMD_FETCH_MESSAGES,
		handlers.HandleGetMessagesByChatID(chatService), // handler
		middleware.AuthMiddleware(userRepo),             // middleware
	)

	r.action.Register(constants.CMD_DELETE_CHAT, handlers.HandleDeleteChat(chatService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_DELETE_MESSAGE, handlers.HandleDeleteMessage(chatService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_DELETE_MESSAGE_FOR_USER, handlers.HandleDeleteMessageForUser(chatService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_DELETE_MESSAGE_FOR_ALL, handlers.HandleDeleteMessageForAll(chatService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_DELETE_CHAT_FOR_USER, handlers.HandleDeleteChatForUser(chatService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_DELETE_CHAT_FOR_ALL, handlers.HandleDeleteChatForAll(chatService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_PIN_MESSAGE, handlers.HandlePinMessage(chatService), middleware.AuthMiddleware(userRepo))
	r.action.Register(constants.CMD_UNPIN_MESSAGE, handlers.HandleUnpinMessage(chatService), middleware.AuthMiddleware(userRepo))

	r.action.Register(
		constants.CMD_CLEAR_CHAT_HISTORY_FOR_USER,
		handlers.HandleClearChatHistoryForUser(chatService), // handler
		middleware.AuthMiddleware(userRepo),                 // middleware
	)

	r.action.Register(
		constants.CMD_CLEAR_CHAT_HISTORY_FOR_ALL,
		handlers.HandleClearChatHistoryForAll(chatService), // handler
		middleware.AuthMiddleware(userRepo),                // middleware
	)

	//PLACE EKRANI ICIN
	r.action.Register(
		constants.CMD_PLACE_FETCH,
		handlers.HandleGetNearByPlaces(placeService),    // handler
		middleware.AuthMiddlewareWithoutCheck(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_PLACE_CATEGORIES,
		handlers.HandleGetPlaceCategories(placeService),
	)

	//NEWS EKRANI
	r.action.Register(
		constants.CMD_NEWS_FETCH,
		handlers.HandleFetchNews(newsService),           // handler
		middleware.AuthMiddlewareWithoutCheck(userRepo), // middleware
	)

	//WEBHOOK

	r.fiber.All("/webhook/bot/telegram/", handlers.HandleTelegramUpdates(tg))

	r.fiber.Post("/webhook/gateway/stripe/thin", handlers.HandleStripeThin(paymentService))
	r.fiber.Post("/webhook/gateway/stripe/snapshot", handlers.HandleStripeSnapshot(paymentService))
	r.fiber.All("/signin-oidc", r.handlePacket)
	r.fiber.All("/signout-callback-oidc", r.handlePacket)

	// hepsi için packet handler
	r.fiber.All("/", r.handlePacket)

	r.fiber.All("/test", r.handlePacket)
	r.fiber.All("/packet", r.handlePacket)

	r.fiber.All("/sitemap.xml", func(c fiber.Ctx) error {
		xml, err := sitemapRepo.GenerateSitemapIndex(GetBaseURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.SendString(xml)
	})

	r.fiber.All("/sitemap-posts.xml", func(c fiber.Ctx) error {
		xml, err := sitemapRepo.GeneratePostSitemap(c.Context(), GetBaseURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.Send(xml)
	})

	r.fiber.All("/sitemap-news.xml", func(c fiber.Ctx) error {
		xml, err := sitemapRepo.GenerateNewsSitemap(c.Context(), GetBaseURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.Send(xml)
	})

	r.fiber.All("/sitemap-pillars.xml", func(c fiber.Ctx) error {
		xml, err := sitemapRepo.GeneratePillarSitemap(c.Context(), GetBaseURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.Send(xml)
	})

	r.fiber.All("/sitemap-clusters.xml", func(c fiber.Ctx) error {
		xml, err := sitemapRepo.GenerateClusterSitemap(c.Context(), GetBaseURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		c.Type("xml")
		return c.Send(xml)
	})

	r.fiber.Get("/sitemap-images.xml", func(c fiber.Ctx) error {
		xmlData, err := sitemapRepo.GenerateImageSitemap(c.Context(), GetBaseURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		c.Type("xml")
		return c.Send(xmlData)
	})

	r.fiber.Get("/sitemap-videos.xml", func(c fiber.Ctx) error {
		xmlData, err := sitemapRepo.GenerateVideoSitemap(c.Context(), GetBaseURL(c))
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		c.Type("xml")
		return c.Send(xmlData)
	})
	return r
}

func GetBaseURL(c fiber.Ctx) string {
	scheme := "http"

	if c.Protocol() == "https" {
		scheme = "https"
	}

	return scheme + "://" + c.Hostname()
}

func (r *Router) handlePacket(c fiber.Ctx) error {
	var action string
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	c.Set("Access-Control-Allow-Headers", "Accept,Authorization,Content-Type,X-CSRF-Token,Token,session,Origin,Host,Connection,Accept-Encoding,Accept-Language,X-Requested-With")
	if c.Method() == fiber.MethodOptions {
		return c.SendStatus(fiber.StatusNoContent)
	}
	if c.Method() == fiber.MethodOptions {
		return c.SendStatus(fiber.StatusNoContent)
	}

	switch c.Method() {
	case fiber.MethodGet:
		action = c.Query("action")

	case fiber.MethodPost:
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

	default:
		return c.Status(fiber.StatusMethodNotAllowed).SendString("method not allowed")
	}

	if action == "" {
		fmt.Println("Default handler çalıştı")
		return c.SendString("Default handler executed")
	}

	route, ok := r.action.GetHandler(action)
	if !ok {
		return c.Status(fiber.StatusBadRequest).SendString("Unknown action")
	}

	// Middleware zincirini uygula (Fiber middleware olduğu varsayımıyla)
	handler := route.Handler
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	return handler(c)
}

func (r *Router) GetFiber() *fiber.App {
	return r.fiber
}
