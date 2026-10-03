package commands

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/switchyard-labs/sy/internal/api"
)

type actionStep struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
type actionJob struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Status string       `json:"status"`
	Steps  []actionStep `json:"steps"`
}
type actionRun struct {
	ID        string `json:"id"`
	SHA       string `json:"source_sha"`
	Ref       string `json:"ref"`
	Actor     string `json:"actor"`
	Trigger   string `json:"trigger"`
	CreatedAt string `json:"created_at"`
	State     struct {
		Status   string `json:"status"`
		Manifest struct {
			Run struct {
				Jobs []actionJob `json:"jobs"`
			} `json:"run"`
			Jobs []actionJob `json:"jobs"`
		} `json:"manifest"`
	} `json:"state"`
}

func actionTerminal(s string) bool {
	return s == "success" || s == "failure" || s == "cancelled" || s == "timed_out"
}
func actionRoot(cmd *cobra.Command, repo string) (string, error) {
	if repo == "" {
		var err error
		repo, err = currentRepoName(cmd.Context())
		if err != nil {
			return "", err
		}
	}
	r, err := resolveRepo(cmd, repo)
	if err != nil {
		return "", err
	}
	return canonicalPath(r) + "/actions", nil
}
func newActionsCmd() *cobra.Command {
	var repo string
	group := &cobra.Command{Use: "actions", Short: "Repository checks and captured Action output"}
	group.PersistentFlags().StringVarP(&repo, "repo", "R", "", "canonical owner/repo (default: checkout)")
	list := &cobra.Command{Use: "list", Args: cobra.NoArgs, Short: "List repository Action runs", RunE: func(cmd *cobra.Command, args []string) error {
		root, err := actionRoot(cmd, repo)
		if err != nil {
			return err
		}
		st := stateFrom(cmd)
		var result struct {
			Items []actionRun `json:"items"`
		}
		if err = st.client.Do(cmd.Context(), "GET", root, &result); err != nil {
			return err
		}
		sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].CreatedAt > result.Items[j].CreatedAt })
		if st.renderer.JSONMode {
			return st.renderer.Emit(result)
		}
		rows := [][]string{}
		for _, r := range result.Items {
			rows = append(rows, []string{r.ID, r.State.Status, r.Ref, r.SHA, r.Trigger, timeAgo(r.CreatedAt)})
		}
		st.renderer.Table([]string{"RUN", "STATUS", "REF", "COMMIT", "TRIGGER", "STARTED"}, rows)
		return nil
	}}
	view := &cobra.Command{Use: "view <run-id>", Args: cobra.ExactArgs(1), Short: "Show source, jobs and checks for a run", RunE: func(cmd *cobra.Command, args []string) error {
		root, err := actionRoot(cmd, repo)
		if err != nil {
			return err
		}
		st := stateFrom(cmd)
		var raw map[string]any
		if err = st.client.Do(cmd.Context(), "GET", root+"/"+url.PathEscape(args[0]), &raw); err != nil {
			return err
		}
		if st.renderer.JSONMode {
			return st.renderer.Emit(raw)
		}
		var result struct {
			Run actionRun `json:"run"`
		}
		encoded, err := json.Marshal(raw)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(encoded, &result); err != nil {
			return err
		}
		r := result.Run
		st.renderer.KV([][2]string{{"Run", r.ID}, {"Status", r.State.Status}, {"Commit", r.SHA}, {"Ref", r.Ref}, {"Trigger", r.Trigger}, {"Actor", r.Actor}})
		rows := [][]string{}
		for _, j := range r.State.Manifest.Jobs {
			rows = append(rows, []string{j.ID, j.Status})
		}
		st.renderer.Table([]string{"JOB", "STATUS"}, rows)
		return nil
	}}
	var sha, ref, pr string
	run := &cobra.Command{Use: "run", Args: cobra.NoArgs, Short: "Run the approved action at an exact commit", RunE: func(cmd *cobra.Command, args []string) error {
		root, err := actionRoot(cmd, repo)
		if err != nil {
			return err
		}
		if sha == "" && pr == "" {
			return fmt.Errorf("supply --sha or --pr")
		}
		return actionMutation(cmd, root+"/dispatch", map[string]any{"sha": sha, "ref": ref, "pr_id": pr, "request_id": actionRequestID()})
	}}
	run.Flags().StringVar(&sha, "sha", "", "exact40hex commit SHA")
	run.Flags().StringVar(&ref, "ref", "refs/heads/main", "full Git ref")
	run.Flags().StringVar(&pr, "pr", "", "resolve the current pull request source")
	var failed bool
	rerun := &cobra.Command{Use: "rerun <run-id>", Args: cobra.ExactArgs(1), Short: "Rerun a terminal action at its exact commit", RunE: func(cmd *cobra.Command, args []string) error {
		root, err := actionRoot(cmd, repo)
		if err != nil {
			return err
		}
		mode := "all"
		if failed {
			mode = "failed"
		}
		return actionMutation(cmd, root+"/"+url.PathEscape(args[0])+"/rerun", map[string]any{"mode": mode, "request_id": actionRequestID()})
	}}
	rerun.Flags().BoolVar(&failed, "failed", false, "rerun failed jobs, reusing passing exact-source jobs")
	cancel := &cobra.Command{Use: "cancel <run-id>", Args: cobra.ExactArgs(1), Short: "Explicitly request cancellation of a run", RunE: func(cmd *cobra.Command, args []string) error {
		root, err := actionRoot(cmd, repo)
		if err != nil {
			return err
		}
		return actionMutation(cmd, root+"/"+url.PathEscape(args[0])+"/cancel", map[string]any{})
	}}
	var follow bool
	var step string
	logs := &cobra.Command{Use: "logs <run-id>", Args: cobra.ExactArgs(1), Short: "Read captured step output; follow until run completion", RunE: func(cmd *cobra.Command, args []string) error {
		root, err := actionRoot(cmd, repo)
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()
		err = followActionLogs(ctx, stateFrom(cmd), root, args[0], step, follow)
		if ctx.Err() != nil {
			return nil
		}
		return err
	}}
	logs.Flags().BoolVar(&follow, "follow", false, "wait for captured output and terminal run; Ctrl-C stops following only")
	logs.Flags().StringVar(&step, "step", "", "only this jobID-stepID")
	group.AddCommand(list, view, run, rerun, cancel, logs)
	return group
}
func actionRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func actionMutation(cmd *cobra.Command, path string, body any) error {
	st := stateFrom(cmd)
	var result map[string]any
	if err := st.client.Do(cmd.Context(), "POST", path, &result, apiBody(body)); err != nil {
		return err
	}
	if st.renderer.JSONMode {
		return st.renderer.Emit(result)
	}
	st.renderer.Print(fmt.Sprintf("Action %v: %v", result["id"], result))
	return nil
}

type actionLogLine struct {
	ID     int    `json:"id"`
	Stream string `json:"stream"`
	Text   string `json:"text"`
}

func followActionLogs(ctx context.Context, st *state, root, id, onlyStep string, follow bool) error {
	cursor := map[string]int{}
	drained := map[string]bool{}
	for {
		var detail struct {
			Run actionRun `json:"run"`
		}
		path := root + "/" + url.PathEscape(id)
		if err := st.client.Do(ctx, "GET", path, &detail); err != nil {
			return err
		}
		found := false
		pending := false
		for _, job := range detail.Run.State.Manifest.Run.Jobs {
			for _, step := range job.Steps {
				label := job.ID + "-" + step.ID
				if onlyStep != "" && label != onlyStep {
					continue
				}
				found = true
				if actionTerminal(detail.Run.State.Status) {
					started := false
					for _, executedJob := range detail.Run.State.Manifest.Jobs {
						if executedJob.ID != job.ID {
							continue
						}
						for _, executedStep := range executedJob.Steps {
							if executedStep.ID == step.ID && executedStep.Status != "" && executedStep.Status != "queued" && executedStep.Status != "skipped" {
								started = true
							}
						}
					}
					if !started {
						continue
					}
				}
				if drained[label] {
					continue
				}
				for {
					var page struct {
						Lines    []actionLogLine `json:"lines"`
						Cursor   int             `json:"cursor"`
						Complete bool            `json:"complete"`
						Pending  bool            `json:"pending"`
						Error    string          `json:"error"`
					}
					err := st.client.Do(ctx, "GET", path+"/logs", &page, api.WithQuery("step", label), api.WithQuery("cursor", fmt.Sprint(cursor[label])))
					if err != nil {
						return err
					}
					if page.Pending {
						pending = true
						break
					}
					if page.Error != "" {
						return fmt.Errorf("%s", page.Error)
					}
					for _, line := range page.Lines {
						if st.renderer.JSONMode {
							if err = st.renderer.Emit(map[string]any{"run_id": id, "step": label, "line": line}); err != nil {
								return err
							}
						} else {
							st.renderer.Print(fmt.Sprintf("[%s %s] %s", label, line.Stream, line.Text))
						}
					}
					if page.Cursor < cursor[label] || (!page.Complete && page.Cursor == cursor[label]) {
						return fmt.Errorf("invalid log cursor")
					}
					cursor[label] = page.Cursor
					if page.Complete {
						drained[label] = true
						break
					}
				}
			}
		}
		if onlyStep != "" && !found {
			return fmt.Errorf("step %q not found", onlyStep)
		}
		if !follow {
			if pending {
				return fmt.Errorf("capture pending; use --follow")
			}
			return nil
		}
		if actionTerminal(detail.Run.State.Status) && !pending {
			return nil
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
