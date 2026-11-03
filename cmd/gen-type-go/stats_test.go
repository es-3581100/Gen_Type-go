package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/es-3581100/Gen_Type-go/internal/graphstats"
)

func TestStatsCLIJSONSummaryAndStdin(t *testing.T) {
	code, got, stderr := execute("stats", example)
	if code != 0 || stderr != "" {
		t.Fatalf("run(stats, %s) = code %d, stderr %q", example, code, stderr)
	}
	var report graphstats.Report
	if err := json.Unmarshal([]byte(got), &report); err != nil {
		t.Fatalf("run(stats) JSON decode: %v", err)
	}
	if report.Schema != graphstats.Schema || report.NodeCount != 2 || report.ExplicitEdgeCount != 1 || report.RootCount != 1 || report.LeafCount != 1 || report.MaxDepth != 1 || report.WeakComponentCount != 1 {
		t.Errorf("run(stats) report = %+v, want valid two-node measurements", report)
	}
	if report.IdentityBoundary != graphstats.IdentityBoundary {
		t.Errorf("run(stats) identity boundary = %q, want %q", report.IdentityBoundary, graphstats.IdentityBoundary)
	}
	for i := 0; i < 12; i++ {
		code, again, stderr := execute("stats", example)
		if code != 0 || stderr != "" || again != got {
			t.Fatalf("run(stats) repeat %d = (%d,%q,%q), want (%d,%q,empty)", i, code, again, stderr, 0, got)
		}
	}
	code, text, stderr := execute("stats", "--summary", example)
	if code != 0 || stderr != "" || !strings.Contains(text, "Nodes: 2") || !strings.Contains(text, "Weak components: 1") || !strings.Contains(text, report.CanonicalSHA256) || !strings.Contains(text, "untrusted") {
		t.Errorf("run(stats --summary) = (%d,%q,%q), want counts and provenance warning", code, text, stderr)
	}
	input, err := os.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	code = run([]string{"stats", "-"}, bytes.NewReader(input), &out, &errs)
	if code != 0 || errs.Len() != 0 || out.String() != got {
		t.Errorf("run(stats, stdin) = (%d,%q,%q), want identical file output", code, out.String(), errs.String())
	}
	inputAfter, err := os.ReadFile(example)
	if err != nil || !bytes.Equal(input, inputAfter) {
		t.Errorf("run(stats) mutated %s", example)
	}
}

func TestStatsCLIRejectsBadInputsWithoutPartialSuccess(t *testing.T) {
	for _, name := range []string{"invalid_duplicate_id.json", "invalid_cycle.json", "malformed_alias.json", "malformed_wrong_type.json"} {
		path := filepath.Join("../../testdata", name)
		code, stdout, stderr := execute("stats", path)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "invalid graph") {
			t.Errorf("run(stats %s) = (%d,%q,%q), want exit 2, no stdout", path, code, stdout, stderr)
		}
	}
	for _, args := range [][]string{
		{"stats"}, {"stats", example, example}, {"stats", "--sha256", strings.Repeat("a", 64), example},
		{"validate", "--summary", example}, {"sha256", "--summary", example},
	} {
		code, stdout, _ := execute(args...)
		if code != 2 || stdout != "" {
			t.Errorf("run(%v) = (%d,%q), want exit 2 and no output", args, code, stdout)
		}
	}
	var out, errbuf bytes.Buffer
	code := run([]string{"stats", "-"}, strings.NewReader(strings.Repeat("x", 2<<20+1)), &out, &errbuf)
	if code != 2 || out.Len() != 0 || !strings.Contains(errbuf.String(), "budget") {
		t.Errorf("run(stats, oversized stdin) = (%d,%q,%q), want rejection", code, out.String(), errbuf.String())
	}
}

type shortStatsWriter struct{}

func (shortStatsWriter) Write(b []byte) (int, error) { return 1, nil }

type failingStatsWriter struct{}

func (failingStatsWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestStatsCLIOutputFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		out  io.Writer
	}{
		{"short write", shortStatsWriter{}},
		{"writer error", failingStatsWriter{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var errors bytes.Buffer
			code := run([]string{"stats", example}, strings.NewReader(""), tc.out, &errors)
			if code != 2 || !strings.Contains(errors.String(), "output:") {
				t.Errorf("run(stats, %s) = (%d,%q), want output failure", tc.name, code, errors.String())
			}
		})
	}
}

func TestStatsCLIInvalidWithUntrustedScriptTitle(t *testing.T) {
	input, err := os.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(input, &doc); err != nil {
		t.Fatal(err)
	}
	doc["nodes"].([]any)[0].(map[string]any)["title"] = "<script>alert(1)</script>"
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var output, errbuf bytes.Buffer
	code := run([]string{"stats", "-"}, bytes.NewReader(b), &output, &errbuf)
	if code != 0 || errbuf.Len() != 0 || !json.Valid(output.Bytes()) {
		t.Errorf("run(stats, script title) = (%d,%q,%q), want valid metrics", code, output.String(), errbuf.String())
	}
	if bytes.Contains(output.Bytes(), []byte("<script>")) || bytes.Contains(output.Bytes(), []byte("alert(1)")) {
		t.Errorf("run(stats, script title) leaked input title: %s", output.Bytes())
	}
}
