package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Build metadata, injected via ldflags at release time.
var (
	Version   = "0.1.0-dev"
	Commit    = ""
	BuildDate = ""
	GoVersion = ""
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Short:   "Print version information",
		Example: "  sy version\n  sy version --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			info := map[string]string{
				"version": Version,
				"commit":  Commit,
				"built":   BuildDate,
				"go":      GoVersion,
			}
			if st != nil && st.renderer.JSONMode {
				return st.renderer.Emit(info)
			}
			_ = info
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "sy "+Version)
			if Commit != "" {
				fmt.Fprintln(out, "commit "+Commit)
			}
			if BuildDate != "" {
				fmt.Fprintln(out, "built "+BuildDate)
			}
			if GoVersion != "" {
				fmt.Fprintln(out, "go "+GoVersion)
			}
			return nil
		},
	}
}
