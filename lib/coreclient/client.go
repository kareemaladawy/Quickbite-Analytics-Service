// Package coreclient is the Go analogue of order-service's
// lib/core-client/core-client.ts: a synchronous HTTP client to core-service,
// authenticated with the shared-secret `api-key` header.
package coreclient

import (
	"context"
	"encoding/json"
	"fmt"

	"quickbite/analytics-service/pkg/httpclient"
)

type Client struct {
	http    *httpclient.Client
	baseURL string
	apiKey  string
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		http:    httpclient.New(httpclient.Config{Retries: 2}),
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	resp, err := c.http.Do(ctx, httpclient.Request{
		Method: "GET",
		URL:    c.baseURL + path,
		Headers: map[string]string{
			"api-key": c.apiKey,
		},
	})
	if err != nil {
		return fmt.Errorf("coreclient: request to %s failed: %w", path, err)
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("coreclient: %s returned status %d: %s", path, resp.StatusCode, string(resp.Body))
	}
	return json.Unmarshal(resp.Body, out)
}
