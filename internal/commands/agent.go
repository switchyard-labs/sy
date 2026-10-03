package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "agent", Short: "Agent roles and executions"}
	cmd.AddCommand(newAgentRolesCmd(), newAgentExecutionsCmd(), newAgentExecutionCmd())
	return cmd
}

func newAgentRolesCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "roles",
		Short:   "List agent roles and capabilities",
		Example: "  sy agent roles --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct {
				Items []struct {
					Name         string   `json:"name"`
					Capabilities []string `json:"capabilities"`
					Profile      string   `json:"profile"`
				} `json:"items"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/roles", &out); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			rows := make([][]string, 0, len(out.Items))
			for _, r := range out.Items {
				rows = append(rows, []string{r.Name, fmt.Sprint(r.Capabilities), r.Profile})
			}
			st.renderer.Table([]string{"ROLE", "CAPABILITIES", "PROFILE"}, rows)
			return nil
		},
	}
}

func newAgentExecutionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "executions",
		Short:   "List agent executions",
		Example: "  sy agent executions --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/executions", &out); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			rows := make([][]string, 0, len(out.Items))
			for _, it := range out.Items {
				rows = append(rows, []string{str(it, "id"), str(it, "role"), str(it, "adapter"), str(it, "status"), str(it, "attempt_id")})
			}
			st.renderer.Table([]string{"EXECUTION", "ROLE", "ADAPTER", "STATUS", "ATTEMPT"}, rows)
			return nil
		},
	}
}

func newAgentExecutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "execution <id>",
		Short:   "Show an agent execution",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/executions", &out); err != nil {
				return err
			}
			var ex map[string]any
			for _, it := range out.Items {
				if str(it, "id") == args[0] {
					ex = it
					break
				}
			}
			if ex == nil {
				return fmt.Errorf("execution %s not found", args[0])
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(ex)
			}
			st.renderer.KV([][2]string{
				{"ID", str(ex, "id")},
				{"Role", str(ex, "role")},
				{"Adapter", str(ex, "adapter")},
				{"Status", str(ex, "status")},
				{"Attempt", str(ex, "attempt_id")},
				{"Started", str(ex, "started_at")},
				{"Output", str(ex, "output")},
			})
			return nil
		},
	}
}

func newOrgCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "org", Short: "Organizations"}
	cmd.AddCommand(newOrgListCmd(), newOrgViewCmd())
	return cmd
}

func newOrgListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List organizations",
		Example: "  sy org list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/orgs", &out); err != nil {
				return err
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(out)
			}
			rows := make([][]string, 0, len(out.Items))
			for _, it := range out.Items {
				rows = append(rows, []string{str(it, "id"), str(it, "display_name"), str(it, "name"), str(it, "description")})
			}
			st.renderer.Table([]string{"ID", "DISPLAY", "NAME", "DESCRIPTION"}, rows)
			return nil
		},
	}
}

func newOrgViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "view <id>",
		Short:   "Show an organization",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct{ Items []map[string]any `json:"items"` }
			if err := st.client.Do(cmd.Context(), "GET", "/api/orgs", &out); err != nil {
				return err
			}
			var o map[string]any
			for _, it := range out.Items {
				if str(it, "id") == args[0] || str(it, "name") == args[0] {
					o = it
					break
				}
			}
			if o == nil {
				return fmt.Errorf("org %s not found", args[0])
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(o)
			}
			st.renderer.KV([][2]string{
				{"ID", str(o, "id")},
				{"Name", str(o, "name")},
				{"Display", str(o, "display_name")},
				{"Description", str(o, "description")},
				{"Location", str(o, "location")},
			})
			return nil
		},
	}
}