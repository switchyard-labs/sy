package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/switchyard-labs/sy/internal/config"
	"github.com/switchyard-labs/sy/internal/gitutil"
)

// newStatusCmd answers "where am I, what is happening" — host/account, and
// repo-contextual open work/PRs/attention when inside a checkout.
func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "status",
		Short:   "Show host, user, and current repository status",
		Example: "  sy status\n  sy status --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			user, err := st.client.Me(cmd.Context())
			if err != nil {
				return err
			}
			// repo context (best effort)
			owner, name := currentRepoContext(cmd.Context())
			out := map[string]any{
				"host":   st.host,
				"user":   user,
				"repo":   nil,
			}
			if name != "" {
				out["repo"] = owner + "/" + name
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			st.renderer.KV([][2]string{
				{"Host", st.host},
				{"User", user},
				{"Repository", owner + "/" + name},
			})
			return nil
		},
	}
}

// newBrowseCmd opens the Switchyard web page for the current repo (or host).
func newBrowseCmd() *cobra.Command {
	var flagRepo string
	cmd := &cobra.Command{
		Use:     "browse",
		Short:   "Open the Switchyard web page for a repository",
		Example: "  sy browse\n  sy browse demo-basic",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			name := flagRepo
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				n, err := currentRepoName(cmd.Context())
				if err == nil {
					name = n
				}
			}
			base := st.host
			if !hasScheme(base) {
				base = "http://" + base
			}
			url := base + "/"
			if name != "" {
				url = base + "/repo.html?name=" + name
			}
			return openBrowser(url)
		},
	}
	cmd.Flags().StringVar(&flagRepo, "repo", "", "owner/repo or repo name")
	return cmd
}

// newConfigCmd exposes safe, token-free config inspection.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Inspect sy configuration"}
	cmd.AddCommand(newConfigListCmd(), newConfigGetCmd(), newConfigSetCmd())
	return cmd
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List configured hosts",
		Example: "  sy config list\n  sy config list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			type hostRow struct {
				Host   string `json:"host"`
				User   string `json:"user"`
				Active bool   `json:"active"`
			}
			rows := []hostRow{}
			for _, h := range st.cfg.HostList() {
				hc := st.cfg.Hosts[h]
				rows = append(rows, hostRow{Host: h, User: hc.User, Active: h == st.cfg.ActiveHost})
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"items": rows})
			}
			tab := make([][]string, 0, len(rows))
			for _, r := range rows {
				mark := ""
				if r.Active {
					mark = "*"
				}
				tab = append(tab, []string{r.Host, r.User, mark})
			}
			st.renderer.Table([]string{"HOST", "USER", "ACTIVE"}, tab)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <host>",
		Short: "Show a host profile (never the token)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			key := config.NormalizeHost(args[0])
			h, ok := st.cfg.Hosts[key]
			if !ok {
				return fmt.Errorf("host %q not configured", key)
			}
			active := ""
			if st.cfg.ActiveHost == key {
				active = "*"
			}
			st.renderer.KV([][2]string{
				{"Host", key},
				{"User", h.User},
				{"Active", active},
				{"Token", "••••• (not shown)"},
			})
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "set-host <host>",
		Short: "Set the active host (token-free)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			key := config.NormalizeHost(args[0])
			if _, ok := st.cfg.Hosts[key]; !ok {
				// allow setting a bare host + user before login
				st.cfg.SetHost(key, config.Host{User: user})
			}
			st.cfg.SetActive(key)
			if err := st.cfg.Save(); err != nil {
				return err
			}
			if !st.renderer.JSONMode {
				st.renderer.Print("active host: " + key)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&user, "user", "", "user for a new host")
	return cmd
}

// currentRepoContext returns owner/name for the current checkout (best effort).
func currentRepoContext(ctx context.Context) (string, string) {
	wd, err := os.Getwd()
	if err != nil {
		return "", ""
	}
	remote, err := gitutil.RemoteURL(ctx, wd)
	if err != nil {
		return "", ""
	}
	owner, name := gitutil.DetectOwnerRepo(remote)
	return owner, name
}

// openBrowser opens a URL in the user's browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}