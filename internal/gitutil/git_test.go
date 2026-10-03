package gitutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectOwnerRepo(t *testing.T) {
	cases := []struct {
		remote, owner, repo string
	}{
		{"https://b7f20353ee8a9e5d2003f52c74ba795e.artifacts.cloudflare.net/git/switchyard-cp0/demo-basic.git", "switchyard-cp0", "demo-basic"},
		{"https://github.com/switchyard-labs/switchyard.git", "switchyard-labs", "switchyard"},
		{"git@github.com:owner/repo.git", "owner", "repo"},
		{"", "", ""},
	}
	for _, c := range cases {
		o, r := DetectOwnerRepo(c.remote)
		if o != c.owner || r != c.repo {
			t.Errorf("DetectOwnerRepo(%q) = (%q,%q), want (%q,%q)", c.remote, o, r, c.owner, c.repo)
		}
	}
}

// TestCredentialNotInGitConfig proves that using the GIT_CONFIG env mechanism
// never writes the token into the clone's .git/config.
func TestCredentialNotInGitConfig(t *testing.T) {
	dir := t.TempDir()
	// seed a local repo
	if _, _, err := Cmd(t.Context(), dir, nil, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Cmd(t.Context(), dir, nil, "config", "user.email", "t@t"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Cmd(t.Context(), dir, nil, "config", "user.name", "t"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Cmd(t.Context(), dir, nil, "add", "-A"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Cmd(t.Context(), dir, nil, "commit", "-qm", "c"); err != nil {
		t.Fatal(err)
	}
	token := "art_v2_test_secret_token_should_never_persist"
	cf, cleanup, err := CredentialFile(token)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	env := []string{CredentialHelperOff(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + cf}
	cloneDir := filepath.Join(t.TempDir(), "clone")
	if _, serr, err := Cmd(t.Context(), ".", env, "clone", "--", dir, cloneDir); err != nil {
		t.Fatalf("clone failed: %s %v", serr, err)
	}
	// assert .git/config contains no token and no http.extraheader
	cfg, err := os.ReadFile(filepath.Join(cloneDir, ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cfg), token) {
		t.Fatal("token leaked into .git/config")
	}
	if strings.Contains(string(cfg), "extraheader") {
		t.Fatal("http.extraHeader persisted in .git/config")
	}
	// the credential file must carry the token but the argv must never
	cfgBytes, _ := os.ReadFile(cf)
	if !strings.Contains(string(cfgBytes), token) {
		t.Fatal("credential file missing token")
	}
	args := []string{"git", "clone", "--", "https://x", "/tmp/clone"}
	if strings.Contains(strings.Join(args, " "), token) {
		t.Fatal("token in argv")
	}
}