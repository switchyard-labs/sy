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

// CredentialFile writes a 0600 git config that supplies an Authorization
// header via GIT_CONFIG_GLOBAL, keeping the token out of argv (process list)
// and out of the clone's .git/config. Returns the path and a cleanup func.
func CredentialFile(token string) (string, func(), error) {
	f, err := os.CreateTemp("", "sy-git-cred-*.config")
	if err != nil {
		return "", nil, err
	}
	if err := os.Chmod(f.Name(), 0600); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, err
	}
	if _, err := f.WriteString("[http]\n	extraHeader = Authorization: Bearer " + token + "\n"); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, err
	}
	f.Close()
	return f.Name(), func() { os.Remove(f.Name()) }, nil
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
	// scp-style: user@host:owner/repo
	if i := strings.Index(s, ":"); i >= 0 && !strings.Contains(s, "://") {
		s = s[i+1:]
	} else {
		// URL-style: scheme://host/owner/repo
		s = strings.TrimPrefix(s, "https://")
		s = strings.TrimPrefix(s, "http://")
		s = strings.TrimPrefix(s, "git@")
		if i := strings.Index(s, "/"); i >= 0 {
			s = s[i+1:]
		}
	}
	parts := strings.Split(s, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
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