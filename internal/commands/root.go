// Package commands implements the sy CLI. Each command produces a domain
// result rendered by the shared output renderer (human or --json).
package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/switchyard-labs/sy/internal/api"
	"github.com/switchyard-labs/sy/internal/config"
	"github.com/switchyard-labs/sy/internal/output"
)

// Stable exit codes (documented contract for humans and agents).
const (
	ExitOK            = 0
	ExitRemoteFailure = 1
	ExitInvalid       = 2
	ExitAuth          = 3
	ExitConflictStale = 4
	ExitUnavailable   = 5
)

// ExitCodeFor maps an error to a stable exit code.
func ExitCodeFor(err error) int {
	var he *api.HTTPError
	if errors.As(err, &he) {
		switch {
		case he.Status == 401:
			return ExitAuth
		case he.Status == 403:
			return ExitAuth
		case he.Status == 404:
			return ExitRemoteFailure
		case he.Status == 409:
			if errors.Is(err, api.ErrStale) {
				return ExitConflictStale
			}
			return ExitConflictStale
		case he.Status == 429 || he.Status >= 500:
			return ExitUnavailable
		}
		return ExitRemoteFailure
	}
	switch {
	case errors.Is(err, api.ErrUnauthorized), errors.Is(err, api.ErrForbidden):
		return ExitAuth
	case errors.Is(err, api.ErrStale), errors.Is(err, api.ErrConflict):
		return ExitConflictStale
	case errors.Is(err, api.ErrUnavailable):
		return ExitUnavailable
	case errors.Is(err, api.ErrNotFound):
		return ExitRemoteFailure
	}
	if errors.Is(err, context.Canceled) {
		return ExitRemoteFailure
	}
	return ExitRemoteFailure
}

// state carries per-invocation shared values.
type state struct {
	cfg      *config.Config
	client   *api.Client
	renderer *output.Renderer
	host     string
	noJSON   bool // set when the command itself manages JSON (e.g. sy api)
}

type rootOpts struct {
	json    bool
	quiet   bool
	noColor bool
	host    string
	verbose bool
}

// New builds the root command.
func New() *cobra.Command {
	opts := &rootOpts{}
	root := &cobra.Command{
		Use:           "sy",
		Short:         "Switchyard CLI — a first-class client for Switchyard",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "version" || cmd.Name() == "login" {
				return nil
			}
			// load config lazily so help/version never read config
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			st := &state{cfg: cfg, renderer: output.New(os.Stdout, opts.json, opts.quiet, opts.noColor)}
			host := opts.host
			if host == "" {
				host = cfg.ActiveHost
			}
			st.host = host
			if host == "" {
				return errors.New("no host configured: run `sy auth login --host <url>` first")
			}
			hostCfg, ok := cfg.Hosts[host]
			if !ok {
				return fmt.Errorf("host %q is not configured: run `sy auth login --host %s`", host, host)
			}
			base := host
			if !hasScheme(host) {
				base = "http://" + host
			}
			st.client = api.New(base, hostCfg.Token)
			cmd.SetContext(context.WithValue(cmd.Context(), stateKey{}, st))
			return nil
		},
	}
	root.PersistentFlags().BoolVar(&opts.json, "json", false, "emit stable JSON")
	root.PersistentFlags().BoolVarP(&opts.quiet, "quiet", "q", false, "suppress human output")
	root.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "disable ANSI color")
	root.PersistentFlags().StringVar(&opts.host, "host", "", "Switchyard host (overrides active host)")

	root.AddCommand(newVersionCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newBrowseCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newAuthCmd())
	root.AddCommand(newRepoCmd())
	root.AddCommand(newWorkCmd())
	root.AddCommand(newAttemptCmd())
	root.AddCommand(newPRCmd())
	root.AddCommand(newQueueCmd())
	root.AddCommand(newAttentionCmd())
	root.AddCommand(newWorkflowCmd())
	root.AddCommand(newAgentCmd())
	root.AddCommand(newOrgCmd())
	root.AddCommand(newAPICmd())
	return root
}

type stateKey struct{}

func stateFrom(cmd *cobra.Command) *state {
	st, _ := cmd.Context().Value(stateKey{}).(*state)
	return st
}

func hasScheme(u string) bool {
	return len(u) > 7 && (u[:7] == "http://" || u[:8] == "https://")
}

func strOrNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
