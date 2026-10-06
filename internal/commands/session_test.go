package commands

import (
	"github.com/switchyard-labs/sy/internal/config"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHostSwitchAndLogoutSession(t *testing.T) {
	t.Setenv("SY_CONFIG_DIR", t.TempDir())
	calls := map[string]int{}
	server := func(user, token string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Cookie") != "switchyard_session="+token {
				t.Errorf("wrong host cookie: %q", r.Header.Get("Cookie"))
			}
			calls[user]++
			w.Write([]byte(`{"authed":true,"user":"` + user + `"}`))
		}))
	}
	a := server("a", "token-a")
	defer a.Close()
	b := server("b", "token-b")
	defer b.Close()
	cfg, _ := config.Load()
	cfg.SetHost(a.URL, config.Host{Token: "token-a", CookieName: "switchyard_session", User: "a"})
	cfg.SetHost(b.URL, config.Host{Token: "token-b", User: "b"})
	cfg.SetActive(a.URL)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		r := New()
		r.SetArgs(args)
		if err := r.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	run("auth", "status", "--json")
	run("auth", "switch", b.URL)
	run("auth", "status", "--json")
	run("auth", "logout")
	cfg, _ = config.Load()
	if cfg.ActiveHost != "" || len(cfg.Hosts) != 1 || cfg.Hosts[a.URL].Token != "token-a" {
		t.Fatalf("logout damaged another host: %+v", cfg)
	}
	if calls["a"] != 1 || calls["b"] != 2 {
		t.Fatalf("calls=%v", calls)
	}
	path, _ := config.Path()
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("config permissions changed")
	}
}
