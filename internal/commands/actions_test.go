package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/switchyard-labs/sy/internal/api"
	"github.com/switchyard-labs/sy/internal/output"
)

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
