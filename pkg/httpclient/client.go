// Package httpclient is a thin net/http wrapper: timeout, JSON encode/decode,
// retry-on-5xx. No app types, no env — callers (lib/coreclient) supply those.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	http    *http.Client
	retries int
}

type Config struct {
	Timeout time.Duration
	Retries int // additional attempts after the first, on 5xx or transport error
}

func New(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Client{
		http:    &http.Client{Timeout: cfg.Timeout},
		retries: cfg.Retries,
	}
}

type Request struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    any // marshaled as JSON when non-nil
}

// Response is what Do returns: the status code and raw body, left to the
// caller to unmarshal into its own envelope type.
type Response struct {
	StatusCode int
	Body       []byte
}

func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		encoded, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("httpclient: marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	var lastErr error
	attempts := c.retries + 1
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff(attempt))
		}

		httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("httpclient: build request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}
		if req.Body != nil {
			// bodyReader was already consumed on a retry; re-seek.
			encoded, _ := json.Marshal(req.Body)
			httpReq.Body = io.NopCloser(bytes.NewReader(encoded))
		}

		resp, doErr := c.http.Do(httpReq)
		if doErr != nil {
			lastErr = doErr
			continue
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		if resp.StatusCode >= 500 && attempt < attempts-1 {
			lastErr = fmt.Errorf("httpclient: upstream status %d", resp.StatusCode)
			continue
		}

		return &Response{StatusCode: resp.StatusCode, Body: data}, nil
	}

	return nil, fmt.Errorf("httpclient: request failed after %d attempts: %w", attempts, lastErr)
}

func backoff(attempt int) time.Duration {
	base := 50 * time.Millisecond
	d := base << (attempt - 1)
	if d > 500*time.Millisecond {
		return 500 * time.Millisecond
	}
	return d
}
