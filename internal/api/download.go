package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// DownloadFile streams a bounded response into a private sibling temporary file.
// The final path is published without replacing an existing file. Redirects
// retain the client's prohibition, so session credentials never cross origins.
func (c *Client) DownloadFile(ctx context.Context, path, destination, expectedCommit string, limit int64) (int64, error) {
	return c.downloadFile(ctx, path, destination, expectedCommit, "", -1, limit)
}

func (c *Client) DownloadAsset(ctx context.Context, path, destination, checksum string, size int64) (int64, error) {
	return c.downloadFile(ctx, path, destination, "", checksum, size, 64<<20)
}

func (c *Client) downloadFile(ctx context.Context, path, destination, expectedCommit, checksum string, size, limit int64) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("Cookie", "switchyard_session="+c.Token)
	}
	client := *c.http
	client.Timeout = 2 * time.Minute
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, &HTTPError{Status: resp.StatusCode, Message: fmt.Sprintf("archive download: HTTP %d", resp.StatusCode)}
	}
	if expectedCommit != "" && resp.Header.Get("X-Switchyard-Commit") != expectedCommit {
		return 0, fmt.Errorf("archive commit does not match resolved ref")
	}
	if limit <= 0 || resp.ContentLength > limit {
		return 0, fmt.Errorf("archive exceeds download limit")
	}
	file, err := os.CreateTemp(filepath.Dir(destination), ".sy-download-*")
	if err != nil {
		return 0, err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	digest := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, digest), io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return 0, err
	}
	if n > limit {
		return 0, fmt.Errorf("archive exceeds download limit")
	}
	if resp.ContentLength >= 0 && n != resp.ContentLength {
		return 0, io.ErrUnexpectedEOF
	}
	if checksum != "" && (hex.EncodeToString(digest.Sum(nil)) != checksum || n != size) {
		return 0, fmt.Errorf("release asset size or SHA256 mismatch")
	}
	if err = file.Sync(); err != nil {
		return 0, err
	}
	if err = file.Close(); err != nil {
		return 0, err
	}
	if err = os.Link(file.Name(), destination); err != nil {
		return 0, fmt.Errorf("publish download without overwrite: %w", err)
	}
	return n, nil
}
