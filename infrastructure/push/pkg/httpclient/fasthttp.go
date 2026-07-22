package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
)

type FastHttpClient struct {
	client *fasthttp.Client
}

// FastHttp tell service to use fasthttp.Client. If client is nil, default client will be used.
func FastHttp(client *fasthttp.Client) *FastHttpClient {
	if client != nil {
		return &FastHttpClient{client: client}
	}

	return &FastHttpClient{
		client: &fasthttp.Client{},
	}
}

func (f *FastHttpClient) RequestDelivery(ctx context.Context, endpoint string, headers *Headers, body *bytes.Buffer) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(endpoint)

	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetContentType("application/octet-stream")
	req.Header.SetContentEncoding("aes128gcm")
	req.Header.SetContentLength(body.Len())
	req.Header.Set("Authorization", headers.Authorization)
	req.Header.Set("TTL", strconv.FormatInt(int64(headers.TTL/time.Second), 10))

	if headers.Urgency != "" {
		req.Header.Set("Urgency", headers.Urgency)
	}

	req.SetBody(body.Bytes())

	deadline, ok := ctx.Deadline()
	if !ok {
		return 0, fmt.Errorf("web push request requires a deadline")
	}
	timeout := time.Until(deadline)
	if timeout <= 0 {
		return 0, context.DeadlineExceeded
	}

	if err := f.client.DoTimeout(req, resp, timeout); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return resp.StatusCode(), ctxErr
		}
		return resp.StatusCode(), err
	}

	return resp.StatusCode(), nil
}
