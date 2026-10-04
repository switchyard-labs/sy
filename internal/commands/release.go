package commands

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

type releaseAsset struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	State  string `json:"state"`
}
type releaseInfo struct {
	Tag         string         `json:"tag"`
	Title       string         `json:"title"`
	Body        string         `json:"body"`
	Draft       string         `json:"draft"`
	Prerelease  string         `json:"prerelease"`
	TargetSHA   string         `json:"target_sha"`
	Author      string         `json:"author"`
	PublishedAt string         `json:"published_at"`
	Assets      []releaseAsset `json:"assets"`
}

func releaseState(r releaseInfo) string {
	if r.Draft == "true" {
		return "draft"
	}
	if r.Prerelease == "true" {
		return "prerelease"
	}
	return "published"
}
func newReleaseCmd() *cobra.Command {
	var repo string
	group := &cobra.Command{Use: "release", Short: "Repository releases and immutable assets"}
	group.PersistentFlags().StringVarP(&repo, "repo", "R", "", "canonical owner/repo (default: checkout)")
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
		return canonicalPath(r), nil
	}
	show := func(cmd *cobra.Command, r releaseInfo) error {
		st := stateFrom(cmd)
		if st.renderer.JSONMode {
			return st.renderer.Emit(r)
		}
		st.renderer.Title(r.Title)
		st.renderer.KV([][2]string{{"Tag", r.Tag}, {"Status", releaseState(r)}, {"Commit", r.TargetSHA}, {"Author", r.Author}, {"Published", r.PublishedAt}})
		st.renderer.Print(r.Body)
		rows := [][]string{}
		for _, a := range r.Assets {
			rows = append(rows, []string{a.Name, strconv.FormatInt(a.Size, 10), a.State, a.SHA256})
		}
		st.renderer.Table([]string{"ASSET", "BYTES", "STATE", "SHA256"}, rows)
		return nil
	}
	list := &cobra.Command{Use: "list", Short: "List visible releases", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var result struct {
			Items []releaseInfo `json:"items"`
		}
		st := stateFrom(cmd)
		if err = st.client.Do(cmd.Context(), "GET", root+"/releases", &result); err != nil {
			return err
		}
		if st.renderer.JSONMode {
			return st.renderer.Emit(result)
		}
		rows := [][]string{}
		for _, r := range result.Items {
			rows = append(rows, []string{r.Tag, r.Title, releaseState(r), strconv.Itoa(len(r.Assets)), r.PublishedAt})
		}
		st.renderer.Table([]string{"TAG", "TITLE", "STATUS", "ASSETS", "PUBLISHED"}, rows)
		return nil
	}}
	view := &cobra.Command{Use: "view <tag>", Short: "Show a release and asset checksums", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var r releaseInfo
		if err = stateFrom(cmd).client.Do(cmd.Context(), "GET", root+"/releases/"+url.PathEscape(args[0]), &r); err != nil {
			return err
		}
		return show(cmd, r)
	}}
	var title, notes, notesFile string
	var prerelease bool
	create := &cobra.Command{Use: "create <existing-tag>", Short: "Create a draft from an existing Git tag", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if notes != "" && notesFile != "" {
			return fmt.Errorf("choose --notes or --notes-file")
		}
		body := notes
		if notesFile != "" {
			b, err := os.ReadFile(notesFile)
			if err != nil {
				return err
			}
			if len(b) > 65536 {
				return fmt.Errorf("notes exceed 64 KiB")
			}
			body = string(b)
		}
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var r releaseInfo
		if err = stateFrom(cmd).client.Do(cmd.Context(), "POST", root+"/releases", &r, apiBody(map[string]any{"tag": args[0], "title": title, "body": body, "prerelease": prerelease})); err != nil {
			return err
		}
		return show(cmd, r)
	}}
	create.Flags().StringVar(&title, "title", "", "Release title")
	create.Flags().StringVar(&notes, "notes", "", "Markdown notes")
	create.Flags().StringVar(&notesFile, "notes-file", "", "Read Markdown notes from a file")
	create.Flags().BoolVar(&prerelease, "prerelease", false, "Mark as a prerelease")
	publish := &cobra.Command{Use: "publish <tag>", Short: "Publish a draft and freeze its assets", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var r releaseInfo
		if err = stateFrom(cmd).client.Do(cmd.Context(), "PATCH", root+"/releases/"+url.PathEscape(args[0]), &r, apiBody(map[string]any{"publish": true})); err != nil {
			return err
		}
		return show(cmd, r)
	}}
	upload := &cobra.Command{Use: "upload <tag> <file>", Short: "Attach an asset to a draft", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		var a releaseAsset
		if err = stateFrom(cmd).client.UploadFile(cmd.Context(), root+"/release-assets?tag="+url.QueryEscape(args[0])+"&name="+url.QueryEscape(filepath.Base(args[1])), args[1], &a); err != nil {
			return err
		}
		st := stateFrom(cmd)
		if st.renderer.JSONMode {
			return st.renderer.Emit(a)
		}
		st.renderer.KV([][2]string{{"Asset", a.Name}, {"Bytes", strconv.FormatInt(a.Size, 10)}, {"SHA256", a.SHA256}, {"State", a.State}})
		return nil
	}}
	var destination string
	download := &cobra.Command{Use: "download <tag> <asset-name>", Short: "Download an asset and verify its size and SHA256", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		root, err := rootFor(cmd)
		if err != nil {
			return err
		}
		st := stateFrom(cmd)
		var r releaseInfo
		if err = st.client.Do(cmd.Context(), "GET", root+"/releases/"+url.PathEscape(args[0]), &r); err != nil {
			return err
		}
		for _, a := range r.Assets {
			if a.Name != args[1] {
				continue
			}
			if a.State != "ready" {
				return fmt.Errorf("asset is not ready")
			}
			target := destination
			if target == "" {
				target = filepath.Base(a.Name)
			}
			target, err = filepath.Abs(target)
			if err != nil {
				return err
			}
			n, err := st.client.DownloadAsset(cmd.Context(), root+"/release-assets/"+url.PathEscape(a.ID)+"?tag="+url.QueryEscape(args[0]), target, a.SHA256, a.Size)
			if err != nil {
				return err
			}
			result := map[string]any{"path": target, "bytes": n, "sha256": a.SHA256, "asset": a.Name}
			if st.renderer.JSONMode {
				return st.renderer.Emit(result)
			}
			st.renderer.Print(fmt.Sprintf("Downloaded %s to %s (%d bytes, SHA256 verified)", a.Name, target, n))
			return nil
		}
		return fmt.Errorf("release asset not found")
	}}
	download.Flags().StringVarP(&destination, "output", "o", "", "Output file; existing files are never overwritten")
	group.AddCommand(list, view, create, publish, upload, download)
	return group
}
