package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWorkflowCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "workflow", Short: "Durable engineering workflows", Aliases: []string{"wf"}}
	cmd.AddCommand(newWorkflowListCmd(), newWorkflowViewCmd(), newWorkflowRunCmd(), newWorkflowRunsCmd())
	return cmd
}

func newWorkflowListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List workflow definitions",
		Example: "  sy workflow list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/workflows", &out); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			rows := make([][]string, 0, len(out.Items))
			for _, it := range out.Items {
				rows = append(rows, []string{str(it, "id"), str(it, "name"), timeAgo(str(it, "created_at"))})
			}
			st.renderer.Table([]string{"ID", "NAME", "CREATED"}, rows)
			return nil
		},
	}
}

func newWorkflowViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <id-or-name>",
		Short:   "Show a workflow definition",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/workflows", &out); err != nil {
				return err
			}
			var w map[string]any
			for _, it := range out.Items {
				if str(it, "id") == args[0] || str(it, "name") == args[0] {
					w = it
					break
				}
			}
			if w == nil {
				return fmt.Errorf("workflow %s not found", args[0])
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(w)
			}
			st.renderer.KV([][2]string{{"ID", str(w, "id")}, {"Name", str(w, "name")}, {"Created", str(w, "created_at")}})
			st.renderer.Print("\nScript:")
			st.renderer.Print(str(w, "script"))
			return nil
		},
	}
}

func newWorkflowRunCmd() *cobra.Command {
	var repo, branch string
	cmd := &cobra.Command{
		Use:     "run <id-or-name>",
		Short:   "Run a workflow",
		Example: "  sy workflow run demo-railctl-flow --repo demo-workflow --branch wf1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/workflows", &out); err != nil {
				return err
			}
			var id string
			for _, it := range out.Items {
				if str(it, "id") == args[0] || str(it, "name") == args[0] {
					id = str(it, "id")
					break
				}
			}
			if id == "" {
				return fmt.Errorf("workflow %s not found", args[0])
			}
			params := map[string]any{}
			if repo != "" {
				params["repo"] = repo
			}
			if branch != "" {
				params["branch"] = branch
			}
			var run struct{ ID string `json:"id"` }
			if err := st.client.Do(cmd.Context(), "POST", "/api/workflows/"+id+"/run", &run, apiBody(map[string]any{"params": params})); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"run_id": run.ID, "workflow_id": id})
			}
			st.renderer.Print(fmt.Sprintf("started run %s", run.ID))
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "repo param (ctx.params.repo)")
	cmd.Flags().StringVar(&branch, "branch", "", "branch param (ctx.params.branch)")
	return cmd
}

func newWorkflowRunsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "runs",
		Short:   "List workflow runs",
		Example: "  sy workflow runs --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/workflow_runs", &out); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			rows := make([][]string, 0, len(out.Items))
			for _, it := range out.Items {
				rows = append(rows, []string{str(it, "id"), str(it, "workflow_id"), str(it, "status"), str(it, "step_count"), str(it, "error")})
			}
			st.renderer.Table([]string{"RUN", "WORKFLOW", "STATUS", "STEPS", "ERROR"}, rows)
			return nil
		},
	}
}