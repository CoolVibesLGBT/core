package httpclient

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeliveryClientsHonorRequestDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(time.Second):
			w.WriteHeader(http.StatusCreated)
		}
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name   string
		client Client
	}{
		{name: "standard library", client: StdHttp(nil)},
		{name: "fasthttp", client: FastHttp(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
			defer cancel()
			started := time.Now()

			_, err := tt.client.RequestDelivery(
				ctx,
				server.URL,
				&Headers{TTL: time.Minute},
				bytes.NewBufferString("encrypted payload"),
			)
			elapsed := time.Since(started)

			if err == nil {
				t.Fatal("RequestDelivery() error = nil; want timeout")
			}
			if elapsed > 500*time.Millisecond {
				t.Fatalf("RequestDelivery() returned after %s; want a bounded request", elapsed)
			}
		})
	}
}
