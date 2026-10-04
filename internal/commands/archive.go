package commands

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
)

func newRepoArchiveCmd() *cobra.Command {
	var ref, format, destination string
	cmd := &cobra.Command{Use: "archive [owner/repo]", Short: "Download an immutable repository source archive", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if format != "zip" && format != "tar.gz" {
			return fmt.Errorf("format must be zip or tar.gz")
		}
		name := ""
		if len(args) > 0 {
			name = args[0]
		} else {
			var err error
			name, err = currentRepoName(cmd.Context())
			if err != nil {
				return err
			}
		}
		repo, err := resolveRepo(cmd, name)
		if err != nil {
			return err
		}
		if ref == "" {
			ref = repo.DefaultBranch
		}
		var overview struct {
			Latest struct {
				Hash string `json:"hash"`
			} `json:"latest_commit"`
		}
		st := stateFrom(cmd)
		if err = st.client.Do(cmd.Context(), "GET", canonicalPath(repo)+"/overview?ref="+url.QueryEscape(ref), &overview); err != nil {
			return err
		}
		sha := overview.Latest.Hash
		if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(sha) {
			return fmt.Errorf("ref has no resolved commit")
		}
		if destination == "" {
			destination = repo.Name + "-" + sha[:12] + "." + format
		}
		destination, err = filepath.Abs(destination)
		if err != nil {
			return err
		}
		if _, err = os.Lstat(destination); err == nil {
			return fmt.Errorf("output file already exists: %s", destination)
		} else if !os.IsNotExist(err) {
			return err
		}
		n, err := st.client.DownloadFile(cmd.Context(), canonicalPath(repo)+"/archive/"+sha+"."+format, destination, sha, 136<<20)
		if err != nil {
			return err
		}
		if st.renderer.JSONMode {
			return st.renderer.Emit(map[string]any{"repository": repo.FullName, "ref": ref, "commit": sha, "format": format, "path": destination, "bytes": n})
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %s at %s to %s (%d bytes)\n", repo.FullName, sha[:12], destination, n)
		return nil
	}}
	cmd.Flags().StringVar(&ref, "ref", "", "Branch, tag or commit (defaults to repository default branch)")
	cmd.Flags().StringVar(&format, "format", "zip", "Archive format: zip or tar.gz")
	cmd.Flags().StringVarP(&destination, "output", "o", "", "Output file; existing files are never overwritten")
	return cmd
}
