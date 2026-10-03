package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newQueueCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "queue", Short: "Integration Queue"}
	cmd.AddCommand(newQueueListCmd(), newQueueViewCmd(), newQueueRequeueCmd())
	return cmd
}

func newQueueListCmd() *cobra.Command {
	var status, repo string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List Integration Queue items",
		Example: "  sy queue list\n  sy queue list --status blocked --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct {
				Items []map[string]any `json:"items"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/queue", &out); err != nil {
				return err
			}
			items := out.Items
			if status != "" {
				filtered := items[:0]
				for _, it := range items {
					if str(it, "status") == status {
						filtered = append(filtered, it)
					}
				}
				items = filtered
			}
			if repo != "" {
				filtered := items[:0]
				for _, it := range items {
					if str(it, "repo") == repo {
						filtered = append(filtered, it)
					}
				}
				items = filtered
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"items": items})
			}
			rows := make([][]string, 0, len(items))
			for _, it := range items {
				rows = append(rows, []string{str(it, "id"), str(it, "repo"), str(it, "branch"), str(it, "status"), str(it, "risk"), str(it, "error")})
			}
			st.renderer.Table([]string{"ID", "REPO", "BRANCH", "STATUS", "RISK", "ERROR"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status (queued/running/blocked/done/failed)")
	cmd.Flags().StringVar(&repo, "repo", "", "filter by repository")
	return cmd
}

func newQueueViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <id>",
		Short:   "Show an Integration Queue item with a clear blocked reason",
		Example: "  sy queue view iq_123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct {
				Items []map[string]any `json:"items"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/queue", &out); err != nil {
				return err
			}
			var it map[string]any
			for _, x := range out.Items {
				if str(x, "id") == args[0] {
					it = x
					break
				}
			}
			if it == nil {
				return fmt.Errorf("queue item %s not found", args[0])
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(it)
			}
			status := str(it, "status")
			if status == "blocked" {
				st.renderer.Print("Integration blocked\n")
				st.renderer.KV([][2]string{
					{"Repository", str(it, "repo")},
					{"Branch", str(it, "branch")},
					{"Pull request", str(it, "pr_id")},
					{"Risk", str(it, "risk")},
				})
				st.renderer.Print("\nReason\n------")
				st.renderer.Print(blockedReason(str(it, "error")))
				st.renderer.Print("\nNext\n----\nResolve the finding or escalate for a decision, then requeue (sy queue requeue " + str(it, "id") + ").")
				return nil
			}
			st.renderer.KV([][2]string{
				{"ID", str(it, "id")},
				{"Repo", str(it, "repo")},
				{"Branch", str(it, "branch")},
				{"Status", statusSym(status) + " " + status},
				{"Risk", str(it, "risk")},
				{"Error", str(it, "error")},
			})
			return nil
		},
	}
}

func newQueueRequeueCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "requeue <id>",
		Short:   "Re-queue a blocked queue item",
		Example: "  sy queue requeue iq_123",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			if err := st.client.Do(cmd.Context(), "POST", "/api/queue/"+args[0]+"/requeue", nil); err != nil {
				return err
			}
			if !st.renderer.JSONMode {
				st.renderer.Print(fmt.Sprintf("requeued %s", args[0]))
			} else {
				return st.renderer.Emit(map[string]any{"id": args[0], "status": "queued"})
			}
			return nil
		},
	}
}

// blockedReason turns a raw queue error into a short human explanation.
func blockedReason(err string) string {
	switch {
	case containsAny(err, "semantic_conflict"):
		return "The combined change violates a repository contract. Git merges cleanly, but Switchyard detected incompatible semantics."
	case containsAny(err, "preview_conflict"):
		return "The attempt does not merge cleanly into canonical (a textual/merge conflict)."
	case containsAny(err, "check_not_passed"):
		return "The change did not pass its check."
	default:
		return err
	}
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}
