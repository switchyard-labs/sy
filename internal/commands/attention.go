package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// AttentionItem is a human-readable item needing direction.
type AttentionItem struct {
	Kind      string `json:"kind"`
	TargetID  string `json:"target_id"`
	Repo      string `json:"repo,omitempty"`
	Branch    string `json:"branch,omitempty"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"created_at"`
}

func newAttentionCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "attention", Short: "Items requiring human direction"}
	cmd.AddCommand(newAttentionListCmd(), newAttentionViewCmd())
	return cmd
}

func newAttentionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List items that need attention",
		Aliases: []string{"ls"},
		Example: "  sy attention\n  sy attention list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct {
				Count int            `json:"count"`
				Items []AttentionItem `json:"items"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/attention", &out); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			if out.Count == 0 {
				st.renderer.Print("nothing needs your attention right now.")
				return nil
			}
			st.renderer.Title(fmt.Sprintf("%d item(s) need your attention:", out.Count))
			rows := make([][]string, 0, len(out.Items))
			for _, it := range out.Items {
				rows = append(rows, []string{it.TargetID, it.Kind, it.Repo, it.Branch, it.Summary, timeAgo(it.CreatedAt)})
			}
			st.renderer.Table([]string{"TARGET", "KIND", "REPO", "BRANCH", "SUMMARY", "WHEN"}, rows)
			return nil
		},
	}
}

func newAttentionViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <target-id>",
		Short:   "Show an attention item and what you can do",
		Example: "  sy attention view wk_123\n  sy attention view iq_91e00cfc8b",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct {
				Count int            `json:"count"`
				Items []AttentionItem `json:"items"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/attention", &out); err != nil {
				return err
			}
			var found *AttentionItem
			for i := range out.Items {
				if out.Items[i].TargetID == args[0] {
					found = &out.Items[i]
					break
				}
			}
			if found == nil {
				return fmt.Errorf("no attention item with target %s", args[0])
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(found)
			}
			st.renderer.KV([][2]string{
				{"Target", found.TargetID},
				{"Kind", found.Kind},
				{"Repo", found.Repo},
				{"Branch", found.Branch},
				{"What happened", found.Summary},
			})
			st.renderer.Print("\nOptions: escalate for a decision packet (sy attention escalate) or resolve via the product UI.")
			return nil
		},
	}
}