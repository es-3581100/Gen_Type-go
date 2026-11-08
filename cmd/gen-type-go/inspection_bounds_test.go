package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/es-3581100/Gen_Type-go/internal/graphdiagnostics"
)

// runWithin fails fast instead of letting a command that never returns consume
// the whole test timeout.
func runWithin(t *testing.T, limit time.Duration, args []string, stdin string) (int, string, string) {
	t.Helper()
	type result struct {
		code     int
		out, err string
	}
	done := make(chan result, 1)
	go func() {
		var o, e bytes.Buffer
		code := run(args, strings.NewReader(stdin), &o, &e)
		done <- result{code, o.String(), e.String()}
	}()
	select {
	case r := <-done:
		return r.code, r.out, r.err
	case <-time.After(limit):
		t.Fatalf("%v did not return within %v", args, limit)
		return 0, "", ""
	}
}

func rejectionReport(t *testing.T, code int, out, errout string) graphdiagnostics.Report {
	t.Helper()
	var r graphdiagnostics.Report
	if code != 2 || errout != "" || json.Unmarshal([]byte(out), &r) != nil || r.Valid || len(r.Diagnostics) != 1 {
		t.Fatalf("want exit 2 with one rejection report on stdout; got exit %d stdout %.120q stderr %.120q", code, out, errout)
	}
	return r
}

func TestDiagnoseCLIRejectsDeepNestingInsteadOfHanging(t *testing.T) {
	for _, depth := range []int{98, 200} {
		in := strings.Repeat("[", depth) + strings.Repeat("]", depth)
		code, out, errout := runWithin(t, 5*time.Second, []string{"diagnose", "-"}, in)
		if r := rejectionReport(t, code, out, errout); r.Diagnostics[0].Code != "NESTING_TOO_DEEP" {
			t.Fatalf("depth %d: %+v", depth, r)
		}
	}
}

func TestDiagnoseCLIHostileKeyStillProducesReport(t *testing.T) {
	claim := strings.Repeat("a", 64)
	in := `{"schema":"gen-type-go/phase-a-graph-v0","project_id":"p","source_id":"s","nodes":[` +
		`{"id":"r","parent_id":"","order":0,"title":"","content":"","source_ref":"x","source_digest_claim":"` + claim + `","` +
		strings.Repeat("<", 800_000) + `":1}],"edges":[]}`
	code, out, errout := runWithin(t, 10*time.Second, []string{"diagnose", "-"}, in)
	r := rejectionReport(t, code, out, errout)
	if r.Diagnostics[0].Code != "UNKNOWN_FIELD" || len(out) > 4096 {
		t.Fatalf("%+v (%d bytes)", r, len(out))
	}
}
