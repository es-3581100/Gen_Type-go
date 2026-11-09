package graphinspect

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

// These tests are deliberate tripwires for the conditional proof in
// docs/GRAPH_INSPECTION_OUTPUT_BOUND_V0.4.md. They check the assumptions against
// actual phasea validation, JSON encoding and source-level edge-kind inventory,
// rather than just copying the algebra into a test.
const boundDigest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func boundDocument(n phasea.Node) phasea.Document {
	return phasea.Document{
		Schema: phasea.Schema, ProjectID: "p", SourceID: "s",
		Nodes: []phasea.Node{n}, Edges: []phasea.Edge{},
	}
}

func boundNode() phasea.Node {
	return phasea.Node{ID: "r", ParentID: "", Order: 0, SourceRef: "x", SourceDigestClaim: boundDigest}
}

func boundRequireValid(t *testing.T, d phasea.Document) {
	t.Helper()
	if err := phasea.Validate(d); err != nil {
		t.Fatalf("proof contract: expected valid document: %v", err)
	}
}

func boundRequireInvalid(t *testing.T, d phasea.Document) {
	t.Helper()
	if err := phasea.Validate(d); err == nil {
		t.Fatal("proof contract drift: oversized or prohibited document was accepted; review output bound")
	}
}

func TestInspectionOutputBoundSourceContracts(t *testing.T) {
	t.Run("input and output caps", func(t *testing.T) {
		if phasea.MaxInputBytes != 2<<20 || phasea.MaxNodes != 10_000 || MaxOutputBytes != 8<<20 {
			t.Fatalf("proof caps changed: input=%d nodes=%d output=%d; re-derive the bound", phasea.MaxInputBytes, phasea.MaxNodes, MaxOutputBytes)
		}
		if phasea.Schema != "gen-type-go/phase-a-graph-v0" || Schema != "gen-type-go/graph-inspection-v0" {
			t.Fatal("schema changed; recheck the envelope and report skeleton")
		}
		minimalEnvelope := []byte(`{"schema":"` + phasea.Schema + `","project_id":"p","source_id":"s","nodes":[],"edges":[]}`)
		if !json.Valid(minimalEnvelope) || len(minimalEnvelope) != 96 {
			t.Fatalf("minimal input envelope changed from 96 bytes to %d", len(minimalEnvelope))
		}
		valid, err := json.Marshal(boundDocument(boundNode()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := phasea.DecodeStrict(valid); err != nil {
			t.Fatal(err)
		}
		padded := append(append([]byte{}, valid...), bytes.Repeat([]byte(" "), phasea.MaxInputBytes-len(valid))...)
		if _, err := phasea.DecodeStrict(padded); err != nil {
			t.Fatalf("exactly MaxInputBytes unexpectedly rejected: %v", err)
		}
		if _, err := phasea.DecodeStrict(append(padded, ' ')); err == nil {
			t.Fatal("MaxInputBytes + 1 unexpectedly accepted")
		}
	})

	t.Run("fixed JSON report skeleton and per-entry costs", func(t *testing.T) {
		nodeJSON, err := json.Marshal(phasea.Node{})
		if err != nil || len(nodeJSON) != 99 {
			t.Fatalf("node skeleton changed: %d bytes, %v", len(nodeJSON), err)
		}
		report := Report{
			Schema: Schema, CanonicalSHA256: strings.Repeat("0", 64), IdentityBoundary: IdentityBoundary,
			Children: []string{}, Ancestors: []string{}, Descendants: []string{},
			Roots: []string{}, Leaves: []string{}, IncomingEdges: []phasea.Edge{}, OutgoingEdges: []phasea.Edge{},
		}
		reportJSON, err := Encode(report)
		if err != nil || len(reportJSON) != 470 || reportJSON[len(reportJSON)-1] != '\n' {
			t.Fatalf("report skeleton (including LF) changed: %d bytes, %v", len(reportJSON), err)
		}
		maxLeaf := boundNode()
		maxLeaf.ID = strings.Repeat("z", 128)
		maxLeaf.ParentID = "r"
		leafJSON, err := json.Marshal(maxLeaf)
		if err != nil || len(leafJSON)+1 != 294 { // 166 + 128; includes one array delimiter
			t.Fatalf("non-root, 128-byte ID wire cost changed: %d bytes, %v", len(leafJSON)+1, err)
		}
		maxEdge := phasea.Edge{From: strings.Repeat("z", 128), To: strings.Repeat("z", 128), Kind: "related_to"}
		edgeJSON, err := json.Marshal(maxEdge)
		if err != nil || len(edgeJSON)+1 != 296 {
			t.Fatalf("maximal edge wire cost changed: %d bytes, %v", len(edgeJSON)+1, err)
		}
	})

	t.Run("field budgets are actual validator boundaries", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			max  int
			set  func(*phasea.Node, string)
		}{
			{"title", 4096, func(n *phasea.Node, v string) { n.Title = v }},
			{"content", 262144, func(n *phasea.Node, v string) { n.Content = v }},
			{"source_ref", 4096, func(n *phasea.Node, v string) { n.SourceRef = v }},
		} {
			t.Run(tc.name, func(t *testing.T) {
				n := boundNode()
				tc.set(&n, strings.Repeat("<", tc.max))
				boundRequireValid(t, boundDocument(n))
				tc.set(&n, strings.Repeat("<", tc.max+1))
				boundRequireInvalid(t, boundDocument(n))
			})
		}
		n := boundNode()
		n.SourceRef = ""
		boundRequireInvalid(t, boundDocument(n))
	})

	t.Run("identifier alphabet, length and digest", func(t *testing.T) {
		n := boundNode()
		n.ID = strings.Repeat("z", 128)
		boundRequireValid(t, boundDocument(n))
		n.ID += "z"
		boundRequireInvalid(t, boundDocument(n))
		n.ID = "r<"
		boundRequireInvalid(t, boundDocument(n))
		n.ID = "é"
		boundRequireInvalid(t, boundDocument(n))
		n.ID = "A1._:@/-z"
		boundRequireValid(t, boundDocument(n))
		n.SourceDigestClaim = strings.Repeat("a", 63)
		boundRequireInvalid(t, boundDocument(n))
		n.SourceDigestClaim = strings.Repeat("A", 64)
		boundRequireInvalid(t, boundDocument(n))
		n.SourceDigestClaim = boundDigest
		parent := boundNode()
		parent.ID = strings.Repeat("p", 128)
		n.ID, n.ParentID, n.Order = "c", parent.ID, 0
		d := boundDocument(parent)
		d.Nodes = append(d.Nodes, n)
		boundRequireValid(t, d)
		d.Nodes[1].ParentID += "p"
		boundRequireInvalid(t, d)
	})

	t.Run("JSON escaping on strict accepted inputs", func(t *testing.T) {
		n := boundNode()
		n.Title, n.Content, n.SourceRef = "<", "<", "<"
		var raw bytes.Buffer
		enc := json.NewEncoder(&raw)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(boundDocument(n)); err != nil {
			t.Fatal(err)
		}
		if bytes.Count(raw.Bytes(), []byte("<")) != 3 {
			t.Fatalf("unexpected unescaped wire fixture: %q", raw.Bytes())
		}
		d, err := phasea.DecodeStrict(raw.Bytes())
		if err != nil {
			t.Fatalf("strict decoder unexpectedly rejected cheap wire encoding: %v", err)
		}
		got, err := json.Marshal(d.Nodes[0])
		if err != nil || bytes.Count(got, []byte(`\u003c`)) != 3 {
			t.Fatalf("HTML escaping contract changed: %q (%v)", got, err)
		}
	})

	t.Run("all seven node fields are required on the wire", func(t *testing.T) {
		raw, err := json.Marshal(boundDocument(boundNode()))
		if err != nil {
			t.Fatal(err)
		}
		var top map[string]json.RawMessage
		if err := json.Unmarshal(raw, &top); err != nil {
			t.Fatal(err)
		}
		var nodes []map[string]json.RawMessage
		if err := json.Unmarshal(top["nodes"], &nodes); err != nil || len(nodes) != 1 {
			t.Fatal(err)
		}
		for _, key := range []string{"id", "parent_id", "order", "title", "content", "source_ref", "source_digest_claim"} {
			copyNode := make(map[string]json.RawMessage, len(nodes[0]))
			for k, v := range nodes[0] {
				if k != key {
					copyNode[k] = v
				}
			}
			without, err := json.Marshal([]map[string]json.RawMessage{copyNode})
			if err != nil {
				t.Fatal(err)
			}
			top["nodes"] = without
			malformed, err := json.Marshal(top)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := phasea.DecodeStrict(malformed); err == nil {
				t.Fatalf("key %q became optional; the strict wire-cost proof requires review", key)
			}
		}
	})

	t.Run("edge-kind inventory and self-loop bound", func(t *testing.T) {
		want := []string{"depends_on", "links_to", "related_to"}
		got := boundEdgeKindsInPhaseASource(t)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("allowed edge kinds changed: %v, wanted %v; self-loop multiplier must be re-derived", got, want)
		}
		d := boundDocument(boundNode())
		for _, kind := range want {
			d.Edges = append(d.Edges, phasea.Edge{From: "r", To: "r", Kind: kind})
		}
		boundRequireValid(t, d)
		d.Edges = append(d.Edges, d.Edges[0])
		boundRequireInvalid(t, d) // a fourth identical typed self-loop is forbidden
		d.Edges[len(d.Edges)-1].Kind = "another_kind"
		boundRequireInvalid(t, d)
	})

	t.Run("exact rational output bound", func(t *testing.T) {
		const (
			maxPayload = 2 * (4096 + 262144 + 4096)
			k          = 1791
			rhoNum     = 131
			rhoDen     = 98
			wireOffset = 94 // 96-byte envelope, minus two missing final commas
		)
		// Integer-only arithmetic prevents a floating-point rounding mistake.
		numerator := k*rhoDen + rhoNum*(phasea.MaxInputBytes-wireOffset) + (6*rhoDen-rhoNum)*maxPayload
		floor, ceil := numerator/rhoDen, (numerator+rhoDen-1)/rhoDen
		if floor != 5_326_298 || ceil != 5_326_299 {
			t.Fatalf("proof constants changed: floor=%d ceil=%d; re-derive", floor, ceil)
		}
		if ceil > MaxOutputBytes {
			t.Fatalf("8 MiB inspection cap no longer covers upper bound: ceil=%d cap=%d", ceil, MaxOutputBytes)
		}
	})
}

// edgeKinds is unexported. A pure behavioral probe against one made-up kind
// cannot detect adding an *unrelated* fourth supported kind; parse the actual
// package-level initializer so that a new kind (or a refactor) forces review.
func boundEdgeKindsInPhaseASource(t *testing.T) []string {
	t.Helper()
	_, ownFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate proof contract source")
	}
	path := filepath.Join(filepath.Dir(ownFile), "..", "phasea", "graph.go")
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("cannot inspect edge-kind contract: %v", err)
	}
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range value.Names {
				if name.Name != "edgeKinds" || i >= len(value.Values) {
					continue
				}
				literal, ok := value.Values[i].(*ast.CompositeLit)
				if !ok {
					t.Fatal("edgeKinds is no longer a literal map; re-evaluate the output bound")
				}
				var kinds []string
				for _, element := range literal.Elts {
					entry, ok := element.(*ast.KeyValueExpr)
					if !ok {
						t.Fatal("unexpected edgeKinds entry")
					}
					key, ok := entry.Key.(*ast.BasicLit)
					if !ok || key.Kind != token.STRING {
						t.Fatal("edgeKinds contains a nonliteral key")
					}
					kind, err := strconv.Unquote(key.Value)
					if err != nil {
						t.Fatal(err)
					}
					kinds = append(kinds, kind)
				}
				sort.Strings(kinds)
				return kinds
			}
		}
	}
	t.Fatal("edgeKinds literal not found; the mathematical assumption requires review")
	return nil
}
