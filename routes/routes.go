// routes/router.go
package routes

import (
	"coolvibes/constants"
	"coolvibes/helpers"
	"coolvibes/middleware"
	"coolvibes/repositories"
	"coolvibes/router"
	"coolvibes/routes/handlers"
	"coolvibes/services/socket"
	services "coolvibes/services/user"
	"fmt"
	"strings"

	fiber "github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"gorm.io/gorm"
)

type Router struct {
	fiber         *fiber.App
	action        *router.ActionRouter
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func NewRouter(db *gorm.DB, snowFlakeNode *helpers.Node) *Router {

	r := &Router{
		action: router.NewActionRouter(db),
		db:     db,
		fiber: fiber.New(fiber.Config{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
		}),
		snowFlakeNode: snowFlakeNode,
	}

	r.fiber.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowCredentials: false,
		AllowMethods:     "POST,GET,OPTIONS,PUT,DELETE",
		AllowHeaders:     "Accept,Authorization,authorization,Content-Type,Content-Length,X-CSRF-Token,Token,session,Origin,Host,Connection,Accept-Encoding,Accept-Language,X-Requested-With",
	}))

	r.fiber.Static("/static", "./static")

	socketService := socket.NewSocketService(r.db)

	notificationRepo := repositories.NewNotificationRepository(r.db, snowFlakeNode)
	notificationService := services.NewNotificationsService(notificationRepo)
	// repository ve service oluştur
	engagementRepo := repositories.NewEngagementRepository(r.db)
	userRepo := repositories.NewUserRepository(r.db, snowFlakeNode, engagementRepo)
	mediaRepo := repositories.NewMediaRepository(r.db, snowFlakeNode)
	postRepo := repositories.NewPostRepository(r.db, snowFlakeNode, mediaRepo, userRepo, notificationRepo)
	matchesRepo := repositories.NewMatchesRepository(r.db, engagementRepo)

	chatRepo := repositories.NewChatRepository(r.db, snowFlakeNode, postRepo, userRepo, notificationRepo)

	userService := services.NewUserService(userRepo, postRepo, mediaRepo, engagementRepo, notificationRepo)
	postService := services.NewPostService(userRepo, postRepo, mediaRepo)
	matchesService := services.NewMatchService(userRepo, postRepo, mediaRepo, matchesRepo)
	chatService := services.NewChatService(socketService, userRepo, postRepo, mediaRepo, matchesRepo, chatRepo, notificationRepo)

	paymentsRepo := repositories.NewPaymentRepositoryy(db, snowFlakeNode, mediaRepo, userRepo, notificationRepo)

	paymentService := services.NewPaymentService(paymentsRepo, userRepo, postRepo, mediaRepo)
	r.action.Register(constants.CMD_INITIAL_SYNC, handlers.HandleInitialSync(r.db))         // middleware yok
	r.action.Register(constants.CMD_GET_VAPID_PUBLIC_KEY, handlers.HandleVapidGetKey(r.db)) // middleware yok vapid
	r.action.Register(constants.CMD_SET_VAPID_SUBSCRIBE, handlers.HandleVapidSubscribe(r.db), middleware.AuthMiddleware(userRepo))

	// Action register
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
	//	r.action.Register(constants.CMD_POST_CREATE, middleware.AuthMiddleware(userRepo) handlers.HandleCreate(postService))
	r.action.Register(
		constants.CMD_POST_CREATE,
		handlers.HandleCreate(postService),  // handler
		middleware.AuthMiddleware(userRepo), // middleware
	)

	r.action.Register(
		constants.CMD_POST_VOTE,
		handlers.HandleVote(postService),    // handler
		middleware.AuthMiddleware(userRepo), // middleware
	)

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

	r.fiber.Post("/webhook/gateway/stripe/thin", handlers.HandleStripeThin(paymentService))
	r.fiber.Post("/webhook/gateway/stripe/snapshot", handlers.HandleStripeSnapshot(paymentService))
	r.fiber.All("/signin-oidc", r.handlePacket)
	r.fiber.All("/signout-callback-oidc", r.handlePacket)

	// hepsi için packet handler
	r.fiber.All("/", r.handlePacket)

	r.fiber.All("/test", r.handlePacket)
	r.fiber.All("/packet", r.handlePacket)

	return r
}

func (r *Router) handlePacket(c *fiber.Ctx) error {
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
			if err := c.BodyParser(&packet); err != nil {
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
