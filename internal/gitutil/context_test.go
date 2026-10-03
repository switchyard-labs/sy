package gitutil

import (
	"testing"
)

// TestNoOrigin verifies context detection fails cleanly without an origin.
func TestNoOrigin(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Cmd(t.Context(), dir, nil, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoteURL(t.Context(), dir); err == nil {
		t.Fatal("expected error with no origin remote")
	}
}

// TestArtifactsRemoteDetection runs inside a real clone whose origin is an
// Artifacts-style URL.
func TestArtifactsRemoteDetection(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Cmd(t.Context(), dir, nil, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	remote := "https://b7f20353ee8a9e5d2003f52c74ba795e.artifacts.cloudflare.net/git/switchyard-cp0/demo-basic.git"
	if _, _, err := Cmd(t.Context(), dir, nil, "remote", "add", "origin", remote); err != nil {
		t.Fatal(err)
	}
	got, err := RemoteURL(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	o, n := DetectOwnerRepo(got)
	if o != "switchyard-cp0" || n != "demo-basic" {
		t.Fatalf("detected %q/%q", o, n)
	}
}
