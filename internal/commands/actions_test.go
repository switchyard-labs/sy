package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/switchyard-labs/sy/internal/api"
	"github.com/switchyard-labs/sy/internal/output"
)

func TestActionMutationCommandsUseCanonicalContractOnce(t *testing.T) {
	for _, tc := range []struct {
		args         []string
		suffix, mode string
	}{
		{[]string{"run", "--sha", strings.Repeat("a", 40)}, "/dispatch", ""},
		{[]string{"rerun", "run-id", "--failed"}, "/run-id/rerun", "failed"},
		{[]string{"cancel", "run-id"}, "/run-id/cancel", ""},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "GET" && r.URL.Path == "/api/repositories" {
					w.Write([]byte(`{"items":[{"owner_slug":"alice","slug":"railway","full_name":"alice/railway"}]}`))
					return
				}
				calls++
				if r.Method != "POST" || r.URL.Path != "/api/repositories/alice/railway/actions"+tc.suffix {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if tc.mode != "" && body["mode"] != tc.mode {
					t.Errorf("mode=%v", body["mode"])
				}
				if tc.suffix != "/run-id/cancel" {
					if id, ok := body["request_id"].(string); !ok || len(id) != 32 {
						t.Errorf("request ID=%v", body["request_id"])
					}
				}
				if tc.suffix == "/dispatch" && (body["sha"] != strings.Repeat("a", 40) || body["ref"] != "refs/heads/main") {
					t.Errorf("dispatch=%v", body)
				}
				w.WriteHeader(429)
				w.Write([]byte(`{"error":"upstream_rate_limited"}`))
			}))
			defer server.Close()
			st := &state{client: api.New(server.URL, ""), renderer: output.New(&bytes.Buffer{}, true, false, true)}
			root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
			root.SetContext(context.WithValue(t.Context(), stateKey{}, st))
			root.AddCommand(newActionsCmd())
			root.SetArgs(append([]string{"actions", "--repo", "alice/railway"}, tc.args...))
			if err := root.Execute(); err == nil {
				t.Fatal("expected rate limit error")
			}
			if calls != 1 {
				t.Fatalf("unsafe mutation retried: %d", calls)
			}
		})
	}
}

func TestAttemptsPaginationAndRepeatedCursor(t *testing.T) {
	for _, repeat := range []bool{false, true} {
		t.Run(fmt.Sprint(repeat), func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path != "/api/attempts" {
					t.Errorf("unexpected Work scan: %s", r.URL.Path)
				}
				if calls == 1 || repeat {
					w.Write([]byte(`{"items":[{"id":"a"}],"next_cursor":"a"}`))
				} else {
					if r.URL.Query().Get("cursor") != "a" {
						t.Error("missing cursor")
					}
					w.Write([]byte(`{"items":[{"id":"b"}]}`))
				}
			}))
			defer server.Close()
			items, err := listAttempts(t.Context(), &state{client: api.New(server.URL, "")})
			if repeat {
				if err == nil {
					t.Fatal("repeated cursor accepted")
				}
			} else if err != nil || len(items) != 2 || items[1].ID != "b" {
				t.Fatalf("items=%v err=%v", items, err)
			}
			if calls != 2 {
				t.Fatalf("calls=%d", calls)
			}
		})
	}
}

func TestActionFollowDrainsPagesAndStopsAtTerminal(t *testing.T) {
	logCalls := 0
	mutations := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			mutations++
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/logs") {
			logCalls++
			cursor := r.URL.Query().Get("cursor")
			if cursor == "0" {
				w.Write([]byte(`{"lines":[{"id":1,"stream":"stdout","text":"first"}],"cursor":1,"complete":false}`))
			} else if cursor == "1" {
				w.Write([]byte(`{"lines":[{"id":2,"stream":"stderr","text":"second"}],"cursor":2,"complete":true}`))
			} else {
				t.Errorf("unexpected cursor %s", cursor)
			}
			return
		}
		w.Write([]byte(`{"run":{"id":"run","state":{"status":"failure","manifest":{"run":{"jobs":[{"id":"test","steps":[{"id":"unit"}]}]},"jobs":[{"id":"test","steps":[{"id":"unit","status":"failed"}]}]}}}}`))
	}))
	defer server.Close()
	var buf bytes.Buffer
	st := &state{client: api.New(server.URL, ""), renderer: output.New(&buf, true, false, true)}
	if err := followActionLogs(t.Context(), st, "/actions", "run", "", true); err != nil {
		t.Fatal(err)
	}
	if logCalls != 2 || mutations != 0 {
		t.Fatalf("calls logs=%d mutations=%d", logCalls, mutations)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatal(buf.String())
	}
	for _, line := range lines {
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) != nil || event["step"] != "test-unit" {
			t.Fatal(line)
		}
	}
}
func TestActionFollowCancellationNeverCancelsRun(t *testing.T) {
	mutations := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			mutations++
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/logs") {
			w.WriteHeader(202)
			w.Write([]byte(`{"error":"logs_pending","pending":true}`))
			return
		}
		w.Write([]byte(`{"run":{"state":{"status":"running","manifest":{"run":{"jobs":[{"id":"test","steps":[{"id":"unit"}]}]},"jobs":[{"id":"test","steps":[{"id":"unit","status":"failed"}]}]}}}}`))
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	st := &state{client: api.New(server.URL, ""), renderer: output.New(&bytes.Buffer{}, false, false, true)}
	if err := followActionLogs(ctx, st, "/actions", "run", "", true); err != context.DeadlineExceeded {
		t.Fatalf("cancel=%v", err)
	}
	if mutations != 0 {
		t.Fatal("following canceled server run")
	}
}
