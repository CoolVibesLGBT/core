package socket

import (
	"context"
	"core/application/ports"
	"core/helpers"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const socketTestJWTSecret = "socket-test-user-jwt-secret-with-at-least-32-bytes"

type fakeSocketConn struct {
	id    string
	ctx   context.Context
	mu    sync.Mutex
	rooms map[string]struct{}
	emits []string
}

func newFakeSocketConn(id string) *fakeSocketConn {
	return &fakeSocketConn{id: id, ctx: context.Background(), rooms: make(map[string]struct{})}
}

func (*fakeSocketConn) Close() error                     { return nil }
func (c *fakeSocketConn) ID() string                     { return c.id }
func (*fakeSocketConn) URL() url.URL                     { return url.URL{} }
func (*fakeSocketConn) LocalAddr() net.Addr              { return nil }
func (*fakeSocketConn) RemoteAddr() net.Addr             { return nil }
func (*fakeSocketConn) RemoteHeader() http.Header        { return http.Header{} }
func (*fakeSocketConn) Serve()                           {}
func (c *fakeSocketConn) Context() context.Context       { return c.ctx }
func (c *fakeSocketConn) SetContext(ctx context.Context) { c.ctx = ctx }
func (*fakeSocketConn) Namespace() string                { return "/" }
func (c *fakeSocketConn) Emit(eventName string, _ ...interface{}) {
	c.mu.Lock()
	c.emits = append(c.emits, eventName)
	c.mu.Unlock()
}
func (c *fakeSocketConn) Join(room string) {
	c.mu.Lock()
	c.rooms[room] = struct{}{}
	c.mu.Unlock()
}
func (c *fakeSocketConn) Leave(room string) {
	c.mu.Lock()
	delete(c.rooms, room)
	c.mu.Unlock()
}
func (c *fakeSocketConn) LeaveAll() {
	c.mu.Lock()
	c.rooms = make(map[string]struct{})
	c.mu.Unlock()
}
func (c *fakeSocketConn) Rooms() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	rooms := make([]string, 0, len(c.rooms))
	for room := range c.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}
func (*fakeSocketConn) Refuse(error) error { return nil }

func (c *fakeSocketConn) hasRoom(room string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.rooms[room]
	return ok
}

func TestSocketAuthenticationAcknowledgesAndIsolatesReauthentication(t *testing.T) {
	t.Setenv("USER_JWT_SECRET", socketTestJWTSecret)
	clearConnections()
	defer clearConnections()

	firstToken, err := helpers.GenerateUserJWT(uuid.New(), 101)
	if err != nil {
		t.Fatalf("GenerateUserJWT(first) error = %v", err)
	}
	secondToken, err := helpers.GenerateUserJWT(uuid.New(), 202)
	if err != nil {
		t.Fatalf("GenerateUserJWT(second) error = %v", err)
	}

	conn := newFakeSocketConn("socket-1")
	conn.Join("stale-room")
	firstAck := authenticateSocketConnection(conn, nil, firstToken)
	if !firstAck.Success || firstAck.Error != "" {
		t.Fatalf("first auth ack = %+v", firstAck)
	}
	if !conn.hasRoom("user:101") || conn.hasRoom("stale-room") {
		t.Fatalf("first auth rooms = %v", conn.Rooms())
	}

	secondAck := authenticateSocketConnection(conn, nil, secondToken)
	if !secondAck.Success || secondAck.Error != "" {
		t.Fatalf("second auth ack = %+v", secondAck)
	}
	if !conn.hasRoom("user:202") || conn.hasRoom("user:101") {
		t.Fatalf("reauth rooms were not isolated: %v", conn.Rooms())
	}
	connectionsMu.RLock()
	rememberedPublicID := userPublicIDs[conn.ID()]
	connectionsMu.RUnlock()
	if rememberedPublicID != 202 {
		t.Fatalf("remembered public ID = %d, want 202", rememberedPublicID)
	}

	invalidAck := authenticateSocketConnection(conn, nil, "Bearer invalid")
	if invalidAck.Success || invalidAck.Error != "invalid_session" {
		t.Fatalf("invalid auth ack = %+v", invalidAck)
	}
	if len(conn.Rooms()) != 0 {
		t.Fatalf("invalid reauth retained rooms: %v", conn.Rooms())
	}
	connectionsMu.RLock()
	_, stillAuthenticated := userPublicIDs[conn.ID()]
	connectionsMu.RUnlock()
	if stillAuthenticated {
		t.Fatal("invalid reauth retained authenticated identity")
	}
}

func TestActiveChatRoomQueryExcludesLeftParticipants(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=test password=test dbname=test sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	var chatIDs []struct {
		ChatID uuid.UUID
	}
	query := activeChatIDsQuery(db, 101).Find(&chatIDs)
	if query.Error != nil {
		t.Fatalf("activeChatIDsQuery() error = %v", query.Error)
	}
	sql := strings.ToLower(query.Statement.SQL.String())
	for _, fragment := range []string{"cp.left_at is null", "u.deleted_at is null", "u.public_id"} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("active chat query missing %q: %s", fragment, query.Statement.SQL.String())
		}
	}
}

func TestPrivatePhotoRealtimeEncodingMatchesVersionedClientContract(t *testing.T) {
	event := ports.PrivatePhotoRealtimeEnvelope{
		Version:    ports.PrivatePhotoRealtimeVersion,
		EventID:    uuid.NewString(),
		Type:       ports.PrivatePhotoEventMediaProcessingUpdated,
		OccurredAt: time.Now().UTC(),
		Data: ports.PrivatePhotoRealtimeEventData{
			OwnerID: "10",
			PhotoID: "20",
			Status:  "ready",
		},
	}
	payload, err := encodePrivatePhotoRealtimeEvent(event)
	if err != nil {
		t.Fatalf("encodePrivatePhotoRealtimeEvent() error = %v", err)
	}
	for _, required := range []string{`"version":"1"`, `"event_id":`, `"type":"private_photos.media.processing_updated"`, `"owner_id":"10"`} {
		if !strings.Contains(payload, required) {
			t.Fatalf("encoded event missing %q: %s", required, payload)
		}
	}
	for _, forbidden := range []string{"storage_path", `"url"`, "owner_uuid", "viewer_uuid"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("encoded event leaked %q: %s", forbidden, payload)
		}
	}

	event.Data.OwnerID = "https://example.invalid/private.jpg"
	if _, err := encodePrivatePhotoRealtimeEvent(event); err == nil {
		t.Fatal("encoder accepted a non-public-ID owner field")
	}
}
