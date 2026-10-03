package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONStable(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, true, false, false)
	err := r.Emit(map[string]any{"b": 1, "a": 2})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["a"] != float64(2) || m["b"] != float64(1) {
		t.Fatalf("bad json: %s", buf.String())
	}
}

func TestHumanTable(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, false, false, false)
	r.Table([]string{"A", "B"}, [][]string{{"x", "y"}})
	if !strings.Contains(buf.String(), "x") {
		t.Fatalf("table missing rows: %q", buf.String())
	}
}

func TestQuietSuppressesHuman(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, false, true, false)
	r.Print("hello")
	if buf.Len() != 0 {
		t.Fatalf("quiet should suppress: %q", buf.String())
	}
}

func TestNoColorNeverEmitsAnsi(t *testing.T) {
	var buf bytes.Buffer
	r := New(&buf, false, false, true)
	got := r.Colorize("31", "red")
	if strings.Contains(got, "\x1b[") {
		t.Fatalf("no-color should not emit ansi: %q", got)
	}
}