package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	c.SetHost("demo", Host{Token: "sess_x", User: "alice"})
	c.SetActive("demo")
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(filepath.Join(dir, "switchyard", "config.json"))
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config perms = %v, want 0600", info.Mode().Perm())
	}
	c2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	h, _ := c2.Host()
	if h == nil || h.Token != "sess_x" || h.User != "alice" {
		t.Fatalf("round trip failed: %+v", c2)
	}
}

func TestNormalizeHost(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"http://45.79.189.46/", "45.79.189.46"},
		{"https://demo.example.com", "demo.example.com"},
		{"demo.example.com", "demo.example.com"},
	} {
		if got := NormalizeHost(c.in); got != c.want {
			t.Errorf("NormalizeHost(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLoadEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.ActiveHost != "" || len(c.Hosts) != 0 {
		t.Fatalf("expected empty config, got %+v", c)
	}
}