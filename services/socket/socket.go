package socket

import (
	"coolvibes/helpers"
	userModel "coolvibes/models"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/cors"
	socketio "github.com/vchitai/go-socket.io/v4"
	"github.com/vchitai/go-socket.io/v4/engineio"
	"github.com/vchitai/go-socket.io/v4/engineio/transport"
	"github.com/vchitai/go-socket.io/v4/engineio/transport/polling"
	"github.com/vchitai/go-socket.io/v4/engineio/transport/websocket"
	"gorm.io/gorm"
)

var Server *socketio.Server
var userConnections = make(map[string]socketio.Conn)
var userPublicIDs = make(map[string]int64) // map[socketID]publicID

var allowOriginFunc = func(r *http.Request) bool {
	return true
}

func updateUserRooms(s socketio.Conn, db *gorm.DB, publicID int64, join bool) error {
	var chatIDs []uuid.UUID

	now := time.Now()

	updateData := map[string]interface{}{
		"last_online": now,
		"socket_id":   s.ID(),
	}
	result := db.Model(&userModel.User{}).Where("public_id = ?", publicID).Updates(updateData)
	if result.Error != nil {
		return result.Error
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

func ListenServer(db *gorm.DB) {

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
		log.Println("connected:", s.ID())
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
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return
		}

		tokenString := parts[1]

		claims, err := helpers.DecodeUserJWT(tokenString)
		if err != nil {
			return
		}

		userPublicIDs[s.ID()] = claims.PublicID
		updateUserRooms(s, db, claims.PublicID, true)

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

	Server.OnDisconnect("/", func(s socketio.Conn, reason string, m map[string]interface{}) {
		publicID, ok := userPublicIDs[s.ID()]
		if ok {
			updateUserRooms(s, db, publicID, false) // false = leave rooms
			delete(userPublicIDs, s.ID())
		}
		fmt.Println("Disconnected:", s.ID())
	})

	go func() {
		if err := Server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()
	defer Server.Close()

	mux := http.NewServeMux()

	mux.Handle("/socket.io/", Server)

	handler := cors.Default().Handler(mux)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	})

	handler = c.Handler(handler)
	log.Fatal(http.ListenAndServe(os.Getenv("SOCKET_PORT"), handler))

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

	now := time.Now()

	updateData := map[string]interface{}{
		"last_online": now,
		"socket_id":   conn.ID(),
	}

	result := s.db.Model(&userModel.User{}).Where("public_id = ?", publicID).Updates(updateData)
	if result.Error != nil {
		return result.Error
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
