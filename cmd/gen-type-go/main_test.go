package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const example = "../../testdata/tiny_graph.json"

func execute(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := run(args, strings.NewReader(""), &o, &e)
	return code, o.String(), e.String()
}
func TestWorkingCommands(t *testing.T) {
	code, out, err := execute("sha256", example)
	if code != 0 || err != "" || len(strings.TrimSpace(out)) != 64 {
		t.Fatalf("sha256: %d %q %q", code, out, err)
	}
	sum := strings.TrimSpace(out)
	for _, tc := range []struct {
		cmd    string
		marker string
	}{
		{"help", "Phase A graph workbench"}, {"version", "0.1.0-phase-a-preview"},
		{"validate", "VALID graph: 2 nodes, 1 edges"},
		{"roundtrip", "ROUNDTRIP PASS"},
		{"svg", "<svg"}, {"html", "<!doctype html>"},
	} {
		var a []string
		if tc.cmd == "help" || tc.cmd == "version" {
			a = []string{tc.cmd}
		} else {
			a = []string{tc.cmd, example}
		}
		code, out, err := execute(a...)
		if code != 0 || err != "" || !strings.Contains(out, tc.marker) {
			t.Errorf("%s: %d %q %q", tc.cmd, code, out, err)
		}
	}
	code, out, err = execute("verify", "--sha256", sum, example)
	if code != 0 || !strings.Contains(out, "MATCH") || err != "" {
		t.Fatalf("verify: %d %q %q", code, out, err)
	}
	code, out, err = execute("canonical", example)
	if code != 0 || err != "" {
		t.Fatalf("canonical: %d %s", code, err)
	}
	var decoded struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
	}
	if json.Unmarshal([]byte(out), &decoded) != nil || len(decoded.Nodes) != 2 || decoded.Nodes[0].ID != "root" {
		t.Fatalf("unexpected canonical: %s", out)
	}
	code, _, _ = execute("verify", "--sha256", strings.Repeat("0", 64), example)
	if code != 1 {
		t.Fatalf("expected mismatch exit 1, got %d", code)
	}
}
func TestStdinCanonicalAndNonMutation(t *testing.T) {
	orig, err := os.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := run([]string{"canonical", "-"}, bytes.NewReader(orig), &o, &e)
	if code != 0 || !json.Valid(o.Bytes()) {
		t.Fatalf("stdin canonical failed: %d %s", code, e.String())
	}
	now, err := os.ReadFile(example)
	if err != nil || !bytes.Equal(now, orig) {
		t.Fatal("input file changed")
	}
	var o2, e2 bytes.Buffer
	code = run([]string{"canonical", "-"}, bytes.NewReader(o.Bytes()), &o2, &e2)
	if code != 0 || !bytes.Equal(o.Bytes(), o2.Bytes()) {
		t.Fatalf("non-idempotent canonical: %d %s", code, e2.String())
	}
}
func TestBadInputs(t *testing.T) {
	for _, name := range []string{"invalid_duplicate_id.json", "invalid_cycle.json", "malformed_alias.json", "malformed_wrong_type.json"} {
		code, out, _ := execute("validate", filepath.Join("../../testdata", name))
		if code != 2 || out != "" {
			t.Errorf("%s: expected exit 2/no stdout got %d %q", name, code, out)
		}
	}
	for _, a := range [][]string{{"mystery"}, {"validate"}, {"verify", "--sha256", "oops", example}, {"version", "surplus"}, {"canonical", example, "extra"}} {
		code, _, _ := execute(a...)
		if code != 2 {
			t.Errorf("args %v: exit %d", a, code)
		}
	}
	var out, errbuf bytes.Buffer
	code := run([]string{"validate", "-"}, strings.NewReader(strings.Repeat("a", 2<<20+1)), &out, &errbuf)
	if code != 2 || out.Len() != 0 || !strings.Contains(errbuf.String(), "budget") {
		t.Fatalf("over-budget: %d %q", code, errbuf.String())
	}
}
func TestCanonicalDigestSameUnderNodeReorder(t *testing.T) {
	b, err := os.ReadFile(example)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]interface{}
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	nodes := v["nodes"].([]interface{})
	nodes[0], nodes[1] = nodes[1], nodes[0]
	alt, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "reordered.json")
	if err = os.WriteFile(path, alt, 0600); err != nil {
		t.Fatal(err)
	}
	_, a, _ := execute("sha256", example)
	_, other, _ := execute("sha256", path)
	if a != other {
		t.Fatalf("hash changed with list order: %q vs %q", a, other)
	}
}
func TestJSONErrorsHaveNoDataOnStdout(t *testing.T) {
	var out, er bytes.Buffer
	bad := `{"schema":"wrong","project_id":"p","source_id":"s","nodes":[],"edges":[]}`
	code := run([]string{"svg", "-"}, strings.NewReader(bad), &out, &er)
	if code != 2 || out.Len() != 0 {
		t.Fatalf("invalid graph leakage: %d %q %q", code, out.String(), er.String())
	}
}
func Example_run() {
	var o, e bytes.Buffer
	run([]string{"version"}, strings.NewReader(""), &o, &e)
	fmt.Print(o.String())
	// Output: gen-type-go 0.1.0-phase-a-preview
}
