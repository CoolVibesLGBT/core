package httpclient

import (
	"bytes"
	"context"
	"time"
)

type Headers struct {
	Authorization string
	Urgency       string
	TTL           time.Duration
}

type Client interface {
	RequestDelivery(ctx context.Context, endpoint string, headers *Headers, body *bytes.Buffer) (int, error)
}
