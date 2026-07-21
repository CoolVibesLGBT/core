package socket

import (
	"context"
	"core/constants"
	"core/helpers"
	"core/infrastructure/socket/managers"
	userModel "core/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"github.com/google/uuid"
	"github.com/rs/cors"
	socketio "github.com/vchitai/go-socket.io/v4"
	"github.com/vchitai/go-socket.io/v4/engineio"
	"github.com/vchitai/go-socket.io/v4/engineio/transport"
	"github.com/vchitai/go-socket.io/v4/engineio/transport/polling"
	"github.com/vchitai/go-socket.io/v4/engineio/transport/websocket"
	socketlogger "github.com/vchitai/go-socket.io/v4/logger"
	"gorm.io/gorm"
)

var Server *socketio.Server
var userConnections = make(map[string]socketio.Conn)
var userPublicIDs = make(map[string]int64) // map[socketID]publicID
var serverMu sync.RWMutex
var connectionsMu sync.RWMutex
var allowOriginFunc = func(r *http.Request) bool {
	return true
}

type Runtime struct {
	socketServer        *socketio.Server
	httpServer          *http.Server
	listener            net.Listener
	errors              chan error
	done                chan struct{}
	wg                  sync.WaitGroup
	shutdownOnce        sync.Once
	shutdownErr         error
	hijackedMu          sync.Mutex
	hijackedConnections map[net.Conn]struct{}
}

func (r *Runtime) SocketServer() *socketio.Server {
	if r == nil {
		return nil
	}
	return r.socketServer
}

func (r *Runtime) Addr() net.Addr {
	if r == nil || r.listener == nil {
		return nil
	}
	return r.listener.Addr()
}

func (r *Runtime) Errors() <-chan error {
	if r == nil {
		return nil
	}
	return r.errors
}

func (r *Runtime) Done() <-chan struct{} {
	if r == nil {
		return nil
	}
	return r.done
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

func rememberConnection(conn socketio.Conn) {
	connectionsMu.Lock()
	userConnections[conn.ID()] = conn
	connectionsMu.Unlock()
}

func rememberPublicID(socketID string, publicID int64) {
	connectionsMu.Lock()
	userPublicIDs[socketID] = publicID
	connectionsMu.Unlock()
}

func forgetConnection(socketID string) (int64, bool) {
	connectionsMu.Lock()
	defer connectionsMu.Unlock()

	delete(userConnections, socketID)
	publicID, ok := userPublicIDs[socketID]
	delete(userPublicIDs, socketID)
	return publicID, ok
}

func snapshotConnections() []socketio.Conn {
	connectionsMu.RLock()
	defer connectionsMu.RUnlock()

	connections := make([]socketio.Conn, 0, len(userConnections))
	for _, conn := range userConnections {
		connections = append(connections, conn)
	}
	return connections
}

func clearConnections() {
	connectionsMu.Lock()
	userConnections = make(map[string]socketio.Conn)
	userPublicIDs = make(map[string]int64)
	connectionsMu.Unlock()
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
	if err := updateUserPresence(db, s.ID(), publicID, join); err != nil {
		return err
	}
	// go-socket.io removes a disconnecting connection from every room before
	// invoking OnDisconnect, so querying every chat merely to leave again is
	// redundant and makes shutdown proportional to each user's chat count.
	if !join {
		return nil
	}

	var chatIDs []uuid.UUID

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

	for _, chatID := range chatIDs {
		s.Join(chatID.String())
	}

	s.Join("news")
	s.Join("notice")
	s.Join("broadcast")
	s.Join("system")

	return nil
}

func newSocketServer(db *gorm.DB, notificationManager *managers.NotificationManager) *socketio.Server {
	server := socketio.NewServer(&engineio.Options{
		PingInterval: 25 * time.Second,
		PingTimeout:  90 * time.Second,
		Transports: []transport.Transport{
			&polling.Transport{CheckOrigin: allowOriginFunc},
			&websocket.Transport{CheckOrigin: allowOriginFunc},
		},
	})

	server.OnConnect("/", func(s socketio.Conn, _ map[string]interface{}) error {
		fmt.Println("connected:", s.ID())
		rememberConnection(s)
		s.Emit("auth", s.ID())
		return nil
	})

	server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
		log.Println("notice:", msg)
		s.Emit("reply", "have "+msg)
	})

	server.OnEvent("/", "auth", func(s socketio.Conn, msg string) {
		if msg == "" {
			fmt.Print("Invalid Auth Header")
			return
		}

		parts := strings.SplitN(msg, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			fmt.Print("Invalid Auth Header")
			return
		}

		claims, err := helpers.DecodeUserJWT(parts[1])
		if err != nil {
			fmt.Print("Invalid JWT Token:")
			return
		}

		rememberPublicID(s.ID(), claims.PublicID)
		if db != nil {
			if err := updateUserRooms(s, db, claims.PublicID, true); err != nil {
				fmt.Printf("Error updating user rooms: %v\n", err)
			}
		}
	})

	server.OnEvent("/", "join", func(s socketio.Conn, msg string) {
		fmt.Println("chatJoin:", msg)
		s.Emit("auth", "have "+msg)
	})
	server.OnEvent("/", "init", func(_ socketio.Conn, msg string) {
		fmt.Println("chatInit:", msg)
	})
	server.OnEvent("/", "leave", func(_ socketio.Conn, msg string) {
		fmt.Println("chatLeave:", msg)
	})

	server.OnEvent("/", "notifications", func(_ socketio.Conn, msg string) {
		type notificationMessage struct {
			Action         string `json:"action"`
			Token          string `json:"token"`
			NotificationID string `json:"notification_id"`
		}

		var notification notificationMessage
		if err := json.Unmarshal([]byte(msg), &notification); err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			return
		}
		if notification.Action == constants.CMD_USER_MARK_NOTIFICATIONS_SEEN && notificationManager != nil {
			if err := notificationManager.MarkNotificationAsRead(notification.NotificationID); err != nil {
				fmt.Println("Error marking notification as read:", err)
			}
		}
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string, _ map[string]interface{}) {
		publicID, ok := forgetConnection(s.ID())
		if ok && db != nil {
			if err := updateUserRooms(s, db, publicID, false); err != nil {
				fmt.Printf("Error updating user rooms: %v\n", err)
			}
		}
		fmt.Printf("Disconnected: %s reason=%s\n", s.ID(), reason)
	})

	return server
}

func StartServer(addr string, db *gorm.DB, notificationManager *managers.NotificationManager) (*Runtime, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, errors.New("socket listen address is required")
	}

	serverMu.Lock()
	if Server != nil {
		serverMu.Unlock()
		return nil, errors.New("socket server is already running")
	}

	configureSocketLogger()
	server := newSocketServer(db, notificationManager)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		serverMu.Unlock()
		_ = server.Close()
		return nil, fmt.Errorf("listen on socket address %q: %w", addr, err)
	}

	clearConnections()
	Server = server
	serverMu.Unlock()

	mux := http.NewServeMux()
	mux.Handle("/socket.io/", server)
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}).Handler(mux)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           corsHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	runtime := &Runtime{
		socketServer:        server,
		httpServer:          httpServer,
		listener:            listener,
		errors:              make(chan error, 2),
		done:                make(chan struct{}),
		hijackedConnections: make(map[net.Conn]struct{}),
	}
	httpServer.ConnState = runtime.trackHTTPConnection
	runtime.start()
	return runtime, nil
}

func (r *Runtime) trackHTTPConnection(conn net.Conn, state http.ConnState) {
	if r == nil || conn == nil {
		return
	}
	r.hijackedMu.Lock()
	defer r.hijackedMu.Unlock()
	switch state {
	case http.StateHijacked:
		r.hijackedConnections[conn] = struct{}{}
	case http.StateClosed:
		delete(r.hijackedConnections, conn)
	}
}

func (r *Runtime) closeHijackedConnections() error {
	if r == nil {
		return nil
	}
	r.hijackedMu.Lock()
	connections := make([]net.Conn, 0, len(r.hijackedConnections))
	for conn := range r.hijackedConnections {
		connections = append(connections, conn)
	}
	r.hijackedConnections = make(map[net.Conn]struct{})
	r.hijackedMu.Unlock()

	var result error
	for _, conn := range connections {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			result = errors.Join(result, err)
		}
	}
	return result
}

func (r *Runtime) start() {
	r.wg.Add(2)

	go func() {
		defer r.wg.Done()
		if err := r.socketServer.Serve(); err != nil && !errors.Is(err, io.EOF) {
			r.publishError(fmt.Errorf("serve socket.io: %w", err))
			_ = r.httpServer.Close()
		}
	}()

	go func() {
		defer r.wg.Done()
		if err := r.httpServer.Serve(r.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			r.publishError(fmt.Errorf("serve socket HTTP: %w", err))
			_ = r.socketServer.Close()
		}
	}()

	go func() {
		r.wg.Wait()
		close(r.done)
		close(r.errors)
	}()
}

func (r *Runtime) publishError(err error) {
	if err == nil {
		return
	}
	select {
	case r.errors <- err:
	default:
		log.Printf("socket server error: %v", err)
	}
}

func closeTrackedConnections(ctx context.Context) error {
	connections := snapshotConnections()
	if len(connections) == 0 {
		return nil
	}

	results := make(chan error, len(connections))
	for _, conn := range connections {
		go func(conn socketio.Conn) {
			results <- conn.Close()
		}(conn)
	}

	var closeErrors []error
	for range connections {
		select {
		case err := <-results:
			if err != nil {
				closeErrors = append(closeErrors, err)
			}
		case <-ctx.Done():
			return errors.Join(errors.Join(closeErrors...), ctx.Err())
		}
	}
	return errors.Join(closeErrors...)
}

func (r *Runtime) Shutdown(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	r.shutdownOnce.Do(func() {
		// Shutdown closes the listener before waiting for HTTP requests. Run it
		// concurrently so active long-poll and websocket clients can be closed
		// immediately instead of consuming the entire shutdown deadline.
		httpResult := make(chan error, 1)
		go func() {
			httpResult <- r.httpServer.Shutdown(ctx)
		}()

		connectionErr := closeTrackedConnections(ctx)
		hijackedErr := r.closeHijackedConnections()
		httpErr := <-httpResult
		remainingConnectionErr := closeTrackedConnections(ctx)
		remainingHijackedErr := r.closeHijackedConnections()
		socketErr := r.socketServer.Close()

		var waitErr error
		select {
		case <-r.done:
		case <-ctx.Done():
			waitErr = ctx.Err()
		}

		serverMu.Lock()
		if Server == r.socketServer {
			Server = nil
		}
		serverMu.Unlock()
		clearConnections()

		r.shutdownErr = errors.Join(httpErr, connectionErr, hijackedErr, remainingConnectionErr, remainingHijackedErr, socketErr, waitErr)
	})

	return r.shutdownErr
}

// ListenServer preserves the old blocking API while callers migrate to
// StartServer and Runtime.Shutdown.
func ListenServer(db *gorm.DB, notificationManager *managers.NotificationManager) (*socketio.Server, error) {
	runtime, err := StartServer(os.Getenv("SOCKET_PORT"), db, notificationManager)
	if err != nil {
		return nil, err
	}

	serveErr, ok := <-runtime.Errors()
	if !ok {
		return runtime.SocketServer(), nil
	}
	_ = runtime.Shutdown(context.Background())
	return runtime.SocketServer(), serveErr
}

type SocketService struct {
	db *gorm.DB
}

func NewSocketService(db *gorm.DB) *SocketService {
	return &SocketService{db: db}
}

func currentSocketServer() *socketio.Server {
	serverMu.RLock()
	defer serverMu.RUnlock()
	return Server
}

func (socketService *SocketService) BroadcastToRoom(namespace string, room string, event string, msg string) error {
	server := currentSocketServer()
	if server == nil {
		return fmt.Errorf("socket server is not initialized")
	}
	server.BroadcastToRoom(namespace, room, event, msg)

	return nil
}

func (socketService *SocketService) BroadcastToNamespace(namespace string, event string, msg string) bool {
	server := currentSocketServer()
	return server != nil && server.BroadcastToNamespace(namespace, event, msg)

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
	if err := updateUserPresence(s.db, conn.ID(), publicID, join); err != nil {
		return err
	}
	if !join {
		return nil
	}

	var chatIDs []uuid.UUID

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

	for _, chatID := range chatIDs {
		conn.Join(chatID.String())
	}

	conn.Join("news")
	conn.Join("notice")
	conn.Join("broadcast")
	conn.Join("system")

	return nil
}
