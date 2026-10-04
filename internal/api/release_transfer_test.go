package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAssetDownloadIntegrityAndNoOverwrite(t *testing.T) {
	payload := []byte("release payload")
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(payload) }))
	defer server.Close()
	client := New(server.URL, "")
	target := filepath.Join(t.TempDir(), "asset")
	if _, err := client.DownloadAsset(t.Context(), "/asset", target, "bad", int64(len(payload))); err == nil {
		t.Fatal("accepted corrupt checksum")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("published unverified download")
	}
	if _, err := client.DownloadAsset(t.Context(), "/asset", target, digest, int64(len(payload))); err != nil {
		t.Fatal(err)
	}
	if _, err := client.DownloadAsset(t.Context(), "/asset", target, digest, int64(len(payload))); err == nil {
		t.Fatal("overwrote existing file")
	}
}
func TestAssetUploadRawBodyAndMutationNotRetried(t *testing.T) {
	payload := []byte("binary\x00payload")
	target := filepath.Join(t.TempDir(), "asset")
	if err := os.WriteFile(target, payload, 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, err := io.ReadAll(r.Body)
		if err != nil || string(b) != string(payload) || r.ContentLength != int64(len(payload)) {
			t.Error("upload body or length changed")
		}
		if r.Header.Get("Cookie") != "switchyard_session=test-token" {
			t.Error("authentication missing")
		}
		w.WriteHeader(503)
		w.Write([]byte(`{"error":"storage_unavailable"}`))
	}))
	defer server.Close()
	if err := New(server.URL, "test-token").UploadFile(t.Context(), "/asset", target, nil); err == nil {
		t.Fatal("expected failure")
	}
	if calls != 1 {
		t.Fatal("mutation retried")
	}
}
