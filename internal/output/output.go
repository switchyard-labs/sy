// Package output centralizes rendering so each command produces a domain
// result and then a renderer (human or JSON). JSON is a stable public contract.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// Renderer renders a domain result.
type Renderer struct {
	Out     io.Writer
	JSONMode bool
	Quiet   bool
	NoColor bool
}

func New(out io.Writer, jsonOut, quiet, noColor bool) *Renderer {
	return &Renderer{Out: out, JSONMode: jsonOut, Quiet: quiet, NoColor: noColor}
}

// JSON writes the result as stable JSON.
func (r *Renderer) Emit(v any) error {
	enc := json.NewEncoder(r.Out)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// Print writes a plain line.
func (r *Renderer) Print(s string) {
	if r.Quiet {
		return
	}
	fmt.Fprintln(r.Out, s)
}

// Table renders a header + rows with aligned columns (human only).
func (r *Renderer) Table(header []string, rows [][]string) {
	if r.JSONMode || r.Quiet {
		return
	}
	w := tabwriter.NewWriter(r.Out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, strings.Join(header, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// KV renders key/value pairs as "key: value" (human only).
func (r *Renderer) KV(pairs [][2]string) {
	if r.JSONMode || r.Quiet {
		return
	}
	for _, p := range pairs {
		fmt.Fprintf(r.Out, "%-16s %s\n", p[0]+":", p[1])
	}
}

// Title renders a prominent heading.
func (r *Renderer) Title(s string) {
	if r.JSONMode || r.Quiet {
		return
	}
	fmt.Fprintf(r.Out, "\n%s\n", s)
}

// TimeAgo renders an RFC3339 timestamp as a short relative duration.
func TimeAgo(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// IsTerminal reports whether the output is a TTY.
func IsTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// Colorize wraps s in ANSI color unless disabled / not a terminal.
func (r *Renderer) Colorize(code, s string) string {
	if r.NoColor || !IsTerminal() || os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// SortedKeys returns sorted map keys (stable JSON-friendly ordering helper).
func SortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}