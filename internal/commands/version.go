package commands

import (
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
			cmd.Println("sy " + Version)
			if Commit != "" {
				cmd.Println("commit " + Commit)
			}
			if BuildDate != "" {
				cmd.Println("built " + BuildDate)
			}
			if GoVersion != "" {
				cmd.Println("go " + GoVersion)
			}
			return nil
		},
	}
}
