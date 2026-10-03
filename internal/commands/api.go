package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// newAPICmd implements a gh-api-style escape hatch: authenticated requests to
// any Switchyard endpoint, with --json output and stable exit codes.
func newAPICmd() *cobra.Command {
	var method, body string
	var raw bool
	cmd := &cobra.Command{
		Use:     "api <path>",
		Short:   "Make an authenticated API request (power-user / agent escape hatch)",
		Example: "  sy api GET /api/work\n  sy api POST /api/work --body '{\"title\":\"x\"}'\n  sy api GET /api/work --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			path := args[0]
			// allow "GET /api/work" as one arg with a space
			if strings.HasPrefix(path, "GET ") || strings.HasPrefix(path, "POST ") || strings.HasPrefix(path, "PATCH ") || strings.HasPrefix(path, "DELETE ") {
				parts := strings.SplitN(path, " ", 2)
				method = parts[0]
				path = parts[1]
			}
			if method == "" {
				method = "GET"
			}
			if !strings.HasPrefix(path, "/") {
				path = "/" + path
			}
			data, status, err := st.client.DoRaw(cmd.Context(), method, path, body)
			if err != nil {
				return err
			}
			if status < 200 || status >= 300 {
				if len(data) > 0 && !raw {
					fmt.Fprintln(os.Stderr, strings.TrimSpace(string(data)))
				}
				return &apiHTTPStatusError{Status: status, Body: string(data)}
			}
			if len(data) == 0 {
				return nil
			}
			if raw || !looksLikeJSON(data) {
				io.Copy(os.Stdout, strings.NewReader(string(data)))
				if !strings.HasSuffix(string(data), "\n") {
					fmt.Fprintln(os.Stdout)
				}
				return nil
			}
			fmt.Fprintln(os.Stdout, string(data))
			return nil
		},
	}
	cmd.Flags().StringVar(&method, "method", "", "HTTP method (GET/POST/PATCH/DELETE)")
	cmd.Flags().StringVar(&body, "body", "", "JSON request body")
	cmd.Flags().BoolVar(&raw, "raw", false, "emit raw body without JSON formatting")
	cmd.Flags().MarkHidden("raw")
	return cmd
}

// apiHTTPStatusError carries a non-2xx status for exit-code mapping.
type apiHTTPStatusError struct {
	Status int
	Body   string
}

func (e *apiHTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Status, strings.TrimSpace(e.Body))
}

func looksLikeJSON(b []byte) bool {
	s := strings.TrimSpace(string(b))
	return len(s) > 0 && (s[0] == '{' || s[0] == '[')
}