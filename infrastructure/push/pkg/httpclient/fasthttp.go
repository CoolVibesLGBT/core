package httpclient

import (
	"bytes"
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

func (f *FastHttpClient) RequestDelivery(endpoint string, headers *Headers, body *bytes.Buffer) (int, error) {
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

	if err := f.client.Do(req, resp); err != nil {
		return resp.StatusCode(), err
	}

	return resp.StatusCode(), nil
}
