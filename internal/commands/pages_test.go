package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/switchyard-labs/sy/internal/api"
	"github.com/switchyard-labs/sy/internal/output"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPagesMutationContractDoesNotRetry(t *testing.T) {
	for _, tc := range []struct {
		args         []string
		path, method string
	}{
		{[]string{"deploy", "--version", "7", "--operation-id", strings.Repeat("a", 32)}, "/deploy", "POST"},
		{[]string{"promote", "dpl_candidate", "--generation", "3", "--operation-id", strings.Repeat("b", 32)}, "/deployments/dpl_candidate/promote", "POST"},
		{[]string{"rollback", "dpl_previous", "--generation", "4", "--operation-id", strings.Repeat("c", 32)}, "/deployments/dpl_previous/rollback", "POST"},
		{[]string{"configure", "--build", "nift build", "--ref", "refs/tags/v1", "--directory", "docs", "--output", "dist"}, "/config", "PATCH"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/repositories" {
					w.Write([]byte(`{"items":[{"owner_slug":"strut-labs","slug":"foo.js","full_name":"strut-labs/foo.js"}]}`))
					return
				}
				calls++
				if r.Method != tc.method || r.URL.Path != "/api/repositories/strut-labs/foo.js/pages"+tc.path {
					t.Errorf("wrong route %s %s", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if tc.path == "/deploy" && body["version"] != "7" {
					t.Error(body)
				}
				if tc.path == "/config" {
					config := body["config"].(map[string]any)
					if config["ref"] != "refs/tags/v1" || config["working_directory"] != "docs" {
						t.Error(config)
					}
				}
				w.WriteHeader(503)
				w.Write([]byte(`{"error":"pages_pending"}`))
			}))
			defer srv.Close()
			st := &state{client: api.New(srv.URL, ""), renderer: output.New(&bytes.Buffer{}, true, false, true)}
			root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
			root.SetContext(context.WithValue(t.Context(), stateKey{}, st))
			root.AddCommand(newPagesCmd())
			root.SetArgs(append([]string{"pages", "--repo", "strut-labs/foo.js"}, tc.args...))
			if root.Execute() == nil {
				t.Fatal("backend error ignored")
			}
			if calls != 1 {
				t.Fatal("mutation retried", calls)
			}
		})
	}
}
