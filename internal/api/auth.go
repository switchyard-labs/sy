package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Login authenticates with username/password and returns the session cookie
// token. Switchyard uses cookie sessions (no PAT/device flow yet); see
// docs/auth.md for the future PAT requirement.
func (c *Client) Login(ctx context.Context, baseURL, username, password string) (string, error) {
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar, Timeout: 30 * time.Second}
	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &HTTPError{Status: resp.StatusCode, Code: http.StatusText(resp.StatusCode)}
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == "switchyard_session" {
			return ck.Value, nil
		}
	}
	// fallback: read from jar
	for _, ck := range jar.Cookies(req.URL) {
		if ck.Name == "switchyard_session" {
			return ck.Value, nil
		}
	}
	return "", fmt.Errorf("login succeeded but no session cookie returned")
}

// Logout calls the logout endpoint (best-effort).
func (c *Client) Logout(ctx context.Context) error {
	return c.Do(ctx, http.MethodPost, "/api/auth/logout", nil)
}

// Me returns the authenticated user.
func (c *Client) Me(ctx context.Context) (string, error) {
	var out struct {
		Authed bool   `json:"authed"`
		User   string `json:"user"`
	}
	if err := c.Do(ctx, http.MethodGet, "/api/auth/me", &out); err != nil {
		return "", err
	}
	if !out.Authed {
		return "", ErrUnauthorized
	}
	return out.User, nil
}