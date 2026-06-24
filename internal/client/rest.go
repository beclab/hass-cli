package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// restClient calls the Home Assistant REST API under /api with a bearer token.
type restClient struct {
	baseURL string
	token   string
	http    *http.Client
}

func newRESTClient(baseURL, token string, insecure bool, timeout time.Duration) *restClient {
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &restClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: timeout, Transport: transport},
	}
}

// do performs an HTTP request against path (relative to /api) and returns the
// raw response body. A non-2xx status is returned as an error including body.
func (c *restClient) do(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}

	// baseURL already ends in /api; tolerate callers passing "/api/..." or
	// "api/..." so raw passthrough paths copied from docs still resolve.
	path = strings.TrimLeft(path, "/")
	path = strings.TrimPrefix(path, "api/")
	url := c.baseURL + "/" + path
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{Status: resp.StatusCode, Body: strings.TrimSpace(string(data))}
	}
	return json.RawMessage(data), nil
}

// HTTPError is returned for non-2xx REST responses.
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Body)
}
