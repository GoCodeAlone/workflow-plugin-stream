package mediamtx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client is a thin MediaMTX Control API client.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// PathConfig is the subset of MediaMTX PathConf this plugin manages.
type PathConfig struct {
	Source            string `json:"source,omitempty"`
	Record            bool   `json:"record,omitempty"`
	OverridePublisher bool   `json:"overridePublisher,omitempty"`
}

// Path is the subset of MediaMTX runtime path state used by stream sessions.
type Path struct {
	Name          string `json:"name"`
	Available     bool   `json:"available"`
	InboundBytes  uint64 `json:"inboundBytes"`
	OutboundBytes uint64 `json:"outboundBytes"`
}

// StartOptions controls the managed path created by Start.
type StartOptions struct {
	Record bool
}

// APIError is returned for non-2xx MediaMTX API responses.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("mediamtx API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("mediamtx API returned HTTP %d: %s", e.StatusCode, e.Message)
}

// NewClient returns a Control API client rooted at a MediaMTX base URL.
func NewClient(baseURL string) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("base URL must include scheme and host")
	}
	return &Client{baseURL: parsed, httpClient: http.DefaultClient}, nil
}

// AddPath creates a MediaMTX path configuration.
func (c *Client) AddPath(ctx context.Context, name string, config PathConfig) error {
	return c.do(ctx, http.MethodPost, "/v3/config/paths/add/"+escapePathName(name), config, nil)
}

// RemovePath deletes a MediaMTX path configuration.
func (c *Client) RemovePath(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodDelete, "/v3/config/paths/delete/"+escapePathName(name), nil, nil)
}

// ListPaths returns runtime path state from the MediaMTX Control API.
func (c *Client) ListPaths(ctx context.Context) ([]Path, error) {
	var result pathList
	if err := c.do(ctx, http.MethodGet, "/v3/paths/list?page=0&itemsPerPage=100", nil, &result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Start creates the MediaMTX publisher path for a stream session.
func (c *Client) Start(ctx context.Context, name string, opts StartOptions) error {
	return c.AddPath(ctx, name, PathConfig{
		Source:            "publisher",
		Record:            opts.Record,
		OverridePublisher: true,
	})
}

// Stop removes the MediaMTX publisher path for a stream session.
func (c *Client) Stop(ctx context.Context, name string) error {
	return c.RemovePath(ctx, name)
}

type pathList struct {
	Items []Path `json:"items"`
}

type apiErrorResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func (c *Client) do(ctx context.Context, method, apiPath string, body any, out any) error {
	if c == nil || c.baseURL == nil {
		return errors.New("client is not initialized")
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: apiPath})
	if strings.Contains(apiPath, "?") {
		pathPart, rawQuery, _ := strings.Cut(apiPath, "?")
		endpoint = c.baseURL.ResolveReference(&url.URL{Path: pathPart, RawQuery: rawQuery})
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody apiErrorResponse
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return &APIError{StatusCode: resp.StatusCode, Message: errBody.Error}
	}
	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func escapePathName(name string) string {
	segments := strings.Split(name, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
