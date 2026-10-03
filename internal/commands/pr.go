package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// PR is the domain view of a pull request.
type PR struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Repo        string `json:"repo"`
	Branch      string `json:"branch"`
	Base        string `json:"base"`
	Status      string `json:"status"`
	CheckStatus string `json:"check_status"`
	WorkID      string `json:"work_id,omitempty"`
	AttemptID   string `json:"attempt_id,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func newPRCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "pr", Short: "Pull requests"}
	cmd.AddCommand(newPRListCmd(), newPRViewCmd(), newPRChecksCmd())
	return cmd
}

func newPRListCmd() *cobra.Command {
	var status, repo string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List pull requests",
		Example: "  sy pr list\n  sy pr list --status open --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []PR `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/prs", &out); err != nil {
				return err
			}
			items := out.Items
			if status != "" {
				items = filterPR(items, func(p PR) bool { return strings.EqualFold(p.Status, status) })
			}
			if repo != "" {
				items = filterPR(items, func(p PR) bool { return p.Repo == repo })
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"items": items})
			}
			rows := make([][]string, 0, len(items))
			for _, p := range items {
				rows = append(rows, []string{p.ID, p.Title, p.Repo, p.Branch, p.Status, p.CheckStatus})
			}
			st.renderer.Table([]string{"ID", "TITLE", "REPO", "BRANCH", "STATUS", "CHECK"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status (open/closed/integrated)")
	cmd.Flags().StringVar(&repo, "repo", "", "filter by repository")
	return cmd
}

func newPRViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <id>",
		Short:   "Show a pull request",
		Example: "  sy pr view pr_123 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var p map[string]any
			if err := st.client.Do(cmd.Context(), "GET", "/api/prs/"+args[0], &p); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(p)
			}
			st.renderer.KV([][2]string{
				{"ID", str(p, "id")},
				{"Title", str(p, "title")},
				{"Repo", str(p, "repo")},
				{"Branch", str(p, "branch")},
				{"Base", str(p, "base")},
				{"Status", str(p, "status")},
				{"Check", str(p, "check_status")},
			})
			return nil
		},
	}
}

func newPRChecksCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "checks <id>",
		Short:   "Show checks for a pull request",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var p map[string]any
			if err := st.client.Do(cmd.Context(), "GET", "/api/prs/"+args[0], &p); err != nil {
				return err
			}
			status := str(p, "check_status")
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"pr": args[0], "check_status": status})
			}
			st.renderer.Print(fmt.Sprintf("%s  %s", statusIcon(status), status))
			return nil
		},
	}
}

func filterPR(items []PR, keep func(PR) bool) []PR {
	out := make([]PR, 0, len(items))
	for _, p := range items {
		if keep(p) {
			out = append(out, p)
		}
	}
	return out
}

func statusIcon(s string) string {
	switch strings.ToLower(s) {
	case "pass":
		return "✓"
	case "fail":
		return "✗"
	case "pending":
		return "…"
	default:
		return "·"
	}
}