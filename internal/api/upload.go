package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// UploadFile sends exactly one raw request. Mutations are never retried implicitly.
func (c *Client) UploadFile(ctx context.Context, path, filename string, out any) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() || stat.Size() < 1 || stat.Size() > 64<<20 {
		return fmt.Errorf("asset must be a regular file between 1 byte and 64 MiB")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+path, file)
	if err != nil {
		return err
	}
	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("User-Agent", c.UserAgent)
	if c.Token != "" {
		req.Header.Set("Cookie", "switchyard_session="+c.Token)
	}
	client := *c.http
	client.Timeout = 2 * time.Minute
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var body struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(b, &body)
		return &HTTPError{Status: resp.StatusCode, Code: body.Error}
	}
	if out != nil {
		return json.Unmarshal(b, out)
	}
	return nil
}
