package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFileBoundedAndNoOverwrite(t *testing.T) {
	for _, mode := range []string{"success", "oversized", "wrong_commit", "existing", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Cookie") != "switchyard_session=test-session" {
					t.Error("missing session")
				}
				if mode == "redirect" {
					http.Redirect(w, r, "https://example.invalid/", http.StatusFound)
					return
				}
				commit := "expected"
				if mode == "wrong_commit" {
					commit = "wrong"
				}
				w.Header().Set("X-Switchyard-Commit", commit)
				w.Write([]byte("archive payload"))
			}))
			defer server.Close()
			dest := filepath.Join(t.TempDir(), "archive.zip")
			if mode == "existing" {
				os.WriteFile(dest, []byte("original"), 0600)
			}
			limit := int64(100)
			if mode == "oversized" {
				limit = 3
			}
			n, err := New(server.URL, "test-session").DownloadFile(context.Background(), "/archive", dest, "expected", limit)
			if mode == "success" {
				data, e := os.ReadFile(dest)
				if err != nil || e != nil || string(data) != "archive payload" || n != 15 {
					t.Fatalf("download: %d %v %q", n, err, data)
				}
			} else {
				if err == nil {
					t.Fatal("expected failure")
				}
				data, e := os.ReadFile(dest)
				if mode == "existing" {
					if e != nil || string(data) != "original" {
						t.Fatal("existing file overwritten")
					}
				} else if !os.IsNotExist(e) {
					t.Fatal("partial download published")
				}
			}
			leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(dest), ".sy-download-*"))
			if len(leftovers) > 0 {
				t.Fatal("temporary file leaked")
			}
		})
	}
}
