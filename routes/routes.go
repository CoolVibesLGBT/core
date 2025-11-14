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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type Router struct {
	mux           *mux.Router
	action        *router.ActionRouter
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func NewRouter(db *gorm.DB, snowFlakeNode *helpers.Node) *Router {

	r := &Router{
		mux:           mux.NewRouter(),
		action:        router.NewActionRouter(db),
		db:            db,
		snowFlakeNode: snowFlakeNode,
	}

	r.mux.PathPrefix("/static/").
		Handler(http.StripPrefix("/static/",
			http.FileServer(http.Dir("./static")),
		))

	socketService := socket.NewSocketService(r.db)

	// repository ve service oluştur
	engagementRepo := repositories.NewEngagementRepository(r.db, socketService)
	userRepo := repositories.NewUserRepository(r.db, snowFlakeNode, engagementRepo)
	mediaRepo := repositories.NewMediaRepository(r.db, snowFlakeNode)
	postRepo := repositories.NewPostRepository(r.db, snowFlakeNode, mediaRepo, userRepo)
	matchesRepo := repositories.NewMatchesRepository(r.db, engagementRepo)
	chatRepo := repositories.NewChatRepository(r.db, snowFlakeNode, postRepo, socketService)
	notificationRepo := repositories.NewNotificationRepository(r.db, snowFlakeNode)

	userService := services.NewUserService(userRepo, postRepo, mediaRepo)
	postService := services.NewPostService(userRepo, postRepo, mediaRepo)
	matchesService := services.NewMatchService(userRepo, postRepo, mediaRepo, matchesRepo)
	chatService := services.NewChatService(userRepo, postRepo, mediaRepo, matchesRepo, chatRepo, notificationRepo)

	r.action.Register(constants.CMD_INITIAL_SYNC, handlers.HandleInitialSync(r.db)) // middleware yok

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

	r.action.Register(constants.CMD_POST_FETCH, handlers.HandleGetByID(postService))
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

	r.mux.HandleFunc("/", r.handlePacket)
	r.mux.HandleFunc("/test", r.handlePacket)

	// Tek packet endpoint
	r.mux.HandleFunc("/packet", r.handlePacket)
	return r
}

func (r *Router) handlePacket(w http.ResponseWriter, req *http.Request) {
	var action string
	switch req.Method {
	case http.MethodGet:
		// GET query parametrelerinden al
		action = req.URL.Query().Get("action")

	case http.MethodPost:
		contentType := req.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			// JSON body
			var packet struct {
				Action string `json:"action"`
			}
			if err := json.NewDecoder(req.Body).Decode(&packet); err != nil {
				http.Error(w, "invalid JSON body", http.StatusBadRequest)
				return
			}
			action = packet.Action
		} else {
			// Form / multipart
			if err := req.ParseMultipartForm(8192 << 20); err != nil {
				http.Error(w, "Could not parse form", http.StatusBadRequest)
				return
			}
			action = req.FormValue("action")
		}

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if action == "" {
		fmt.Println("Default handler çalıştı")
		w.Write([]byte("Default handler executed"))
		return
	}

	route, ok := r.action.GetHandler(action)
	if !ok {
		http.Error(w, "Unknown action", http.StatusBadRequest)
		return
	}

	// Middleware zincirini uygula
	handler := route.Handler
	for i := len(route.Middlewares) - 1; i >= 0; i-- {
		handler = route.Middlewares[i](handler)
	}

	// Handler çalıştır
	handler.ServeHTTP(w, req)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func (r *Router) GetMux() *mux.Router {
	return r.mux
}
