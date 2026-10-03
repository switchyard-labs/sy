package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/switchyard-labs/sy/internal/gitutil"
	"github.com/switchyard-labs/sy/internal/output"
)

// Repo is the domain view of a repository.
type Repo struct {
	Owner        string `json:"owner,omitempty"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	DefaultBranch string `json:"default_branch"`
	Remote       string `json:"remote"`
	Source       string `json:"source,omitempty"`
	ReadOnly     bool   `json:"read_only"`
	Registered   bool   `json:"registered"`
	Visibility   string `json:"visibility,omitempty"`
}

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Work with repositories",
	}
	cmd.AddCommand(newRepoListCmd(), newRepoViewCmd(), newRepoCloneCmd())
	return cmd
}

func newRepoListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List repositories",
		Example: "  sy repo list\n  sy repo list --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			var out struct {
				Items []struct {
					Name          string `json:"name"`
					DefaultBranch string `json:"default_branch"`
					Remote        string `json:"remote"`
					ReadOnly      bool   `json:"read_only"`
					Registered    bool   `json:"registered"`
					RegisteredAt  string `json:"registered_at,omitempty"`
				} `json:"items"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/repos", &out); err != nil {
				return err
			}
			repos := make([]Repo, 0, len(out.Items))
			rows := make([][]string, 0, len(out.Items))
			for _, it := range out.Items {
				owner, name := gitutil.DetectOwnerRepo(it.Remote)
				vis := "private"
				if !it.ReadOnly {
					vis = "public"
				}
				repos = append(repos, Repo{Owner: owner, Name: name, DefaultBranch: it.DefaultBranch, Remote: it.Remote, ReadOnly: it.ReadOnly, Registered: it.Registered, Visibility: vis})
				rows = append(rows, []string{owner, name, vis, it.DefaultBranch, output.TimeAgo(it.RegisteredAt)})
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(map[string]any{"items": repos})
			}
			st.renderer.Table([]string{"OWNER", "REPOSITORY", "VISIBILITY", "DEFAULT", "UPDATED"}, rows)
			return nil
		},
	}
}

func newRepoViewCmd() *cobra.Command {
	var flagRepo string
	cmd := &cobra.Command{
		Use:     "view [owner/repo]",
		Short:   "Show repository details",
		Example: "  sy repo view demo-basic\n  sy repo view alice/demo-basic\n  sy repo view --json",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			name := flagRepo
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				n, err := currentRepoName(cmd.Context())
				if err != nil {
					return err
				}
				name = n
			}
			// canonical metadata first (owner/repo); fall back to legacy flat name.
			var meta struct {
				Items []struct {
					Owner string `json:"owner"`
					Name  string `json:"name"`
					Description string `json:"description,omitempty"`
					Visibility string `json:"visibility,omitempty"`
					DefaultBranch string `json:"default_branch,omitempty"`
					Remote string `json:"remote,omitempty"`
				} `json:"items"`
			}
			_ = st.client.Do(cmd.Context(), "GET", "/api/repositories", &meta)
			var found *Repo
			for _, it := range meta.Items {
				if it.Name == name || (it.Owner+"/"+it.Name) == name {
					found = &Repo{Owner: it.Owner, Name: it.Name, Description: it.Description, Visibility: it.Visibility, DefaultBranch: it.DefaultBranch, Remote: it.Remote, Registered: true}
					break
				}
			}
			if found == nil {
				var r struct {
					Name          string `json:"name"`
					DefaultBranch string `json:"default_branch"`
					Remote        string `json:"remote"`
					Source        string `json:"source"`
					ReadOnly      bool   `json:"read_only"`
				}
				if err := st.client.Do(cmd.Context(), "GET", "/api/repos/"+name, &r); err != nil {
					return err
				}
				owner, rname := gitutil.DetectOwnerRepo(r.Remote)
				vis := "private"
				if !r.ReadOnly {
					vis = "public"
				}
				found = &Repo{Owner: owner, Name: rname, DefaultBranch: r.DefaultBranch, Remote: r.Remote, Source: r.Source, ReadOnly: r.ReadOnly, Visibility: vis}
			}
			if st.renderer.JSONMode {
				return st.renderer.Emit(found)
			}
			st.renderer.KV([][2]string{
				{"Name", found.Name},
				{"Visibility", found.Visibility},
				{"Default branch", found.DefaultBranch},
				{"Clone", found.Remote},
				{"Description", found.Description},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&flagRepo, "repo", "", "owner/repo or repo name")
	return cmd
}

func newRepoCloneCmd() *cobra.Command {
	var flagRepo, token, dir string
	cmd := &cobra.Command{
		Use:     "clone [owner/repo] [directory]",
		Short:   "Clone a Switchyard repository using ordinary Git",
		Example: "  sy repo clone demo-basic\n  sy repo clone alice/demo-basic ./demo",
		Args:    cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := stateFrom(cmd)
			name := flagRepo
			if len(args) > 0 {
				name = args[0]
			}
			if name == "" {
				n, err := currentRepoName(cmd.Context())
				if err != nil {
					return err
				}
				name = n
			}
			if len(args) > 1 {
				dir = args[1]
			}
			var r struct {
				Name          string `json:"name"`
				DefaultBranch string `json:"default_branch"`
				Remote        string `json:"remote"`
			}
			if err := st.client.Do(cmd.Context(), "GET", "/api/repos/"+name, &r); err != nil {
				return err
			}
			if token == "" {
				token = os.Getenv("SY_GIT_TOKEN")
			}
			if dir == "" {
				dir = r.Name
			}
			env := []string{gitutil.CredentialHelperOff(), "GIT_CONFIG_NOSYSTEM=1"}
			credFile := ""
			cleanup := func() {}
			if token != "" {
				// credential via GIT_CONFIG_GLOBAL temp file: never in argv,
				// never in .git/config
				credFile, cleanup, _ = gitutil.CredentialFile(token)
				defer cleanup()
				if credFile != "" {
					env = append(env, "GIT_CONFIG_GLOBAL="+credFile)
				}
			} else {
				st.renderer.Print("note: Switchyard API does not yet expose a scoped git credential; cloning anonymously (may fail). Provide SY_GIT_TOKEN or --token.")
			}
			parent := filepath.Dir(dir)
			if parent != "." && parent != "/" {
				if err := os.MkdirAll(parent, 0755); err != nil {
					return err
				}
			}
			_, serr, err := gitutil.Cmd(cmd.Context(), ".", env, "clone", "--", r.Remote, dir)
			if err != nil {
				return fmt.Errorf("clone %s: %s", r.Remote, strings.TrimSpace(serr))
			}
			// ensure no credential leaked into the clone's config
			_, _, _ = gitutil.Cmd(cmd.Context(), dir, nil, "config", "--unset-all", "http.extraheader")
			st.renderer.Print(fmt.Sprintf("cloned %s (%s) -> %s", name, r.Remote, dir))
			return nil
		},
	}
	cmd.Flags().StringVar(&flagRepo, "repo", "", "owner/repo or repo name")
	cmd.Flags().StringVar(&token, "token", "", "scoped Git token (avoids SY_GIT_TOKEN)")
	cmd.Flags().StringVar(&dir, "dir", "", "clone directory (default: repo name)")
	cmd.Flags().MarkHidden("token")
	return cmd
}

// currentRepoName derives the repo name from the current git checkout's origin
// remote (Artifacts-style remote or any owner/repo remote).
func currentRepoName(ctx context.Context) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	remote, err := gitutil.RemoteURL(ctx, wd)
	if err != nil {
		return "", fmt.Errorf("not inside a git checkout with an origin remote: %w", err)
	}
	owner, name := gitutil.DetectOwnerRepo(remote)
	if name == "" {
		return "", fmt.Errorf("cannot derive repository name from remote %q", remote)
	}
	if owner != "" {
		return owner + "/" + name, nil
	}
	return name, nil
}