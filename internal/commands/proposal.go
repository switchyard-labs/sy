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

	var updateVersion string
	update := &cobra.Command{Use: "update <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		input := map[string]any{"version": updateVersion}
		for _, field := range []string{"title", "description", "type", "priority", "severity"} {
			if cmd.Flags().Changed(field) {
				input[field], _ = cmd.Flags().GetString(field)
			}
		}
		if len(input) == 1 {
			return fmt.Errorf("provide at least one metadata field")
		}
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "PATCH", root+"/"+url.PathEscape(args[0]), &result, api.WithBody(input)); err != nil {
			return err
		}
		return emit(cmd, result)
	}}
	update.Flags().StringVar(&updateVersion, "version", "", "expected Proposal version")
	for _, field := range []string{"title", "description", "type", "priority", "severity"} {
		update.Flags().String(field, "", "new "+field)
	}
	_ = update.MarkFlagRequired("version")
	group.AddCommand(update)
	var commentVersion, commentBody string
	comment := &cobra.Command{Use: "comment <id>", Short: "Add discussion with expected version", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "POST", root+"/"+url.PathEscape(args[0])+"/comments", &result, api.WithBody(map[string]any{"version": commentVersion, "body": commentBody})); err != nil {
			return err
		}
		if stateFrom(cmd).renderer.JSONMode {
			return stateFrom(cmd).renderer.Emit(result)
		}
		stateFrom(cmd).renderer.Print("Comment added")
		return nil
	}}
	comment.Flags().StringVar(&commentVersion, "version", "", "expected Proposal version")
	comment.Flags().StringVar(&commentBody, "body", "", "comment text")
	_ = comment.MarkFlagRequired("version")
	_ = comment.MarkFlagRequired("body")
	group.AddCommand(comment)
	var graphVersion, relation, targetKind string
	link := &cobra.Command{Use: "link <id> <target-id>", Short: "Link a Proposal or existing Work", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "POST", root+"/"+url.PathEscape(args[0])+"/links", &result, api.WithBody(map[string]any{"version": graphVersion, "relation": relation, "target_kind": targetKind, "target_id": args[1]})); err != nil {
			return err
		}
		if stateFrom(cmd).renderer.JSONMode {
			return stateFrom(cmd).renderer.Emit(result)
		}
		stateFrom(cmd).renderer.Print("Relationship saved")
		return nil
	}}
	link.Flags().StringVar(&graphVersion, "version", "", "expected graph version from links")
	link.Flags().StringVar(&relation, "relation", "related", "related, duplicates, supersedes, superseded_by or work")
	link.Flags().StringVar(&targetKind, "target-kind", "proposal", "proposal or work")
	_ = link.MarkFlagRequired("version")
	group.AddCommand(link)
	links := &cobra.Command{Use: "links <id>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result map[string]any
		if err = stateFrom(cmd).client.Do(cmd.Context(), "GET", root+"/"+url.PathEscape(args[0])+"/links", &result); err != nil {
			return err
		}
		if stateFrom(cmd).renderer.JSONMode {
			return stateFrom(cmd).renderer.Emit(result)
		}
		stateFrom(cmd).renderer.Print(fmt.Sprint(result))
		return nil
	}}
	group.AddCommand(links)
	return group
}
