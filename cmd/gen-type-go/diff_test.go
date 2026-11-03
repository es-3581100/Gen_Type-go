package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/es-3581100/Gen_Type-go/internal/graphdiff"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

const changedExample = "../../testdata/diff_after.json"

func TestDiffCLIIdenticalAndExample(t *testing.T) {
	code, raw, stderr := execute("diff", example, example)
	if code != 0 || stderr != "" {
		t.Fatalf("diff identical: exit=%d stderr=%q", code, stderr)
	}
	var identical graphdiff.Report
	if err := json.Unmarshal([]byte(raw), &identical); err != nil {
		t.Fatal(err)
	}
	if !identical.Identical || identical.BeforeSHA256 != identical.AfterSHA256 || len(identical.AddedNodes) != 0 {
		t.Errorf("diff identical: %+v", identical)
	}
	code, raw, stderr = execute("diff", example, changedExample)
	if code != 0 || stderr != "" {
		t.Fatalf("diff changed: exit=%d stderr=%q", code, stderr)
	}
	var report graphdiff.Report
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatal(err)
	}
	if report.Identical || report.Schema != graphdiff.Schema || len(report.AddedNodes) != 1 || report.AddedNodes[0] != "note" || len(report.RemovedEdges) != 1 || len(report.AddedEdges) != 2 {
		t.Errorf("diff example: %+v", report)
	}
	for i := 0; i < 8; i++ {
		againCode, again, againErr := execute("diff", example, changedExample)
		if againCode != 0 || againErr != "" || again != raw {
			t.Fatalf("diff example run %d not byte-identical to first: %d %q", i, againCode, againErr)
		}
	}
}

func TestDiffCLIStdinAndSummary(t *testing.T) {
	original, err := os.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	code, want, errout := execute("diff", example, changedExample)
	if code != 0 || errout != "" {
		t.Fatalf("diff normal: %d %s", code, errout)
	}
	for _, args := range [][]string{{"diff", "-", changedExample}, {"diff", example, "-"}} {
		var out, errbuf bytes.Buffer
		input := original
		if args[2] == "-" {
			input, err = os.ReadFile(changedExample)
			if err != nil {
				t.Fatal(err)
			}
		}
		code := run(args, bytes.NewReader(input), &out, &errbuf)
		if code != 0 || out.String() != want || errbuf.Len() != 0 {
			t.Errorf("run(%v, stdin): %d, %q, %q", args, code, out.String(), errbuf.String())
		}
	}
	code, summary, errout := execute("diff", "--summary", example, changedExample)
	if code != 0 || errout != "" || !strings.Contains(summary, "Added nodes: 1") || !strings.Contains(summary, "Before SHA-256:") || strings.Contains(summary, "authorized") {
		t.Errorf("diff --summary: %d, %q, %q", code, summary, errout)
	}
}

func TestDiffCLIFailureIsNotSuccess(t *testing.T) {
	cases := [][]string{
		{"diff"}, {"diff", example}, {"diff", example, changedExample, "extra"},
		{"diff", "-", "-"}, {"diff", "--sha256", strings.Repeat("0", 64), example, changedExample},
		{"diff", "../../testdata/invalid_cycle.json", changedExample},
		{"diff", example, "../../testdata/invalid_duplicate_id.json"},
		{"diff", example, "../../testdata/malformed_alias.json"},
		{"diff", "../../testdata/malformed_wrong_type.json", changedExample},
		{"diff", example, "missing-file.json"},
		{"validate", "--summary", example},
	}
	for _, args := range cases {
		code, stdout, stderr := execute(args...)
		if code != 2 || stdout != "" || stderr == "" {
			t.Errorf("run(%v): (%d, %q, %q), want (2, empty, nonempty)", args, code, stdout, stderr)
		}
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"diff", "-", changedExample}, strings.NewReader(strings.Repeat("a", phasea.MaxInputBytes+1)), &stdout, &stderr)
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "budget") {
		t.Errorf("diff overbudget: (%d, %q, %q), want exit 2 and no stdout", code, stdout.String(), stderr.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write(p []byte) (int, error) { return 0, errors.New("injected output failure") }

func TestDiffCLIBrokenOutput(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"diff", example, changedExample}, strings.NewReader(""), brokenWriter{}, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "injected output failure") {
		t.Errorf("diff output failure: (%d, %q), want exit 2", code, stderr.String())
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestDiffCLIShortWrite(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"diff", example, changedExample}, strings.NewReader(""), shortWriter{}, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "short write") {
		t.Errorf("diff short output: (%d, %q), want exit 2 and short-write diagnosis", code, stderr.String())
	}
}
