package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/switchyard-labs/sy/internal/api"
	"github.com/switchyard-labs/sy/internal/config"
	"github.com/switchyard-labs/sy/internal/output"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with a Switchyard host",
	}
	cmd.AddCommand(newAuthLoginCmd(), newAuthStatusCmd(), newAuthLogoutCmd(), newAuthSwitchCmd())
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var host string
	var username, password string
	var useFlags bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to a Switchyard host",
		Example: `  sy auth login --host http://45.79.189.46
  sy auth login --host demo --username alice --password-stdin`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if host == "" {
				return fmt.Errorf("--host is required (e.g. --host http://45.79.189.46)")
			}
			if useFlags {
				// --password-stdin: read username then password from stdin
				r := bufio.NewReader(os.Stdin)
				if u, _ := r.ReadString('\n'); u != "" {
					username = strings.TrimSpace(u)
				}
				if p, _ := r.ReadString('\n'); p != "" {
					password = strings.TrimSpace(p)
				}
			} else {
				if username == "" {
					fmt.Fprintf(os.Stdout, "Switchyard host: %s\nUsername: ", host)
					u, _ := bufio.NewReader(os.Stdin).ReadString('\n')
					username = strings.TrimSpace(u)
				}
				if password == "" {
					p, err := readPassword("Password: ")
					if err != nil {
						return err
					}
					password = p
				}
			}
			if username == "" || password == "" {
				return fmt.Errorf("username and password are required")
			}
			base := host
			if !hasScheme(base) {
				base = "http://" + base
			}
			client := api.New(base, "")
			key := config.NormalizeHost(host)
			// set User-Agent
			client.UserAgent = "sy/" + version
			token, err := client.Login(cmd.Context(), base, username, password)
			if err != nil {
				return err
			}
			cfg, _ := config.Load()
			cfg.SetHost(key, config.Host{Token: token, User: username})
			prev := cfg.SetActive(key)
			if err := cfg.Save(); err != nil {
				return err
			}
			r := output.New(os.Stdout, false, false, false)
			if prev != "" && prev != key {
				r.Print(fmt.Sprintf("Switched from %s to %s", prev, key))
			}
			r.Print(fmt.Sprintf("Logged in to %s as %s", key, username))
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Switchyard host URL")
	cmd.Flags().StringVar(&username, "username", "", "username (omit to prompt)")
	cmd.Flags().StringVar(&password, "password", "", "password (avoid on shared shells; prefer prompt or --password-stdin)")
	cmd.Flags().BoolVar(&useFlags, "password-stdin", false, "read username/password from stdin (user\\npassword)")
	cmd.Flags().MarkHidden("password")
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the active host and authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			user, err := st.client.Me(cmd.Context())
			if err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"host": st.host, "user": user, "authenticated": true})
			}
			st.renderer.Print(fmt.Sprintf("Host:   %s", st.host))
			st.renderer.Print(fmt.Sprintf("User:   %s", user))
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out of the active host",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			_ = st.client.Logout(cmd.Context())
			delete(st.cfg.Hosts, st.host)
			if st.cfg.ActiveHost == st.host {
				st.cfg.ActiveHost = ""
			}
			if err := st.cfg.Save(); err != nil {
				return err
			}
			st.renderer.Print(fmt.Sprintf("Logged out of %s", st.host))
			return nil
		},
	}
}

func newAuthSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <host>",
		Short: "Switch the active Switchyard host",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			key := config.NormalizeHost(args[0])
			if _, ok := st.cfg.Hosts[key]; !ok {
				return fmt.Errorf("host %q is not configured", key)
			}
			prev := st.cfg.SetActive(key)
			if err := st.cfg.Save(); err != nil {
				return err
			}
			if prev != "" && prev != key {
				st.renderer.Print(fmt.Sprintf("Switched from %s to %s", prev, key))
			} else {
				st.renderer.Print(fmt.Sprintf("Active host: %s", key))
			}
			return nil
		},
	}
}

// readPassword reads a password without echo using x/term.
func readPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if int(os.Stdin.Fd()) >= 0 {
		if p, err := term.ReadPassword(int(os.Stdin.Fd())); err == nil {
			fmt.Fprintln(os.Stderr)
			return strings.TrimSpace(string(p)), nil
		}
	}
	r := bufio.NewReader(os.Stdin)
	s, _ := r.ReadString('\n')
	return strings.TrimSpace(s), nil
}