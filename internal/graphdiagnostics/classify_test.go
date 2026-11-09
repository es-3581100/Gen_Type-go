package graphdiagnostics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

func nodeJSON(id, parent string, order int) string {
	return fmt.Sprintf(`{"id":%q,"parent_id":%q,"order":%d,"title":"","content":"","source_ref":"x","source_digest_claim":%q}`, id, parent, order, claim)
}

func graphJSON(edges string, nodes ...string) []byte {
	return []byte(`{"schema":"gen-type-go/phase-a-graph-v0","project_id":"p","source_id":"s","nodes":[` +
		strings.Join(nodes, ",") + `],"edges":[` + edges + `]}`)
}

// keyLiteral encodes an object key without HTML escaping, so a '<' costs one
// input byte, as an attacker would send it.
func keyLiteral(t *testing.T, key string) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(key); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

// diagnoseWithin fails fast rather than letting a non-terminating Diagnose
// consume the whole test timeout (the classifier once spun forever on input
// nested more than 97 arrays deep).
func diagnoseWithin(t *testing.T, limit time.Duration, raw []byte) Report {
	t.Helper()
	done := make(chan Report, 1)
	go func() { done <- Diagnose(raw) }()
	select {
	case r := <-done:
		return r
	case <-time.After(limit):
		t.Fatalf("Diagnose did not return within %v for a %d-byte input", limit, len(raw))
		return Report{}
	}
}

// TestEveryRejectionClassIsPinned gives each classifier branch a case that the
// strict decoder also rejects, so the advisory code cannot drift from the oracle
// unnoticed.
func TestEveryRejectionClassIsPinned(t *testing.T) {
	b := valid()
	rep := func(old, replacement string) []byte {
		out := bytes.Replace(b, []byte(old), []byte(replacement), 1)
		if bytes.Equal(out, b) {
			t.Fatalf("fixture drift: %q not found", old)
		}
		return out
	}
	const head = `{"schema":"gen-type-go/phase-a-graph-v0","project_id":"p","source_id":"s",`
	bigNodes := make([]string, phasea.MaxNodes+1)
	for i := range bigNodes {
		bigNodes[i] = nodeJSON(fmt.Sprintf("n%d", i), "", i)
	}
	manyEdges := strings.TrimSuffix(strings.Repeat(`{"from":"root","to":"child","kind":"links_to"},`, phasea.MaxEdges+1), ",")

	cases := []struct {
		name       string
		raw        []byte
		code, path string
	}{
		{"top-level array", []byte(`[]`), "FIELD_TYPE", "$"},
		{"top-level null", []byte(`null`), "FIELD_TYPE", "$"},
		{"schema not a string", rep(`"schema":"gen-type-go/phase-a-graph-v0"`, `"schema":7`), "FIELD_TYPE", "$.schema"},
		{"nodes not an array", []byte(head + `"nodes":{},"edges":[]}`), "FIELD_TYPE", "$.nodes"},
		{"edges not an array", []byte(head + `"nodes":[],"edges":{}}`), "FIELD_TYPE", "$.edges"},
		{"node not an object", []byte(head + `"nodes":[1],"edges":[]}`), "FIELD_TYPE", "$.nodes[0]"},
		{"node field not a string", rep(`"title":"Child"`, `"title":5`), "FIELD_TYPE", "$.nodes[1].title"},
		{"order not an integer", rep(`"order":0,"title":"Child"`, `"order":1.5,"title":"Child"`), "FIELD_TYPE", "$.nodes[1].order"},
		{"edge not an object", rep(`"edges":[]`, `"edges":[1]`), "FIELD_TYPE", "$.edges[0]"},
		{"edge field not a string", rep(`"edges":[]`, `"edges":[{"from":1,"to":"child","kind":"links_to"}]`), "FIELD_TYPE", "$.edges[0].from"},
		{"edge field missing", rep(`"edges":[]`, `"edges":[{"from":"root","to":"child"}]`), "MISSING_FIELD", "$.edges[0].kind"},
		{"project id invalid", rep(`"project_id":"p"`, `"project_id":"!"`), "INVALID_ID", "$"},
		{"node id invalid", rep(`"id":"child"`, `"id":"ch!ld"`), "INVALID_ID", "$.nodes[1].id"},
		{"no nodes", []byte(head + `"nodes":[],"edges":[]}`), "INVALID_HIERARCHY", "$.nodes"},
		{"order negative", rep(`"order":0,"title":"Child"`, `"order":-1,"title":"Child"`), "INVALID_ORDER", "$.nodes[1].order"},
		{"order repeated among siblings", graphJSON("", nodeJSON("r", "", 0), nodeJSON("a", "r", 0), nodeJSON("b", "r", 0)), "INVALID_ORDER", "$.nodes[2].order"},
		{"order gap among siblings", graphJSON("", nodeJSON("r", "", 0), nodeJSON("a", "r", 0), nodeJSON("b", "r", 5)), "INVALID_ORDER", "$.nodes"},
		{"empty source ref", rep(`"source_ref":"local"`, `"source_ref":""`), "INVALID_SOURCE_CLAIM", "$.nodes[0]"},
		{"uppercase digest claim", bytes.Replace(b, []byte(claim), []byte(strings.ToUpper(claim)), 1), "INVALID_SOURCE_CLAIM", "$.nodes[0]"},
		{"field budget", rep(`"title":"Child"`, `"title":"`+strings.Repeat("x", 4097)+`"`), "INVALID_GRAPH", "$.nodes[1]"},
		{"self parent", graphJSON("", nodeJSON("r", "", 0), nodeJSON("a", "a", 0)), "INVALID_PARENT", "$.nodes[1].parent_id"},
		{"dangling parent only (not 'no root')", graphJSON("", nodeJSON("a", "ghost", 0)), "INVALID_PARENT", "$.nodes[0].parent_id"},
		{"no root, mutual parents", graphJSON("", nodeJSON("a", "b", 0), nodeJSON("b", "a", 0)), "INVALID_HIERARCHY", "$.nodes"},
		{"cycle unreachable from root", graphJSON("", nodeJSON("r", "", 0), nodeJSON("a", "b", 0), nodeJSON("b", "a", 0)), "INVALID_HIERARCHY", "$.nodes"},
		{"edge source missing", rep(`"edges":[]`, `"edges":[{"from":"zz","to":"root","kind":"links_to"}]`), "INVALID_EDGE_ENDPOINT", "$.edges[0].from"},
		{"edge target missing", rep(`"edges":[]`, `"edges":[{"from":"root","to":"zz","kind":"links_to"}]`), "INVALID_EDGE_ENDPOINT", "$.edges[0].to"},
		{"edge repeated", rep(`"edges":[]`, `"edges":[{"from":"root","to":"child","kind":"links_to"},{"from":"root","to":"child","kind":"links_to"}]`), "DUPLICATE_EDGE", "$.edges[1]"},
		{"node record budget", []byte(head + `"nodes":[` + strings.Join(bigNodes, ",") + `],"edges":[]}`), "INPUT_TOO_LARGE", "$"},
		{"edge record budget", rep(`"edges":[]`, `"edges":[`+manyEdges+`]`), "INPUT_TOO_LARGE", "$"},
		{"unpaired high surrogate", rep(`"title":"Child"`, `"title":"\ud800"`), "INVALID_ESCAPE", "$"},
		{"unpaired low surrogate", rep(`"title":"Child"`, `"title":"\udc00"`), "INVALID_ESCAPE", "$"},
		{"high surrogate then non-surrogate", rep(`"title":"Child"`, `"title":"\ud800\u0041"`), "INVALID_ESCAPE", "$"},
		{"high surrogate then high surrogate", rep(`"title":"Child"`, `"title":"\ud800\ud800"`), "INVALID_ESCAPE", "$"},
		{"trailing second value", append(append([]byte{}, b...), '{', '}'), "MALFORMED_JSON", "$"},
		{"nesting past the depth budget", []byte(strings.Repeat("[", 98) + strings.Repeat("]", 98)), "NESTING_TOO_DEEP", "$"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := phasea.DecodeStrict(tc.raw); err == nil {
				t.Fatal("fixture is accepted by the strict decoder; the case proves nothing")
			}
			r := diagnoseWithin(t, 10*time.Second, tc.raw)
			if r.Valid || r.CanonicalSHA256 != "" || len(r.Diagnostics) != 1 {
				t.Fatalf("unexpected report: %+v", r)
			}
			d := r.Diagnostics[0]
			if d.Code != tc.code || d.Path != tc.path || d.Message == "" || d.Action == "" {
				t.Fatalf("got %s @ %s, want %s @ %s", d.Code, d.Path, tc.code, tc.path)
			}
			if _, err := Encode(r); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSurrogatePairAndEscapedBackslashAreAccepted(t *testing.T) {
	b := valid()
	for name, title := range map[string]string{
		"valid pair":            `"\ud83d\ude00"`,
		"escaped backslash + u": `"\\ud800"`,
	} {
		raw := bytes.Replace(b, []byte(`"title":"Child"`), []byte(`"title":`+title), 1)
		if _, err := phasea.DecodeStrict(raw); err != nil {
			t.Fatalf("%s: strict decoder rejects the fixture: %v", name, err)
		}
		if r := diagnoseWithin(t, 10*time.Second, raw); !r.Valid || len(r.Diagnostics) != 0 {
			t.Fatalf("%s: %+v", name, r)
		}
	}
}

// TestNestingTerminatesAndMatchesDecoderBudget guards the infinite loop found in
// review: input nested more than 97 arrays deep (about 200 bytes) never returned.
func TestNestingTerminatesAndMatchesDecoderBudget(t *testing.T) {
	inputs := map[string]string{
		"objects": strings.Repeat(`{"a":`, 150) + `1` + strings.Repeat(`}`, 150),
		"mixed":   strings.Repeat(`[{"a":`, 80) + `1` + strings.Repeat(`}]`, 80),
	}
	for _, depth := range []int{1, 95, 96, 97, 98, 99, 200, 5000} {
		inputs[fmt.Sprintf("arrays-%d", depth)] = strings.Repeat("[", depth) + strings.Repeat("]", depth)
	}
	for name, in := range inputs {
		raw := []byte(in)
		_, strictErr := phasea.DecodeStrict(raw)
		if strictErr == nil {
			t.Fatalf("%s: strict decoder accepted a non-object", name)
		}
		wantDeep := strings.Contains(strictErr.Error(), "depth budget")
		r := diagnoseWithin(t, 5*time.Second, raw)
		gotDeep := len(r.Diagnostics) == 1 && r.Diagnostics[0].Code == "NESTING_TOO_DEEP"
		if r.Valid || gotDeep != wantDeep {
			t.Fatalf("%s: decoder depth-budget=%v, classifier NESTING_TOO_DEEP=%v (%+v)", name, wantDeep, gotDeep, r)
		}
	}
}

func TestHostileKeyReportStaysBounded(t *testing.T) {
	b := valid()
	keys := map[string]string{
		"huge":       strings.Repeat("<", 800_000), // 6 output bytes per input byte once JSON-escaped
		"controls":   "\x1b[31mred\nline",
		"dotted":     "x.title[0]",
		"non-ascii":  "é\u2028",
		"empty":      "",
		"multibyte":  strings.Repeat("é", 100),
		"plain":      "TITLE",
		"underscore": "_x1",
	}
	wantPath := map[string]string{
		"dotted":     `$.nodes[1]["x.title[0]"]`,
		"empty":      `$.nodes[1][""]`,
		"plain":      `$.nodes[1].TITLE`,
		"underscore": `$.nodes[1]._x1`,
	}
	for name, key := range keys {
		insert := append(append([]byte(`"title":"Child",`), keyLiteral(t, key)...), []byte(`:1`)...)
		raw := bytes.Replace(b, []byte(`"title":"Child"`), insert, 1)
		if _, err := phasea.DecodeStrict(raw); err == nil {
			t.Fatalf("%s: strict decoder accepted an unknown field", name)
		}
		r := diagnoseWithin(t, 10*time.Second, raw)
		if r.Valid || len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != "UNKNOWN_FIELD" {
			t.Fatalf("%s: %+v", name, r)
		}
		out, err := Encode(r)
		if err != nil || len(out) > 4096 {
			t.Fatalf("%s: report is not bounded: %d bytes, err %v", name, len(out), err)
		}
		path := r.Diagnostics[0].Path
		for _, c := range path {
			if c < 0x20 || c > 0x7e {
				t.Fatalf("%s: path carries a non-printable or non-ASCII rune %U: %q", name, c, path)
			}
		}
		if want, ok := wantPath[name]; ok && path != want {
			t.Fatalf("%s: path %q, want %q", name, path, want)
		}
		if (name == "huge" || name == "multibyte") && !strings.HasSuffix(path, `...(truncated)"]`) {
			t.Fatalf("%s: long key was not truncated: %q", name, path)
		}
	}
}

func TestPathKeyBoundaries(t *testing.T) {
	long64 := strings.Repeat("a", 64)
	cases := map[string]string{
		"title":                       ".title",
		"1abc":                        `["1abc"]`,
		"a-b":                         `["a-b"]`,
		long64:                        "." + long64,
		strings.Repeat("a", 65):       `["` + long64 + `...(truncated)"]`,
		strings.Repeat("a", 63) + "é": `["` + strings.Repeat("a", 63) + `...(truncated)"]`, // never split a rune
		"\xff":                        `["\xff"]`,
	}
	for in, want := range cases {
		if got := pathKey(in); got != want {
			t.Errorf("pathKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func FuzzDiagnoseAgreesWithStrict(f *testing.F) {
	b := valid()
	f.Add(b)
	f.Add([]byte(`{}`))
	f.Add([]byte(strings.Repeat("[", 98) + strings.Repeat("]", 98)))
	f.Add(bytes.Replace(b, []byte(`"title":"Child"`), []byte(`"title":"\ud800"`), 1))
	f.Add(bytes.Replace(b, []byte(`"title":"Child"`), []byte(`"title":"Child","k.k[0]":1`), 1))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > phasea.MaxInputBytes+1 {
			t.Skip()
		}
		r := diagnoseWithin(t, 10*time.Second, raw)
		if _, err := phasea.DecodeStrict(raw); r.Valid != (err == nil) {
			t.Fatalf("oracle disagreement: valid=%v strict=%v", r.Valid, err)
		}
		if out, err := Encode(r); err != nil || len(out) > 4096 {
			t.Fatalf("report not bounded: %d bytes, err %v", len(out), err)
		}
	})
}
