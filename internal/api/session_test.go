package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionEnvironments(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(map[bool]string{false: "http", true: "https"}[secure], func(t *testing.T) {
			name := "switchyard_session"
			if secure {
				name = "__Host-switchyard_session"
			}
			calls := 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/auth/login" {
					http.SetCookie(w, &http.Cookie{Name: name, Value: "session-test", Path: "/", Secure: secure, HttpOnly: true})
					w.Write([]byte(`{}`))
					return
				}
				calls++
				if got := r.Header.Get("Cookie"); got != name+"=session-test" {
					t.Errorf("wrong/ambiguous cookie: %q", got)
				}
				w.Write([]byte(`{"authed":true,"user":"test"}`))
			})
			var s *httptest.Server
			if secure {
				s = httptest.NewTLSServer(handler)
			} else {
				s = httptest.NewServer(handler)
			}
			defer s.Close()
			c := New(s.URL, "")
			c.http = s.Client()
			token, err := c.Login(t.Context(), s.URL, "test", "password")
			if err != nil || token != "session-test" || c.CookieName != name {
				t.Fatalf("login: %q %s %v", token, c.CookieName, err)
			}
			if _, err = c.Me(t.Context()); err != nil {
				t.Fatal(err)
			}
			if _, _, err = c.DoRaw(t.Context(), "GET", "/api/auth/me", ""); err != nil {
				t.Fatal(err)
			}
			if err = c.Logout(t.Context()); err != nil {
				t.Fatal(err)
			}
			// Legacy config has only a token; use one environment-appropriate cookie.
			c.CookieName = ""
			if _, err = c.Me(t.Context()); err != nil {
				t.Fatal(err)
			}
			if calls != 4 {
				t.Fatalf("calls=%d", calls)
			}
		})
	}
}

func TestRejectAmbiguousAndInvalidSessionCookies(t *testing.T) {
	for _, cookies := range [][]*http.Cookie{
		{{Name: "switchyard_session", Value: "a"}, {Name: "__Host-switchyard_session", Value: "b", Path: "/", Secure: true}},
		{{Name: "__Host-switchyard_session", Value: "a", Path: "/", Secure: true}},
	} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, c := range cookies {
				http.SetCookie(w, c)
			}
		}))
		_, err := New(s.URL, "").Login(t.Context(), s.URL, "x", "p")
		s.Close()
		if err == nil {
			t.Fatal("invalid/ambiguous login accepted")
		}
	}
}

func TestSessionDoesNotCrossOriginOrDowngrade(t *testing.T) {
	c := New("https://configured.example", "private")
	c.CookieName = "__Host-switchyard_session"
	for _, url := range []string{"https://other.example/api/auth/me", "http://configured.example/api/auth/me"} {
		req, _ := http.NewRequest("GET", url, nil)
		if err := c.addSession(req); err == nil || req.Header.Get("Cookie") != "" {
			t.Fatal("credential escaped configured origin")
		}
	}
	c = New("http://configured.example", "private")
	c.CookieName = "__Host-switchyard_session"
	req, _ := http.NewRequest("GET", c.BaseURL+"/api/auth/me", nil)
	if c.addSession(req) == nil {
		t.Fatal("secure cookie sent over HTTP")
	}
}

func TestLoginDoesNotFollowRedirect(t *testing.T) {
	calls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer source.Close()
	if _, err := New(source.URL, "").Login(t.Context(), source.URL, "x", "secret"); err == nil || calls != 0 {
		t.Fatal("login credentials followed redirect")
	}
}
