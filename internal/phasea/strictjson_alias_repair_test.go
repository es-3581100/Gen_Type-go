package phasea

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// This suite intentionally inverts the adversarial reproducer assertions:
// every semantic alias must now FAIL strict decoding, regardless of order.
// It covers every declared graph field at every object depth.
func TestRepair001RejectsCaseAliasedKeysAtAllLevels(t *testing.T) {
	base, _, err := EncodeCanonical(fixture())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ path, canonical, alias, value string }{
		{"graph-schema", "schema", "SCHEMA", `"gen-type-go/phase-a-graph-v0"`},
		{"graph-project", "project_id", "PROJECT_ID", `"other-project"`},
		{"graph-source", "source_id", "SOURCE_ID", `"other-source"`},
		{"graph-nodes", "nodes", "NODES", `[]`},
		{"graph-edges", "edges", "EDGES", `[]`},
		{"node-id", "id", "ID", `"hijack"`},
		{"node-parent", "parent_id", "PARENT_ID", `"hijack"`},
		{"node-order", "order", "ORDER", `1`},
		{"node-title", "title", "TITLE", `"substituted-title"`},
		{"node-content", "content", "CONTENT", `"substituted-content"`},
		{"node-ref", "source_ref", "SOURCE_REF", `"other.md"`},
		{"node-digest", "source_digest_claim", "SOURCE_DIGEST_CLAIM", `"` + strings.Repeat("a", 64) + `"`},
		{"edge-from", "from", "FROM", `"peer"`},
		{"edge-to", "to", "TO", `"peer"`},
		{"edge-kind", "kind", "KIND", `"related_to"`},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			// Inject into the first object containing the named field, not all
			// objects. The fixture contains nodes and edges for nested cases.
			needle := []byte(`"` + tc.canonical + `":`)
			idx := bytes.Index(base, needle)
			if idx < 0 {
				t.Fatalf("fixture missing %s", tc.canonical)
			}
			for _, reversed := range []bool{false, true} {
				aliasPart := []byte(`"` + tc.alias + `":` + tc.value + `,`)
				var hostile []byte
				if reversed {
					// Alias appended AFTER canonical field; parser must reject
					// even when encoding/json would override the value.
					// Insert immediately after the canonical name and value.
					hostile = spliceAfterJSONField(base, idx, tc.alias, tc.value)
				} else {
					hostile = append(append([]byte{}, base[:idx]...), append(aliasPart, base[idx:]...)...)
				}
				if bytes.Equal(hostile, base) {
					t.Fatal("fixture injection failed")
				}
				if _, err := DecodeStrict(hostile); err == nil {
					t.Fatalf("accepted %s alias (%q), reversed=%t", tc.canonical, tc.alias, reversed)
				} else if !strings.Contains(err.Error(), "undeclared or case-aliased JSON field") {
					t.Fatalf("rejected via unrelated error, %s reversed=%t: %v", tc.canonical, reversed, err)
				}
			}
		})
	}
}

// Extracts and finds the end of one JSON field with a token decoder; the
// alternate key is inserted without changing existing field values or order.
func spliceAfterJSONField(raw []byte, start int, alias, replacement string) []byte {
	p := start + bytes.IndexByte(raw[start:], ':') + 1
	dec := json.NewDecoder(bytes.NewReader(raw[p:]))
	var value json.RawMessage
	if err := dec.Decode(&value); err != nil {
		panic(err)
	}
	end := p + int(dec.InputOffset())
	return append(append([]byte{}, raw[:end]...), append([]byte(`,"`+alias+`":`+replacement), raw[end:]...)...)
}

func TestRepair001EscapedCaseAliasesFailAndCanonicalNamesWork(t *testing.T) {
	base, _, err := EncodeCanonical(fixture())
	if err != nil {
		t.Fatal(err)
	}
	const prefix = `"project_id":"project-1"`
	if !bytes.Contains(base, []byte(prefix)) {
		t.Fatal("missing project id")
	}
	for _, extra := range []string{
		`"PROJECT_ID":"alternate"`,
		`"projecT_id":"alternate"`,
		`"\u0050ROJECT_ID":"alternate"`,
		`"\u0070roject_id":"alternate"`, // semantically duplicate exact name
	} {
		hostile := bytes.Replace(base, []byte(prefix), []byte(prefix+`,`+extra), 1)
		if _, err := DecodeStrict(hostile); err == nil {
			t.Fatalf("accepted ambiguous key %q", extra)
		} else if !strings.Contains(err.Error(), "undeclared or case-aliased JSON field") && !strings.Contains(err.Error(), "duplicate JSON key") {
			t.Fatalf("unexpected rejection reason for %s: %v", extra, err)
		}
	}
	// Escaped spelling of the SAME exact allowed key (not a second key)
	// is unambiguous and must not be spuriously rejected.
	validEscaped := bytes.Replace(base, []byte(`"project_id":`), []byte(`"\u0070roject_id":`), 1)
	got, err := DecodeStrict(validEscaped)
	if err != nil || got.ProjectID != "project-1" {
		t.Fatalf("escaped canonical key rejected: %v", err)
	}
	// Graph fields and node text remain case-sensitive *values* only, not
	// an arbitrary requirement to lowercase content or identifiers.
	validContent := bytes.Replace(base, []byte(`"title":"Root"`), []byte(`"title":"TITLE"`), 1)
	if _, err := DecodeStrict(validContent); err != nil {
		t.Fatalf("valid title value rejected: %v", err)
	}
}
