package httpclient

import (
	"bytes"
	"time"
)

type Headers struct {
	Authorization string
	Urgency       string
	TTL           time.Duration
}

type Client interface {
	RequestDelivery(endpoint string, headers *Headers, body *bytes.Buffer) (int, error)
}
