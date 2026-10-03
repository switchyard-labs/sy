package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return s
}

func TestRateLimitContractAndCancellation(t *testing.T) {
	calls := 0
	s := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "20")
		w.WriteHeader(429)
		w.Write([]byte(`{"error":"upstream_rate_limited"}`))
	})
	c := New(s.URL, "tok")
	err := c.Do(t.Context(), "POST", "/mutation", nil)
	var he *HTTPError
	if !errors.As(err, &he) || he.Code != "upstream_rate_limited" || he.RetryAfter != "20" || !errors.Is(err, ErrUnavailable) || calls != 1 {
		t.Fatalf("mutation throttle contract: %v calls=%d", err, calls)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	err = c.Do(ctx, "GET", "/read", nil)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > time.Second {
		t.Fatalf("retry ignored cancellation: %v", err)
	}
}

func TestAuthDoesNotFollowRedirect(t *testing.T) {
	targetCalls := 0
	target := testServer(t, func(w http.ResponseWriter, r *http.Request) { targetCalls++; w.Write([]byte(`{}`)) })
	source := testServer(t, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) })
	err := New(source.URL, "private-session").Do(t.Context(), "GET", "/read", nil)
	if err == nil || targetCalls != 0 {
		t.Fatal("redirect followed")
	}
}

func TestErrorMapping(t *testing.T) {
	s := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"code":"unauthorized","message":"bad token"}}`))
	})
	c := New(s.URL, "tok")
	var out any
	err := c.Do(context.Background(), "GET", "/api/auth/me", &out)
	he, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if he.Status != 401 || !strings.Contains(he.Message, "bad token") {
		t.Fatalf("bad 401 mapping: %+v", he)
	}
	if err := he.Unwrap(); err != ErrUnauthorized {
		t.Fatalf("Unwrap = %v, want ErrUnauthorized", err)
	}
}

func TestErrorMappingConflictAndStale(t *testing.T) {
	s := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(409)
		w.Write([]byte(`{"error":{"code":"draft_stale","message":"draft moved on","current_revision":"13","expected_revision":"12"}}`))
	})
	c := New(s.URL, "tok")
	var out any
	err := c.Do(context.Background(), "PATCH", "/api/drafts/x", &out)
	he := err.(*HTTPError)
	if he.Extra["current_revision"] != "13" {
		t.Fatalf("extra not parsed: %+v", he.Extra)
	}
	if err := he.Unwrap(); err != ErrStale {
		t.Fatalf("Unwrap = %v, want ErrStale", err)
	}
}

func TestAuthCookieSent(t *testing.T) {
	var gotCookie string
	s := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"authed":true,"user":"alice"}`))
	})
	c := New(s.URL, "sess_abc")
	var out struct {
		Authed bool   `json:"authed"`
		User   string `json:"user"`
	}
	if err := c.Do(context.Background(), "GET", "/api/auth/me", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotCookie, "switchyard_session=sess_abc") {
		t.Fatalf("session cookie not sent: %q", gotCookie)
	}
	if !out.Authed || out.User != "alice" {
		t.Fatalf("decode failed: %+v", out)
	}
}

func TestLoginCapturesSessionCookie(t *testing.T) {
	s := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "switchyard_session", Value: "sess_from_server"})
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	})
	c := New(s.URL, "")
	tok, err := c.Login(context.Background(), s.URL, "alice", "pw")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "sess_from_server" {
		t.Fatalf("token = %q", tok)
	}
}

func TestNetworkErrorIsUnavailable(t *testing.T) {
	c := New("http://127.0.0.1:1", "tok") // nothing listens here
	var out any
	err := c.Do(context.Background(), "GET", "/api/x", &out)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "server unavailable") {
		t.Fatalf("want unavailable, got %v", err)
	}
}
