package main

import (
	"bytes"
	"encoding/json"
	"github.com/es-3581100/Gen_Type-go/internal/graphdiagnostics"
	"github.com/es-3581100/Gen_Type-go/internal/graphinspect"
	"os"
	"strings"
	"testing"
)

func TestInspectCLI(t *testing.T) {
	args := []string{"inspect", "--id", "c", "../../testdata/stats_forest.json"}
	code, raw, stderr := execute(args...)
	if code != 0 || stderr != "" {
		t.Fatalf("%d %q", code, stderr)
	}
	var r graphinspect.Report
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.Node.ID != "c" || r.Parent == nil || r.Parent.ID != "a" || len(r.Ancestors) != 2 || len(r.IncomingEdges) != 1 || len(r.OutgoingEdges) != 1 {
		t.Fatalf("%+v", r)
	}
	source, err := os.ReadFile("../../testdata/stats_forest.json")
	if err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code = run([]string{"inspect", "--id", "c", "-"}, bytes.NewReader(source), &o, &e)
	if code != 0 || e.Len() != 0 || o.String() != raw {
		t.Fatalf("stdin %d %q", code, e.String())
	}
	for i := 0; i < 5; i++ {
		_, got, _ := execute(args...)
		if got != raw {
			t.Fatal("nondeterministic inspection")
		}
	}
}
func TestDiagnoseCLI(t *testing.T) {
	code, raw, stderr := execute("diagnose", example)
	if code != 0 || stderr != "" {
		t.Fatalf("valid %d %q", code, stderr)
	}
	var r graphdiagnostics.Report
	if err := json.Unmarshal([]byte(raw), &r); err != nil || !r.Valid || r.CanonicalSHA256 == "" || len(r.Diagnostics) != 0 {
		t.Fatalf("%+v %v", r, err)
	}
	code, raw, stderr = execute("diagnose", "../../testdata/invalid_duplicate_id.json")
	if code != 2 || stderr != "" {
		t.Fatalf("invalid %d %q", code, stderr)
	}
	if err := json.Unmarshal([]byte(raw), &r); err != nil || r.Valid || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "DUPLICATE_NODE" {
		t.Fatalf("%+v %v", r, err)
	}
	b, err := os.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code = run([]string{"diagnose", "-"}, bytes.NewReader(b), &o, &e)
	if code != 0 || e.Len() != 0 || o.Len() == 0 {
		t.Fatalf("stdin %d %q", code, e.String())
	}
}
func TestInspectionCLIRejectionAndWriter(t *testing.T) {
	cases := [][]string{
		{"inspect"}, {"inspect", example}, {"inspect", "--id", "root"},
		{"inspect", "--id", "missing", example}, {"inspect", "--id", "root", example, example},
		{"inspect", "--sha256", "a", example}, {"inspect", "--id", "root", "../../testdata/invalid_cycle.json"},
		{"diagnose"}, {"diagnose", example, example}, {"diagnose", "--summary", example}, {"diagnose", "missing.json"},
	}
	for _, args := range cases {
		code, out, err := execute(args...)
		if code != 2 || out != "" || err == "" {
			t.Fatalf("%v: %d %q %q", args, code, out, err)
		}
	}
	code, raw, stderr := execute("diagnose", "../../testdata/malformed_alias.json")
	if code != 2 || stderr != "" || !strings.Contains(raw, `"code":"UNKNOWN_FIELD"`) {
		t.Fatalf("malformed %d %q %q", code, raw, stderr)
	}
	for _, args := range [][]string{{"inspect", "--id", "root", example}, {"diagnose", example}} {
		var errout bytes.Buffer
		code = run(args, strings.NewReader(""), brokenWriter{}, &errout)
		if code != 2 || !strings.Contains(errout.String(), "injected output failure") {
			t.Fatalf("%v: %d %q", args, code, errout.String())
		}
		errout.Reset()
		code = run(args, strings.NewReader(""), shortWriter{}, &errout)
		if code != 2 || !strings.Contains(errout.String(), "short write") {
			t.Fatalf("%v: %d %q", args, code, errout.String())
		}
	}
}
