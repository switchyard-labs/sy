// sy — the first-class command-line client for Switchyard.
package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/switchyard-labs/sy/internal/api"
	"github.com/switchyard-labs/sy/internal/commands"
)

// Build metadata set by ldflags at release time.
var (
	version   = "0.1.0-dev"
	commit    = ""
	buildDate = ""
)

func main() {
	commands.Version = version
	commands.Commit = commit
	commands.BuildDate = buildDate
	commands.GoVersion = runtime.Version()
	err := commands.New().Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err.Error())
		// actionable auth-expiry message
		var he *api.HTTPError
		if errors.As(err, &he) && he.Status == 401 {
			fmt.Fprintln(os.Stderr, "run: sy auth login --host <url>")
		} else if errors.Is(err, api.ErrUnauthorized) {
			fmt.Fprintln(os.Stderr, "run: sy auth login --host <url>")
		}
		os.Exit(commands.ExitCodeFor(err))
	}
}

var _ = strings.TrimSpace
