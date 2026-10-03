// Package gitutil centralizes safe Git subprocess execution: argument arrays
// only (never shell strings), controlled environment, and helpers that never
// leak credentials into process lists or .git/config.
package gitutil

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
)

// Cmd runs git with an argument array. envExtras are KEY=VALUE additions.
func Cmd(ctx context.Context, dir string, envExtras []string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cmd.Env = append(cmd.Env, envExtras...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ExtraHeader returns a git env extra that supplies an Authorization header
// without putting the token in argv (visible in `ps`).
func ExtraHeader(token string) string {
	return "GIT_CONFIG_COUNT=1\x00GIT_CONFIG_KEY_0=http.extraheader\x00GIT_CONFIG_VALUE_0=Authorization: Bearer " + token
}

// CredentialHelperOff disables the global credential helper so tokens never
// land in the user's credential store.
func CredentialHelperOff() string { return "GIT_TERMINAL_PROMPT=0" }

// DetectOwnerRepo derives owner/repo from a git remote URL. Supports:
//   https://<account>.artifacts.cloudflare.net/git/<ns>/<repo>.git
//   https://github.com/owner/repo.git
//   git@host:owner/repo.git
// Returns ("","") when it cannot be determined.
func DetectOwnerRepo(remote string) (string, string) {
	s := strings.TrimSpace(remote)
	s = strings.TrimSuffix(s, ".git")
	// scp-style: user@host:path
	if i := strings.Index(s, ":"); i >= 0 && !strings.Contains(s, "://") {
		s = s[i+1:]
	}
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "git@")
	if i := strings.Index(s, "/"); i >= 0 {
		rest := s[i+1:]
		parts := strings.Split(rest, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2], parts[len(parts)-1]
		}
	}
	return "", ""
}

// RemoteURL returns the origin remote URL from a working tree.
func RemoteURL(ctx context.Context, dir string) (string, error) {
	out, _, err := Cmd(ctx, dir, nil, "config", "--get", "remote.origin.url")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// CurrentBranch returns the checked-out branch name.
func CurrentBranch(ctx context.Context, dir string) (string, error) {
	out, _, err := Cmd(ctx, dir, nil, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// RepoRoot returns the top-level of the git working tree.
func RepoRoot(ctx context.Context, dir string) (string, error) {
	out, _, err := Cmd(ctx, dir, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}