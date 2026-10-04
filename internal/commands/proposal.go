package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/switchyard-labs/sy/internal/api"
	"net/url"
)

func newProposalCmd() *cobra.Command {
	var repo string
	group := &cobra.Command{Use: "proposal", Short: "Optional repository intake and decisions"}
	group.PersistentFlags().StringVarP(&repo, "repo", "R", "", "owner/repo (default: checkout)")
	rootFor := func(cmd *cobra.Command) (string, error) {
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
		return canonicalPath(r) + "/proposals", nil
	}
	emit := func(cmd *cobra.Command, result map[string]any) error {
		st := stateFrom(cmd)
		if st.renderer.JSONMode {
			return st.renderer.Emit(result)
		}
		if items, ok := result["items"].([]any); ok {
			rows := [][]string{}
			for _, raw := range items {
				p, ok := raw.(map[string]any)
				if ok {
					rows = append(rows, []string{fmt.Sprint(p["id"]), fmt.Sprint(p["type"]), fmt.Sprint(p["state"]), fmt.Sprint(p["title"])})
				}
			}
			st.renderer.Table([]string{"ID", "TYPE", "STATE", "TITLE"}, rows)
			return nil
		}
		st.renderer.KV([][2]string{{"ID", fmt.Sprint(result["id"])}, {"Title", fmt.Sprint(result["title"])}, {"State", fmt.Sprint(result["state"])}, {"Version", fmt.Sprint(result["version"])}})
		if body, ok := result["description"].(string); ok {
			st.renderer.Print(body)
		}
		return nil
	}
	var state, kind string
	var page int
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		query := url.Values{"state": {state}, "type": {kind}, "page": {fmt.Sprint(page)}}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "GET", root+"?"+query.Encode(), &result); err != nil {
			return err
		}
		return emit(cmd, result)
	}}
	list.Flags().StringVar(&state, "state", "", "state filter")
	list.Flags().StringVar(&kind, "type", "", "type filter")
	list.Flags().IntVar(&page, "page", 1, "page number")
	view := &cobra.Command{Use: "view <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "GET", root+"/"+url.PathEscape(args[0]), &result); err != nil {
			return err
		}
		return emit(cmd, result)
	}}
	var title, body, createType string
	create := &cobra.Command{Use: "create", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "POST", root, &result, api.WithBody(map[string]any{"title": title, "description": body, "type": createType})); err != nil {
			return err
		}
		return emit(cmd, result)
	}}
	create.Flags().StringVar(&title, "title", "", "title")
	create.Flags().StringVar(&body, "body", "", "description")
	create.Flags().StringVar(&createType, "type", "Other", "Proposal type")
	_ = create.MarkFlagRequired("title")
	group.AddCommand(list, view, create)
	for _, transition := range []struct{ name, state, outcome string }{{"accept", "accepted", ""}, {"defer", "deferred", ""}, {"reject", "closed", "rejected"}, {"complete", "closed", "completed"}, {"reopen", "open", ""}} {
		tr := transition
		var version, reason string
		cmd := &cobra.Command{Use: tr.name + " <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			root, err := rootFor(cmd)
			if err != nil {
				return err
			}
			var result map[string]any
			err = stateFrom(cmd).client.Do(cmd.Context(), "PATCH", root+"/"+url.PathEscape(args[0]), &result, api.WithBody(map[string]any{"version": version, "state": tr.state, "closure_outcome": tr.outcome, "reason": reason}))
			if err != nil {
				return err
			}
			return emit(cmd, result)
		}}
		cmd.Flags().StringVar(&version, "version", "", "expected version from view")
		cmd.Flags().StringVar(&reason, "reason", "", "decision reason")
		_ = cmd.MarkFlagRequired("version")
		group.AddCommand(cmd)
	}
	var version, operation, workTitle, workBody, workKind string
	work := &cobra.Command{Use: "work <id>", Short: "Create linked Work with a retry-safe operation ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		err = stateFrom(cmd).client.Do(cmd.Context(), "POST", root+"/"+url.PathEscape(args[0])+"/work", &result, api.WithBody(map[string]any{"version": version, "operation_id": operation, "title": workTitle, "body": workBody, "kind": workKind}))
		if err != nil {
			return err
		}
		if stateFrom(cmd).renderer.JSONMode {
			return stateFrom(cmd).renderer.Emit(result)
		}
		stateFrom(cmd).renderer.Print(fmt.Sprint(result["work"]))
		return nil
	}}
	work.Flags().StringVar(&version, "version", "", "expected Proposal version")
	work.Flags().StringVar(&operation, "operation-id", "", "unique operation ID; reuse only for same input")
	work.Flags().StringVar(&workTitle, "title", "", "Work title (defaults to Proposal)")
	work.Flags().StringVar(&workBody, "body", "", "Work description")
	work.Flags().StringVar(&workKind, "kind", "feature", "Work kind")
	_ = work.MarkFlagRequired("version")
	_ = work.MarkFlagRequired("operation-id")
	group.AddCommand(work)
	return group
}
