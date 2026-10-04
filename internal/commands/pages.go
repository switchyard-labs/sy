package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/switchyard-labs/sy/internal/api"
	"net/url"
)

func newPagesCmd() *cobra.Command {
	var repo, kind string
	group := &cobra.Command{Use: "pages", Short: "Build and promote immutable repository or owner Pages sites"}
	group.PersistentFlags().StringVarP(&repo, "repo", "R", "", "owner/repo (default: checkout)")
	group.PersistentFlags().StringVar(&kind, "kind", "project", "project or owner; backing repository is explicit")
	rootFor := func(cmd *cobra.Command) (string, error) {
		if kind != "project" && kind != "owner" {
			return "", fmt.Errorf("kind must be project or owner")
		}
		name := repo
		if name == "" {
			var err error
			name, err = currentRepoName(cmd.Context())
			if err != nil {
				return "", err
			}
		}
		r, err := resolveRepo(cmd, name)
		if err != nil {
			return "", err
		}
		return canonicalPath(r) + "/pages", nil
	}
	emit := func(cmd *cobra.Command, result map[string]any) error {
		st := stateFrom(cmd)
		if st.renderer.JSONMode {
			return st.renderer.Emit(result)
		}
		if items, ok := result["items"].([]any); ok {
			rows := [][]string{}
			for _, item := range items {
				d, ok := item.(map[string]any)
				if !ok {
					continue
				}
				state, _ := d["state"].(map[string]any)
				rows = append(rows, []string{fmt.Sprint(d["id"]), fmt.Sprint(state["status"]), fmt.Sprint(d["source_ref"]), fmt.Sprint(d["source_sha"]), fmt.Sprint(d["run_id"])})
			}
			st.renderer.Table([]string{"DEPLOYMENT", "STATUS", "REF", "SHA", "ACTION"}, rows)
			return nil
		}
		for _, key := range []string{"site", "version", "production", "production_status", "run_id", "deployment_id", "source_sha", "source_ref", "action", "public_hosting"} {
			if v, ok := result[key]; ok {
				st.renderer.KV([][2]string{{key, fmt.Sprint(v)}})
			}
		}
		return nil
	}
	for _, entry := range []struct{ name, path string }{{"status", "/config"}, {"deployments", "/deployments"}} {
		e := entry
		var page int
		cmd := &cobra.Command{Use: e.name, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
			root, err := rootFor(cmd)
			if err != nil {
				return err
			}
			var result map[string]any
			err = stateFrom(cmd).client.Do(cmd.Context(), "GET", root+e.path+"?kind="+kind+"&page="+fmt.Sprint(page), &result)
			if err != nil {
				return err
			}
			return emit(cmd, result)
		}}
		cmd.Flags().IntVar(&page, "page", 1, "deployment page")
		group.AddCommand(cmd)
	}
	view := &cobra.Command{Use: "view <deployment>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "GET", root+"/deployments/"+url.PathEscape(args[0]), &result); err != nil {
			return err
		}
		if !stateFrom(cmd).renderer.JSONMode {
			result = map[string]any{"items": []any{result}}
		}
		return emit(cmd, result)
	}}
	group.AddCommand(view)
	var version, operation string
	deploy := &cobra.Command{Use: "deploy", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "POST", root+"/deploy?kind="+kind, &result, api.WithBody(map[string]any{"version": version, "operation_id": operation})); err != nil {
			return err
		}
		return emit(cmd, result)
	}}
	deploy.Flags().StringVar(&version, "version", "", "expected config version from status")
	deploy.Flags().StringVar(&operation, "operation-id", "", "stable 16–64 character operation ID; reuse only for retry of identical input")
	_ = deploy.MarkFlagRequired("version")
	_ = deploy.MarkFlagRequired("operation-id")
	group.AddCommand(deploy)
	for _, name := range []string{"promote", "rollback"} {
		action := name
		var generation int64
		var op string
		cmd := &cobra.Command{Use: action + " <deployment>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			root, err := rootFor(cmd)
			if err != nil {
				return err
			}
			var result map[string]any
			if err = stateFrom(cmd).client.Do(cmd.Context(), "POST", root+"/deployments/"+url.PathEscape(args[0])+"/"+action, &result, api.WithBody(map[string]any{"generation": generation, "operation_id": op})); err != nil {
				return err
			}
			return emit(cmd, result)
		}}
		cmd.Flags().Int64Var(&generation, "generation", 0, "expected production generation from deployments")
		cmd.Flags().StringVar(&op, "operation-id", "", "stable operation ID for explicit retries")
		_ = cmd.MarkFlagRequired("generation")
		_ = cmd.MarkFlagRequired("operation-id")
		group.AddCommand(cmd)
	}
	var configVersion, ref, working, build, output, spa string
	var enabled, ack bool
	configure := &cobra.Command{Use: "configure", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		body := map[string]any{"version": configVersion, "enabled": enabled, "config": map[string]any{"ref": ref, "working_directory": working, "build_command": build, "output_directory": output, "spa_fallback": spa, "public_acknowledged": ack}}
		if err = stateFrom(cmd).client.Do(cmd.Context(), "PATCH", root+"/config?kind="+kind, &result, api.WithBody(body)); err != nil {
			return err
		}
		return emit(cmd, result)
	}}
	configure.Flags().StringVar(&configVersion, "version", "", "expected version; empty for initial configuration")
	configure.Flags().StringVar(&ref, "ref", "main", "branch or qualified branch/tag ref")
	configure.Flags().StringVar(&working, "directory", ".", "working directory")
	configure.Flags().StringVar(&build, "build", "", "build command")
	configure.Flags().StringVar(&spa, "spa-fallback", "", "explicit SPA fallback file relative to output")
	configure.Flags().StringVar(&output, "output", "public", "output directory relative to working directory")
	configure.Flags().BoolVar(&enabled, "enabled", true, "enable new builds (does not remove existing production)")
	configure.Flags().BoolVar(&ack, "acknowledge-public", false, "acknowledge publishing output from private source")
	_ = configure.MarkFlagRequired("build")
	group.AddCommand(configure)
	return group
}
