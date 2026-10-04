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
	"testing"
)

func TestProposalGenerationRecoveryContract(t *testing.T) {
	for _, recovery := range []bool{false, true} {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/api/repositories" {
				w.Write([]byte(`{"items":[{"owner_slug":"n-ham","slug":"foo.js","full_name":"n-ham/foo.js"}]}`))
				return
			}
			calls++
			if r.Method != "POST" || r.URL.Path != "/api/repositories/n-ham/foo.js/proposals/generate" {
				t.Errorf("route: %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if recovery {
				if body["execution_id"] != "exec_saved" || body["prompt"] != "" || r.Header.Get("X-Switchyard-Agent") != "" {
					t.Error(body)
				}
			} else {
				var selected map[string]string
				if json.Unmarshal([]byte(r.Header.Get("X-Switchyard-Agent")), &selected) != nil || selected["credential_id"] != "personal-id" || selected["model"] != "review-model" {
					t.Error(selected)
				}
				if body["proposal_id"] != "prop_target" {
					t.Error(body)
				}
			}
			w.WriteHeader(503)
			w.Write([]byte(`{"error":"pending"}`))
		}))
		st := &state{client: api.New(srv.URL, ""), renderer: output.New(&bytes.Buffer{}, true, false, true)}
		root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
		root.SetContext(context.WithValue(t.Context(), stateKey{}, st))
		root.AddCommand(newProposalCmd())
		args := []string{"proposal", "--repo", "n-ham/foo.js", "generate"}
		if recovery {
			args = append(args, "--execution-id", "exec_saved")
		} else {
			args = append(args, "--prompt", "Consider evidence", "--proposal-id", "prop_target", "--provider", "openai", "--model", "review-model", "--credential", "personal-id")
		}
		root.SetArgs(args)
		if root.Execute() == nil || calls != 1 {
			t.Fatal("error swallowed or mutation retried", calls)
		}
		srv.Close()
	}
}
