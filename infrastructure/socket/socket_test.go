package socket

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestShouldSuppressSocketError(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		err  error
		want bool
	}{
		{
			name: "suppress ping timeout",
			msg:  "failed to get ping writer",
			err:  errors.New("write: timeout"),
			want: true,
		},
		{
			name: "suppress closed ping writer",
			msg:  "failed to close ping writer",
			err:  errors.New("use of closed network connection"),
			want: true,
		},
		{
			name: "keep unrelated error",
			msg:  "failed to get ping writer",
			err:  errors.New("permission denied"),
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldSuppressSocketError(tc.msg, tc.err); got != tc.want {
				t.Fatalf("shouldSuppressSocketError(%q, %v) = %v, want %v", tc.msg, tc.err, got, tc.want)
			}
		})
	}
}

func TestStartServerReportsBindFailureSynchronously(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer func() { _ = occupied.Close() }()

	runtime, err := StartServer(occupied.Addr().String(), nil, nil)
	if err == nil {
		if runtime != nil {
			_ = runtime.Shutdown(context.Background())
		}
		t.Fatal("StartServer() error = nil, want bind error")
	}
	if runtime != nil {
		t.Fatalf("StartServer() runtime = %#v, want nil", runtime)
	}
}

func TestRuntimeStartsAndShutsDown(t *testing.T) {
	runtime, err := StartServer("127.0.0.1:0", nil, nil)
	if err != nil {
		t.Fatalf("StartServer() error = %v", err)
	}
	addr := runtime.Addr()
	if addr == nil {
		t.Fatal("Runtime.Addr() = nil")
	}
	if runtime.SocketServer() == nil {
		t.Fatal("Runtime.SocketServer() = nil")
	}

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + addr.String() + "/")
	if err != nil {
		t.Fatalf("GET socket server: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("GET status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := runtime.Shutdown(ctx); err != nil {
		t.Fatalf("second Shutdown() error = %v", err)
	}

	select {
	case <-runtime.Done():
	default:
		t.Fatal("Runtime.Done() was not closed")
	}
	if server := currentSocketServer(); server != nil {
		t.Fatalf("global socket server = %#v, want nil", server)
	}

	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		t.Fatalf("Runtime.Addr() type = %T, want *net.TCPAddr", addr)
	}
	rebound, err := net.Listen("tcp", net.JoinHostPort(tcpAddr.IP.String(), strconv.Itoa(tcpAddr.Port)))
	if err != nil {
		t.Fatalf("listener was not released after shutdown: %v", err)
	}
	_ = rebound.Close()
}

func TestConnectionBookkeepingIsConcurrentSafe(t *testing.T) {
	clearConnections()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			socketID := strconv.Itoa(i)
			rememberPublicID(socketID, int64(i))
			_, _ = forgetConnection(socketID)
		}()
	}
	wg.Wait()

	connectionsMu.RLock()
	defer connectionsMu.RUnlock()
	if len(userPublicIDs) != 0 {
		t.Fatalf("public ID entries = %d, want 0", len(userPublicIDs))
	}
}

func TestRuntimeClosesHijackedConnections(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	runtime := &Runtime{hijackedConnections: make(map[net.Conn]struct{})}
	runtime.trackHTTPConnection(serverConn, http.StateHijacked)

	if err := runtime.closeHijackedConnections(); err != nil {
		t.Fatalf("closeHijackedConnections() error = %v", err)
	}
	_ = clientConn.SetReadDeadline(time.Now().Add(time.Second))
	buffer := make([]byte, 1)
	if _, err := clientConn.Read(buffer); err == nil {
		t.Fatal("peer remained readable after hijacked connection close")
	}
}
