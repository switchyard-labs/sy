package commands

import (
	"bytes"
	"os"
	"os/exec"
	"testing"
)

// TestVersionNoHostAndNoTTY verifies version works and that auth-required
// commands fail fast (never prompt) when stdin is not a TTY.
func TestVersionNoHostAndNoTTY(t *testing.T) {
	bin := buildCLI(t)
	cmd := exec.Command(bin, "version")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !bytes.Contains(out.Bytes(), []byte("sy ")) {
		t.Fatalf("version output: %q", out.String())
	}
	// no-host command must fail fast, not hang waiting for a prompt
	cmd2 := exec.Command(bin, "work", "list")
	var errb bytes.Buffer
	cmd2.Stdin = bytes.NewReader([]byte{}) // empty stdin, no TTY
	cmd2.Stderr = &errb
	if err := cmd2.Run(); err == nil {
		t.Fatal("expected failure without a configured host")
	}
	if !bytes.Contains(errb.Bytes(), []byte("no host configured")) {
		t.Fatalf("stderr: %q", errb.String())
	}
}

func buildCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := dir + "/sy"
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/sy")
	cmd.Dir = "../.."
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	return bin
}
