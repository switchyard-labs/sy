package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/switchyard-labs/sy/internal/gitutil"
)

// Repo is the domain view of a repository.
type Repo struct {
	Owner         string `json:"owner,omitempty"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	DefaultBranch string `json:"default_branch"`
	Remote        string `json:"remote"`
	Source        string `json:"source,omitempty"`
	ReadOnly      bool   `json:"read_only"`
	Registered    bool   `json:"registered"`
	Visibility    string `json:"visibility,omitempty"`
}

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Work with repositories",
	}
	cmd.AddCommand(newRepoListCmd(), newRepoViewCmd(), newRepoCloneCmd())
	return cmd
}

type canonicalRepo struct {
	Owner         string `json:"owner_slug"`
	Name          string `json:"slug"`
	FullName      string `json:"full_name"`
	ArtifactName  string `json:"artifact_name"`
	Description   string `json:"description"`
	Visibility    string `json:"visibility"`
	DefaultBranch string `json:"default_branch"`
	UpdatedAt     string `json:"updated_at"`
}

func canonicalRepos(cmd *cobra.Command) ([]canonicalRepo, error) {
	var out struct {
		Items []canonicalRepo `json:"items"`
	}
	err := stateFrom(cmd).client.Do(cmd.Context(), "GET", "/api/repositories", &out)
	return out.Items, err
}
func resolveRepo(cmd *cobra.Command, name string) (canonicalRepo, error) {
	items, err := canonicalRepos(cmd)
	if err != nil {
		return canonicalRepo{}, err
	}
	var matches []canonicalRepo
	for _, r := range items {
		if r.FullName == name || r.Name == name || r.ArtifactName == name {
			matches = append(matches, r)
		}
	}
	if len(matches) != 1 {
		return canonicalRepo{}, fmt.Errorf("repository %q matched %d registered repositories; specify owner/repo", name, len(matches))
	}
	return matches[0], nil
}
func canonicalPath(r canonicalRepo) string {
	return "/api/repositories/" + url.PathEscape(r.Owner) + "/" + url.PathEscape(r.Name)
}
func newRepoListCmd() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List repositories", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		st := stateFrom(cmd)
		items, err := canonicalRepos(cmd)
		if err != nil {
			return err
		}
		repos := make([]Repo, 0, len(items))
		rows := [][]string{}
		for _, r := range items {
			repos = append(repos, Repo{Owner: r.Owner, Name: r.Name, Description: r.Description, DefaultBranch: r.DefaultBranch, Visibility: r.Visibility, Registered: true})
			rows = append(rows, []string{r.FullName, r.Visibility, r.DefaultBranch, timeAgo(r.UpdatedAt)})
		}
		if st.renderer.JSONMode {
			return st.renderer.Emit(map[string]any{"items": repos})
		}
		st.renderer.Table([]string{"REPOSITORY", "VISIBILITY", "DEFAULT", "UPDATED"}, rows)
		return nil
	}}
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
			resolved, err := resolveRepo(cmd, name)
			if err != nil {
				return err
			}
			var backend struct {
				ReadOnly bool   `json:"read_only"`
				Source   string `json:"source"`
			}
			if err = st.client.Do(cmd.Context(), "GET", canonicalPath(resolved), &backend); err != nil {
				return err
			}
			found := &Repo{Owner: resolved.Owner, Name: resolved.Name, Description: resolved.Description, Visibility: resolved.Visibility, DefaultBranch: resolved.DefaultBranch, Registered: true, ReadOnly: backend.ReadOnly, Source: backend.Source}
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
	var flagRepo, dir string
	cmd := &cobra.Command{Use: "clone [owner/repo] [directory]", Short: "Clone using a short-lived scoped Switchyard credential", Args: cobra.MaximumNArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		st := stateFrom(cmd)
		name := flagRepo
		if len(args) > 0 {
			name = args[0]
		}
		if name == "" {
			var err error
			name, err = currentRepoName(cmd.Context())
			if err != nil {
				return err
			}
		}
		r, err := resolveRepo(cmd, name)
		if err != nil {
			return err
		}
		var capability struct {
			Remote     string `json:"remote"`
			Token      string `json:"token"`
			Repository string `json:"repository"`
		}
		if err = st.client.Do(cmd.Context(), "POST", canonicalPath(r)+"/git-credential", &capability, apiBody(map[string]any{"scope": "read", "ttl_seconds": 600})); err != nil {
			return err
		}
		if capability.Repository != r.FullName {
			return fmt.Errorf("clone capability repository mismatch")
		}
		env, err := gitutil.ScopedCredentialEnv(capability.Remote, capability.Token)
		if err != nil {
			return err
		}
		if len(args) > 1 {
			dir = args[1]
		}
		if dir == "" {
			dir = r.Name
		}
		parent := filepath.Dir(dir)
		if parent != "." && parent != "/" {
			if err = os.MkdirAll(parent, 0755); err != nil {
				return err
			}
		}
		_, serr, err := gitutil.Cmd(cmd.Context(), ".", env, "clone", "--", capability.Remote, dir)
		if err != nil {
			return fmt.Errorf("clone %s: %s", r.FullName, strings.ReplaceAll(strings.TrimSpace(serr), capability.Token, "[redacted]"))
		}
		gitdir, _, err := gitutil.Cmd(cmd.Context(), dir, nil, "rev-parse", "--absolute-git-dir")
		if err != nil {
			return err
		}
		contextBytes, _ := json.Marshal(map[string]string{"repository": r.FullName})
		if err = os.WriteFile(filepath.Join(strings.TrimSpace(gitdir), "switchyard.json"), contextBytes, 0600); err != nil {
			return err
		}
		if st.renderer.JSONMode {
			return st.renderer.Emit(map[string]any{"repository": r.FullName, "directory": dir, "remote": capability.Remote})
		}
		st.renderer.Print(fmt.Sprintf("Cloned %s into %s", r.FullName, dir))
		return nil
	}}
	cmd.Flags().StringVar(&flagRepo, "repo", "", "owner/repo or unique repository name")
	cmd.Flags().StringVar(&dir, "dir", "", "clone directory")
	return cmd
}

// currentRepoName derives the repo name from the current git checkout's origin
// remote (Artifacts-style remote or any owner/repo remote).
func currentRepoName(ctx context.Context) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	gitdir, _, giterr := gitutil.Cmd(ctx, wd, nil, "rev-parse", "--absolute-git-dir")
	if giterr == nil {
		if data, readerr := os.ReadFile(filepath.Join(strings.TrimSpace(gitdir), "switchyard.json")); readerr == nil {
			var saved struct {
				Repository string `json:"repository"`
			}
			if json.Unmarshal(data, &saved) == nil && strings.Count(saved.Repository, "/") == 1 {
				return saved.Repository, nil
			}
		}
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
