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
	cmd.AddCommand(newWorkListCmd(), newWorkViewCmd(), newWorkCreateCmd(), newWorkCloseCmd())
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
			var out struct{ Items []WorkItem `json:"items"` }
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
			st.renderer.KV([][2]string{
				{"ID", str(w, "id")},
				{"Title", str(w, "title")},
				{"Kind", str(w, "kind")},
				{"Status", str(w, "status")},
				{"Owner", str(w, "owner")},
				{"Created", str(w, "created_at")},
			})
			return nil
		},
	}
}

func newWorkCreateCmd() *cobra.Command {
	var title, kind string
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
			var out struct{ ID string `json:"id"` }
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
	return cmd
}

func newWorkCloseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "close <id>",
		Short:   "Close a Work item",
		Args:    cobra.ExactArgs(1),
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