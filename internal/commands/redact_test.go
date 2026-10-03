package commands

import (
	"os"
	"strings"
	"testing"
)

// TestSecretStringsNeverInOutputs guards against fixture credentials leaking
// into any code path (a static regression guard).
func TestSecretStringsNeverInOutputs(t *testing.T) {
	secrets := []string{"password123", "art_v2_secret", "sess_secret"}
	srcs := []string{"auth.go", "repo.go", "root.go", "status.go", "queue.go"}
	for _, src := range srcs {
		b, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		for _, s := range secrets {
			if strings.Contains(string(b), s) {
				t.Errorf("%s contains fixture secret %q", src, s)
			}
		}
	}
}
