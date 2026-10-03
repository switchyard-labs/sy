// Package api is the Switchyard HTTP API client. It centralizes auth, JSON,
// error mapping, timeouts, and context handling so commands stay thin and
// deterministic (the CLI is also an API dogfood client).
package api

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
	"time"
)

// Typed errors for stable exit-code mapping and agent-readable messages.
var (
	ErrUnauthorized = errors.New("authentication required or failed")
	ErrForbidden    = errors.New("not authorized for this operation")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrStale        = errors.New("stale state")
	ErrUnavailable  = errors.New("server unavailable")
	ErrInvalid      = errors.New("invalid request")
)

type HTTPError struct {
	Status   int    `json:"-"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	// Extra carries structured fields (e.g. current_revision) for stale/conflict.
	Extra map[string]any `json:"extra,omitempty"`
}

func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

// Classify maps an HTTPError to a typed sentinel.
func (e *HTTPError) Unwrap() error {
	switch {
	case e.Status == 401:
		return ErrUnauthorized
	case e.Status == 403:
		return ErrForbidden
	case e.Status == 404:
		return ErrNotFound
	case e.Status == 409:
		if strings.Contains(e.Code, "stale") || strings.Contains(e.Message, "stale") || strings.Contains(e.Message, "revision") {
			return ErrStale
		}
		return ErrConflict
	case e.Status >= 500:
		return ErrUnavailable
	case e.Status >= 400:
		return ErrInvalid
	}
	return nil
}

type Client struct {
	BaseURL   string
	Token     string // session cookie value (host token)
	UserAgent string
	http      *http.Client
}

func New(baseURL, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL:   baseURL,
		Token:     token,
		UserAgent: "sy/dev",
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type requestOptions struct {
	Method string
	Path   string
	Body   any
	Query  url.Values
	Header http.Header
}

type RequestOption func(*requestOptions)

func WithBody(v any) RequestOption          { return func(o *requestOptions) { o.Body = v } }
func WithQuery(k, v string) RequestOption   { return func(o *requestOptions) { if o.Query == nil { o.Query = url.Values{} }; o.Query.Set(k, v) } }
func WithHeader(k, v string) RequestOption  { return func(o *requestOptions) { if o.Header == nil { o.Header = http.Header{} }; o.Header.Set(k, v) } }

// Do performs an authenticated request and decodes a JSON envelope into out.
// Non-2xx responses become *HTTPError.
func (c *Client) Do(ctx context.Context, method, path string, out any, opts ...RequestOption) error {
	ro := &requestOptions{Method: method, Path: path}
	for _, o := range opts {
		o(ro)
	}
	u := c.BaseURL + ro.Path
	if len(ro.Query) > 0 {
		u += "?" + ro.Query.Encode()
	}
	var body io.Reader
	if ro.Body != nil {
		b, err := json.Marshal(ro.Body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, ro.Method, u, body)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if ro.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Cookie", "switchyard_session="+c.Token)
	}
	for k, vs := range ro.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		he := &HTTPError{Status: resp.StatusCode, Extra: map[string]any{}}
		var env struct {
			Error map[string]any `json:"error"`
		}
		if json.Unmarshal(b, &env) == nil && env.Error != nil {
			he.Code, _ = env.Error["code"].(string)
			he.Message, _ = env.Error["message"].(string)
			he.RequestID, _ = env.Error["requestId"].(string)
			for k, v := range env.Error {
				if k != "code" && k != "message" && k != "requestId" {
					he.Extra[k] = v
				}
			}
		}
		if he.Code == "" {
			he.Code = http.StatusText(resp.StatusCode)
		}
		return he
	}
	if out != nil && len(b) > 0 {
		if err := json.Unmarshal(b, out); err != nil {
			return fmt.Errorf("decode %s: %w", ro.Path, err)
		}
	}
	return nil
}

// DoRaw performs a request and returns the raw body (for sy api / streaming).
func (c *Client) DoRaw(ctx context.Context, method, path string, body string) ([]byte, int, error) {
	ro := &requestOptions{Method: method, Path: path}
	if body != "" {
		ro.Body = json.RawMessage(body)
	}
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, rd)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Cookie", "switchyard_session="+c.Token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	return b, resp.StatusCode, nil
}