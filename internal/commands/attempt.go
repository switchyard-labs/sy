package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Attempt is the domain view of an Attempt (an isolated branch from a Work item).
type Attempt struct {
	ID        string `json:"id"`
	WorkID    string `json:"work_id"`
	Repo      string `json:"repo"`
	Branch    string `json:"branch"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Owner     string `json:"owner,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func newAttemptCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "attempt", Short: "Attempts (isolated work branches)", Aliases: []string{"attempts"}}
	cmd.AddCommand(newAttemptListCmd(), newAttemptViewCmd(), newAttemptRunCmd())
	return cmd
}

// listAttempts scans Work items for their attempts. Switchyard's work LIST
// omits attempts (they appear only in work detail), so this fetches details for
// a bounded window. Capped to avoid hammering the server; see
// docs/switchyard-api-feedback.md (server should include attempts in the list).
func listAttempts(ctx context.Context, st *state) ([]Attempt, error) {
	var out struct{ Items []struct {
		ID string `json:"id"`
	} `json:"items"` }
	if err := st.client.Do(ctx, "GET", "/api/work", &out); err != nil {
		return nil, err
	}
	const maxScan = 30
	limit := maxScan
	if len(out.Items) < limit {
		limit = len(out.Items)
	}
	var atts []Attempt
	for i := 0; i < limit; i++ {
		var w struct {
			ID       string   `json:"id"`
			Attempts []Attempt `json:"attempts"`
		}
		if err := st.client.Do(ctx, "GET", "/api/work/"+out.Items[i].ID, &w); err != nil {
			continue
		}
		for _, a := range w.Attempts {
			a.WorkID = w.ID
			atts = append(atts, a)
		}
	}
	return atts, nil
}

func newAttemptListCmd() *cobra.Command {
	var workID, repo, status string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List attempts",
		Example: "  sy attempt list\n  sy attempt list --work wk_123\n  sy attempt list --status running --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			atts, err := listAttempts(cmd.Context(), st)
			if err != nil {
				return err
			}
			if workID != "" {
				atts = filterAttempts(atts, func(a Attempt) bool { return a.WorkID == workID })
			}
			if repo != "" {
				atts = filterAttempts(atts, func(a Attempt) bool { return a.Repo == repo })
			}
			if status != "" {
				atts = filterAttempts(atts, func(a Attempt) bool { return strings.EqualFold(a.Status, status) })
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"items": atts})
			}
			rows := make([][]string, 0, len(atts))
			for _, a := range atts {
				rows = append(rows, []string{a.ID, a.WorkID, a.Repo, a.Branch, a.Status, timeAgo(a.UpdatedAt)})
			}
			st.renderer.Table([]string{"ATTEMPT", "WORK", "REPO", "BRANCH", "STATUS", "UPDATED"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&workID, "work", "", "filter by Work id")
	cmd.Flags().StringVar(&repo, "repo", "", "filter by repository")
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	return cmd
}

func newAttemptViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <id>",
		Short:   "Show an attempt with its Work, execution and PR context",
		Example: "  sy attempt view wk_f50bf064cf8940ff --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			atts, err := listAttempts(cmd.Context(), st)
			if err != nil {
				return err
			}
			var a *Attempt
			for i := range atts {
				if atts[i].ID == args[0] {
					a = &atts[i]
					break
				}
			}
			if a == nil {
				return fmt.Errorf("attempt %s not found", args[0])
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"attempt": a, "work_id": a.WorkID})
			}
			st.renderer.KV([][2]string{
				{"Attempt", a.ID},
				{"Work", a.WorkID},
				{"Repository", a.Repo},
				{"Branch", a.Branch},
				{"Status", statusSym(a.Status) + " " + a.Status},
				{"Created", relative(a.CreatedAt)},
			})
			if a.Message != "" {
				st.renderer.Print("\n" + a.Message)
			}
			return nil
		},
	}
}

func newAttemptRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "run <id>",
		Short:   "Run the implementer agent against an attempt",
		Example: "  sy attempt run wk_f50bf064cf8940ff",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out map[string]any
			if err := st.client.Do(cmd.Context(), "POST", "/api/attempts/"+args[0]+"/run", &out); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			st.renderer.KV([][2]string{
				{"Attempt", str(out, "attempt_id")},
				{"Execution", str(out, "execution")},
				{"Status", str(out, "status")},
				{"New SHA", str(out, "new_sha")},
			})
			return nil
		},
	}
}

func filterAttempts(items []Attempt, keep func(Attempt) bool) []Attempt {
	out := make([]Attempt, 0, len(items))
	for _, a := range items {
		if keep(a) {
			out = append(out, a)
		}
	}
	return out
}

// relative returns a short relative timestamp for human output.
func relative(ts string) string {
	if ts == "" {
		return ""
	}
	return timeAgo(ts)
}

func statusSym(s string) string {
	switch strings.ToLower(s) {
	case "completed", "pass", "resolved", "done":
		return "✓"
	case "running", "pending", "queued":
		return "●"
	case "blocked", "conflict", "semantic_conflict", "failed":
		return "×"
	default:
		return "·"
	}
}