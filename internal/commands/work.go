package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// WorkItem is the domain view of a Work item.
type WorkItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	Owner     string `json:"owner"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func newWorkCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "work", Short: "Work items (issues/tasks)"}
	cmd.AddCommand(newWorkListCmd(), newWorkViewCmd(), newWorkCreateCmd(), newWorkCommentCmd(), newWorkCloseCmd())
	return cmd
}

func newWorkListCmd() *cobra.Command {
	var status, repo string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List Work items",
		Example: "  sy work list\n  sy work list --status open --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct {
				Items []WorkItem `json:"items"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/work", &out); err != nil {
				return err
			}
			items := out.Items
			if status != "" {
				items = filterWork(items, func(w WorkItem) bool { return strings.EqualFold(w.Status, status) })
			}
			if repo != "" {
				items = filterWork(items, func(w WorkItem) bool { return strings.Contains(w.ID, "") || w.Owner == repo })
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"items": items})
			}
			rows := make([][]string, 0, len(items))
			for _, w := range items {
				rows = append(rows, []string{w.ID, w.Title, w.Kind, w.Status, w.Owner, timeAgo(w.UpdatedAt)})
			}
			st.renderer.Table([]string{"ID", "TITLE", "KIND", "STATUS", "OWNER", "UPDATED"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status (open/closed)")
	cmd.Flags().StringVar(&repo, "repo", "", "filter by repository")
	return cmd
}

func newWorkViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <id>",
		Short:   "Show a Work item",
		Example: "  sy work view wk_123 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var w map[string]any
			if err := st.client.Do(cmd.Context(), "GET", "/api/work/"+args[0], &w); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(w)
			}
			title := str(w, "title")
			kind := str(w, "kind")
			status := str(w, "status")
			st.renderer.Print(fmt.Sprintf("%s  · %s · %s %s", title, str(w, "id"), kind, statusSym(status)))
			st.renderer.Print("")
			st.renderer.KV([][2]string{
				{"Repository", str(w, "repo")},
				{"Owner", str(w, "owner")},
				{"Assignee", str(w, "assignee")},
				{"Updated", relative(str(w, "updated_at"))},
			})
			if body := str(w, "body"); body != "" {
				st.renderer.Print("\nBody\n----\n" + body)
			}
			if atts, ok := w["attempts"].([]any); ok && len(atts) > 0 {
				st.renderer.Print("\nAttempts\n--------")
				for _, a := range atts {
					m, _ := a.(map[string]any)
					st.renderer.Print(fmt.Sprintf("  %s  %s  %s", str(m, "id"), str(m, "branch"), str(m, "status")))
				}
			}
			if prs, ok := w["pull_requests"].([]any); ok && len(prs) > 0 {
				st.renderer.Print("\nPull requests\n-------------")
				for _, p := range prs {
					m, _ := p.(map[string]any)
					st.renderer.Print(fmt.Sprintf("  %s  %s  %s", str(m, "id"), str(m, "title"), str(m, "status")))
				}
			}
			return nil
		},
	}
}

func newWorkCreateCmd() *cobra.Command {
	var title, kind, body, bodyFile string
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a Work item",
		Example: `  sy work create --title "Add CI logs" --kind feature`,
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if kind == "" {
				kind = "task"
			}
			var out struct {
				ID string `json:"id"`
			}
			if err := st.client.Do(cmd.Context(), "POST", "/api/work", &out, apiBody(map[string]any{"title": title, "kind": kind})); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"id": out.ID, "title": title, "kind": kind})
			}
			st.renderer.Print(fmt.Sprintf("created %s (%s)", out.ID, title))
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "work title")
	cmd.Flags().StringVar(&kind, "kind", "task", "work kind (feature/fix/task)")
	cmd.Flags().StringVar(&body, "body", "", "work body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "read body from file ('-' for stdin)")
	return cmd
}

func newWorkCommentCmd() *cobra.Command {
	var body, bodyFile string
	cmd := &cobra.Command{
		Use:     "comment <id>",
		Short:   "Add a comment to a Work item",
		Example: "  sy work comment wk_123 --body 'please review'",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			if body == "" && bodyFile != "" {
				b, err := os.ReadFile(bodyFile)
				if err != nil {
					return err
				}
				body = string(b)
			}
			if body == "" {
				return fmt.Errorf("--body or --body-file is required")
			}
			var out map[string]any
			if err := st.client.Do(cmd.Context(), "POST", "/api/work/"+args[0]+"/comments", &out, apiBody(map[string]any{"body": body})); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"work": args[0], "body": body})
			}
			st.renderer.Print(fmt.Sprintf("commented on %s", args[0]))
			return nil
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "comment body")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "read body from file")
	return cmd
}

func newWorkCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "close <id>",
		Short: "Close a Work item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			if err := st.client.Do(cmd.Context(), "PATCH", "/api/work/"+args[0], nil, apiBody(map[string]any{"status": "closed"})); err != nil {
				return err
			}
			if !st.renderer.JSONMode {
				st.renderer.Print(fmt.Sprintf("closed %s", args[0]))
			} else {
				return st.renderer.Emit(map[string]any{"id": args[0], "status": "closed"})
			}
			return nil
		},
	}
}

func filterWork(items []WorkItem, keep func(WorkItem) bool) []WorkItem {
	out := make([]WorkItem, 0, len(items))
	for _, w := range items {
		if keep(w) {
			out = append(out, w)
		}
	}
	return out
}

func str(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

var _ = os.Getenv
