package telegrammedia

import (
	"context"
	"core/application/ports"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGIFBridgeValidatesTransportBoundary(t *testing.T) {
	directory, err := os.MkdirTemp("/tmp", "cv-gif-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("remove socket directory: %v", err)
		}
	})
	socket := filepath.Join(directory, "media.sock")
	t.Setenv("TELEGRAM_MEDIA_SOCKET", socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	address := "/static/telegram/gifs/" + strings.Repeat("a", 64) + ".mp4"
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/prepare/"+strings.Repeat("a", 64)+".mp4" {
			if _, err := w.Write([]byte(`{"ready":true}`)); err != nil {
				t.Errorf("write prepare response: %v", err)
			}
			return
		}
		if r.URL.Query().Get("query") != "hello & 👋" || r.URL.Query().Get("offset") != "next+page" {
			t.Error("query/offset corrupted")
		}
		if _, err := fmt.Fprintf(w, `{"documents":[{"kind":"gif","id":"test","url":%q,"width":320,"height":240}],"nextOffset":"last"}`, address); err != nil {
			t.Errorf("write search response: %v", err)
		}
	})}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("serve GIF bridge: %v", err)
		}
	}()
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("close GIF bridge: %v", err)
		}
	})
	client := NewClient()
	page, err := client.SearchGIFs(context.Background(), "hello & 👋", "next+page")
	if err != nil || len(page.Documents) != 1 || page.NextOffset != "last" {
		t.Fatalf("GIF page failed: %+v %v", page, err)
	}
	if err := client.PrepareGIF(context.Background(), strings.Repeat("a", 64)+".mp4"); err != nil {
		t.Fatal(err)
	}
	if err := client.PrepareGIF(context.Background(), "../../.env"); err == nil {
		t.Fatal("path traversal accepted")
	}
	address = "https://untrusted.example/a.mp4"
	if _, err := client.SearchGIFs(context.Background(), "hello & 👋", "next+page"); err != ports.ErrChatGIFUnavailable {
		t.Fatal("untrusted media escaped transport validation")
	}
	if err := server.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SearchGIFs(context.Background(), "hello & 👋", "next+page"); err != ports.ErrChatGIFUnavailable {
		t.Fatal("missing session is not an availability failure")
	}
}
