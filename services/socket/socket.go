package socket

import (
	"core/constants"
	"core/helpers"
	userModel "core/models"
	"core/services/socket/managers"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/google/uuid"
	"github.com/google/wire"
	"github.com/rs/cors"
	socketio "github.com/vchitai/go-socket.io/v4"
	"github.com/vchitai/go-socket.io/v4/engineio"
	"github.com/vchitai/go-socket.io/v4/engineio/transport"
	"github.com/vchitai/go-socket.io/v4/engineio/transport/polling"
	"github.com/vchitai/go-socket.io/v4/engineio/transport/websocket"
	socketlogger "github.com/vchitai/go-socket.io/v4/logger"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(
	NewSocketService,
	managers.NewNotificationManager,
)

var Server *socketio.Server
var userConnections = make(map[string]socketio.Conn)
var userPublicIDs = make(map[string]int64) // map[socketID]publicID
var allowOriginFunc = func(r *http.Request) bool {
	return true
}

type socketLogFilter struct {
	inner logr.LogSink
}

func (s *socketLogFilter) Init(info logr.RuntimeInfo) {
	s.inner.Init(info)
}

func (s *socketLogFilter) Enabled(level int) bool {
	return s.inner.Enabled(level)
}

func (s *socketLogFilter) Info(level int, msg string, keysAndValues ...any) {
	s.inner.Info(level, msg, keysAndValues...)
}

func (s *socketLogFilter) Error(err error, msg string, keysAndValues ...any) {
	if shouldSuppressSocketError(msg, err) {
		return
	}
	s.inner.Error(err, msg, keysAndValues...)
}

func (s *socketLogFilter) WithValues(keysAndValues ...any) logr.LogSink {
	return &socketLogFilter{inner: s.inner.WithValues(keysAndValues...)}
}

func (s *socketLogFilter) WithName(name string) logr.LogSink {
	return &socketLogFilter{inner: s.inner.WithName(name)}
}

func shouldSuppressSocketError(msg string, err error) bool {
	if err == nil {
		return false
	}

	errText := strings.ToLower(err.Error())
	if msg == "failed to get ping writer" && (strings.Contains(errText, "timeout") || strings.Contains(errText, "closed network connection")) {
		return true
	}
	if msg == "failed to close ping writer" && strings.Contains(errText, "closed network connection") {
		return true
	}

	return false
}

func configureSocketLogger() {
	base := stdr.New(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile))
	socketlogger.ReplaceLogger(logr.New(&socketLogFilter{inner: base.GetSink()}))
}

func updateUserPresence(db *gorm.DB, socketID string, publicID int64, online bool) error {
	now := time.Now()
	updateData := map[string]any{
		"last_online": now,
		"is_online":   online,
	}
	query := db.Model(&userModel.User{}).Where("public_id = ?", publicID)

	if online {
		updateData["socket_id"] = socketID
	} else {
		updateData["socket_id"] = nil
		query = query.Where("socket_id = ?", socketID)
	}

	return query.Updates(updateData).Error
}

func updateUserRooms(s socketio.Conn, db *gorm.DB, publicID int64, join bool) error {
	var chatIDs []uuid.UUID

	if err := updateUserPresence(db, s.ID(), publicID, join); err != nil {
		return err
	}

	err := db.
		Table("chat_participants AS cp").
		Select("cp.chat_id").
		Joins("JOIN users u ON u.id = cp.user_id").
		Where("u.public_id = ?", publicID).
		Order("cp.id ASC").
		Scan(&chatIDs).Error

	if err != nil {
		return err
	}

	// İşlem fonksiyonu: Join veya Leave
	operation := s.Leave
	if join {
		operation = s.Join
	}

	for _, chatID := range chatIDs {
		operation(chatID.String())
	}

	operation("news")
	operation("notice")
	operation("broadcast")
	operation("system")

	return nil
}

func ListenServer(db *gorm.DB, notificationManager *managers.NotificationManager) (*socketio.Server, error) {
	configureSocketLogger()

	Server = socketio.NewServer(&engineio.Options{
		PingInterval: 25 * time.Second, // Sunucunun istemciye ping atma sıklığı
		PingTimeout:  90 * time.Second, // Maksimum bekleme süresi (cevap gelmezse bağlantıyı kopar)
		Transports: []transport.Transport{
			&polling.Transport{
				CheckOrigin: allowOriginFunc,
			},
			&websocket.Transport{
				CheckOrigin: allowOriginFunc,
			},
		},
	})

	Server.OnConnect("/", func(s socketio.Conn, m map[string]interface{}) error {
		fmt.Println("connected:", s.ID())
		userConnections[s.ID()] = s
		s.Emit("auth", s.ID())
		return nil
	})

	Server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		log.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	Server.OnEvent("/", "auth", func(s socketio.Conn, msg string) {
		authHeader := msg
		if authHeader == "" {
			fmt.Printf("Invalid Auth Header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			fmt.Printf("Invalid Auth Header")
			return
		}

		tokenString := parts[1]

		claims, err := helpers.DecodeUserJWT(tokenString)
		if err != nil {
			fmt.Printf("Invalid JWT Token:")
			return
		}

		userPublicIDs[s.ID()] = claims.PublicID
		err = updateUserRooms(s, db, claims.PublicID, true)
		if err != nil {
			fmt.Printf("Error updating user rooms: %v\n", err)
		}

	})

	Server.OnEvent("/", "join", func(s socketio.Conn, msg string) {
		fmt.Println("chatJoin:", msg)
		s.Emit("auth", "have "+msg)
	})

	Server.OnEvent("/", "init", func(s socketio.Conn, msg string) {
		fmt.Println("chatInit:", msg)
	})

	Server.OnEvent("/", "leave", func(s socketio.Conn, msg string) {
		fmt.Println("chatLeave:", msg)
	})

	Server.OnEvent("/", "notifications", func(s socketio.Conn, msg string) {

		type NotificationMessage struct {
			Action         string `json:"action"`
			Token          string `json:"token"`
			NotificationID string `json:"notification_id"`
		}

		var notificationMsg NotificationMessage
		err := json.Unmarshal([]byte(msg), &notificationMsg)
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			return
		}
		if notificationMsg.Action == constants.CMD_USER_MARK_NOTIFICATIONS_SEEN {
			err := notificationManager.MarkNotificationAsRead(notificationMsg.NotificationID)
			if err != nil {
				fmt.Println("Error marking notification as read:", err)
			}
		}

	})

	Server.OnDisconnect("/", func(s socketio.Conn, reason string, m map[string]interface{}) {
		delete(userConnections, s.ID())
		publicID, ok := userPublicIDs[s.ID()]
		if ok {
			err := updateUserRooms(s, db, publicID, false) // false = leave rooms
			if err != nil {
				fmt.Printf("Error updating user rooms: %v\n", err)
			}
			delete(userPublicIDs, s.ID())
		}
		fmt.Printf("Disconnected: %s reason=%s\n", s.ID(), reason)
	})

	go func() {
		if err := Server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer func() {
		if err := Server.Close(); err != nil {
			log.Printf("socketio close error: %v\n", err)
		}
	}()

	mux := http.NewServeMux()

	mux.Handle("/socket.io/", Server)

	handler := cors.Default().Handler(mux)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	handler = c.Handler(handler)
	log.Fatal(http.ListenAndServe(os.Getenv("SOCKET_PORT"), handler))

	return Server, nil

}

type SocketService struct {
	db *gorm.DB
}

func NewSocketService(db *gorm.DB) *SocketService {
	return &SocketService{db: db}
}

func (socketService *SocketService) BroadcastToRoom(namespace string, room string, event string, msg string) error {
	Server.BroadcastToRoom(namespace, room, event, msg)

	return nil
}

func (socketService *SocketService) BroadcastToNamespace(namespace string, event string, msg string) bool {
	return Server.BroadcastToNamespace(namespace, event, msg)

}

func (socketService *SocketService) SendMessageToUser(userId uuid.UUID, event string, message string) error {
	/*
		userRepo := &db.UserRepositoryImpl{DB: repo.DB}
		user, err := userRepo.GetUser(&models.User{ID: userID})
		if err != nil {
			return errors.New("User not found")
		}
		if conn, ok := userConnections[*user.SocketID]; ok {
			conn.Emit(event, message)
			return nil
		}*/
	return nil
}

func (s *SocketService) UpdateUserRooms(conn socketio.Conn, publicID int64, join bool) error {
	var chatIDs []uuid.UUID

	if err := updateUserPresence(s.db, conn.ID(), publicID, join); err != nil {
		return err
	}

	err := s.db.
		Table("chat_participants AS cp").
		Select("cp.chat_id").
		Joins("JOIN users u ON u.id = cp.user_id").
		Where("u.public_id = ?", publicID).
		Order("cp.id ASC").
		Scan(&chatIDs).Error

	if err != nil {
		return err
	}

	operation := conn.Leave
	if join {
		operation = conn.Join
	}

	for _, chatID := range chatIDs {
		operation(chatID.String())
	}

	operation("news")
	operation("notice")
	operation("broadcast")
	operation("system")

	return nil
}
