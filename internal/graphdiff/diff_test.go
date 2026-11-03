package graphdiff

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

func fixture(t *testing.T) phasea.Document {
	t.Helper()
	b, err := os.ReadFile("../../testdata/tiny_graph.json")
	if err != nil {
		t.Fatal(err)
	}
	d, err := phasea.DecodeStrict(b)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func cloned(t *testing.T, d phasea.Document) phasea.Document {
	t.Helper()
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var copy phasea.Document
	if err := json.Unmarshal(b, &copy); err != nil {
		t.Fatal(err)
	}
	return copy
}

func diffAndEncode(t *testing.T, before, after phasea.Document) (Report, []byte) {
	t.Helper()
	r, err := Compare(before, after)
	if err != nil {
		t.Fatalf("Compare(%d nodes, %d nodes): %v", len(before.Nodes), len(after.Nodes), err)
	}
	b, err := Encode(r)
	if err != nil {
		t.Fatalf("Encode(Compare(...)): %v", err)
	}
	if !json.Valid(b) || !bytes.HasSuffix(b, []byte("\n")) {
		t.Fatalf("Encode(Compare(...)) = %q, want JSON with LF", b)
	}
	var observed map[string]json.RawMessage
	if err := json.Unmarshal(b, &observed); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"metadata_changes", "added_nodes", "removed_nodes", "modified_nodes", "reparented_nodes", "child_order_changes", "added_edges", "removed_edges"} {
		if bytes.Equal(observed[key], []byte("null")) {
			t.Errorf("Encode(Compare(...)) %s = null, want array", key)
		}
	}
	return r, b
}

func TestIdenticalAndShuffledInputs(t *testing.T) {
	d := fixture(t)
	_, expectedHash, err := phasea.EncodeCanonical(d)
	if err != nil {
		t.Fatal(err)
	}
	r, b := diffAndEncode(t, d, d)
	if !r.Identical || r.BeforeSHA256 != expectedHash || r.AfterSHA256 != expectedHash || !bytes.Contains(b, []byte(`"added_nodes":[]`)) {
		t.Fatalf("Compare(d,d) = %+v; want identical and empty changes", r)
	}
	if len(r.MetadataChanges)+len(r.AddedNodes)+len(r.RemovedNodes)+len(r.ModifiedNodes)+len(r.ChildOrderChanges)+len(r.ReparentedNodes)+len(r.AddedEdges)+len(r.RemovedEdges) != 0 {
		t.Fatalf("Compare(d,d) reports changes: %+v", r)
	}
	d.Nodes[0], d.Nodes[1] = d.Nodes[1], d.Nodes[0]
	r2, shuffled := diffAndEncode(t, fixture(t), d)
	if !r2.Identical || !bytes.Equal(shuffled, b) {
		t.Fatalf("Compare(d, shuffled d) = %s, want %s", shuffled, b)
	}
}

func TestAddedRemovedAndModified(t *testing.T) {
	before := fixture(t)
	after := cloned(t, before)
	after.Nodes = after.Nodes[:1]
	after.Nodes[0].ParentID = ""
	after.Nodes[0].Order = 0
	after.Nodes[0].Title = "Changed \u003ctag>"
	after.Edges = []phasea.Edge{}
	r, b := diffAndEncode(t, before, after)
	if !slices.Equal(r.RemovedNodes, []string{"root"}) || len(r.ModifiedNodes) != 1 || !slices.Equal(r.ModifiedNodes[0].Fields, []string{"title"}) {
		t.Fatalf("Compare(remove root, promote child) = %+v", r)
	}
	if len(r.ReparentedNodes) != 1 || r.ReparentedNodes[0].BeforeParentID != "root" || r.ReparentedNodes[0].AfterParentID != "" {
		t.Fatalf("Compare(reparent child) = %+v", r.ReparentedNodes)
	}
	if len(r.RemovedEdges) != 1 || len(r.AddedEdges) != 0 {
		t.Fatalf("Compare(remove edge) = %+v", r)
	}
	if bytes.Contains(b, []byte("<tag>")) || bytes.Contains(b, []byte("Changed")) {
		t.Fatalf("Compare text field leaked raw untrusted content: %s", b)
	}
	newNode := cloned(t, before)
	newNode.Nodes = append(newNode.Nodes, phasea.Node{
		ID: "extra", ParentID: "root", Order: 1, Title: "Extra", Content: "Hi", SourceRef: "fixture.md", SourceDigestClaim: strings.Repeat("a", 64),
	})
	r, _ = diffAndEncode(t, before, newNode)
	if !slices.Equal(r.AddedNodes, []string{"extra"}) || len(r.RemovedNodes) != 0 || len(r.ChildOrderChanges) != 1 {
		t.Errorf("Compare(add extra) = %+v, want one added node and child-list change", r)
	}
}

func TestReorderMetadataAndEdges(t *testing.T) {
	before := fixture(t)
	before.Nodes = append(before.Nodes, phasea.Node{
		ID: "second", ParentID: "root", Order: 1, Title: "Second", Content: "", SourceRef: "fixture.md", SourceDigestClaim: strings.Repeat("b", 64),
	})
	after := cloned(t, before)
	for i := range after.Nodes {
		switch after.Nodes[i].ID {
		case "child":
			after.Nodes[i].Order = 1
			// Test source claim diffs as untrusted field names.
			after.Nodes[i].SourceDigestClaim = strings.Repeat("c", 64)
		case "second":
			after.Nodes[i].Order = 0
		}
	}
	after.ProjectID = "different-project"
	after.SourceID = "different-source"
	after.Edges = []phasea.Edge{{From: "second", To: "child", Kind: "related_to"}}
	r, first := diffAndEncode(t, before, after)
	if !reflect.DeepEqual(r.MetadataChanges, []MetadataChange{
		{Field: "project_id", Before: before.ProjectID, After: after.ProjectID},
		{Field: "source_id", Before: before.SourceID, After: after.SourceID},
	}) {
		t.Errorf("Compare(meta) = %+v", r.MetadataChanges)
	}
	if len(r.ChildOrderChanges) != 1 || !slices.Equal(r.ChildOrderChanges[0].Before, []string{"child", "second"}) || !slices.Equal(r.ChildOrderChanges[0].After, []string{"second", "child"}) {
		t.Errorf("Compare(reorder) = %+v", r.ChildOrderChanges)
	}
	if len(r.AddedEdges) != 1 || len(r.RemovedEdges) != 1 || len(r.ModifiedNodes) != 1 || !slices.Equal(r.ModifiedNodes[0].Fields, []string{"source_digest_claim"}) {
		t.Errorf("Compare(provenance + edge) = %+v", r)
	}
	for i := 0; i < 35; i++ {
		b := cloned(t, before)
		a := cloned(t, after)
		rand.New(rand.NewSource(int64(i))).Shuffle(len(b.Nodes), func(i, j int) { b.Nodes[i], b.Nodes[j] = b.Nodes[j], b.Nodes[i] })
		rand.New(rand.NewSource(int64(i+100))).Shuffle(len(a.Nodes), func(i, j int) { a.Nodes[i], a.Nodes[j] = a.Nodes[j], a.Nodes[i] })
		_, got := diffAndEncode(t, b, a)
		if !bytes.Equal(got, first) {
			t.Fatalf("Compare(shuffled inputs, iteration %d) != canonical diff", i)
		}
	}
}

func TestCrossImportIDCollisionIsNotTrusted(t *testing.T) {
	before := fixture(t)
	b, err := os.ReadFile("../../testdata/ambiguous_reused_node_id.json")
	if err != nil {
		t.Fatal(err)
	}
	after, err := phasea.DecodeStrict(b)
	if err != nil {
		t.Fatal(err)
	}
	r, bytes := diffAndEncode(t, before, after)
	if r.Identical || r.IdentityBoundary != IdentityBoundary || len(r.MetadataChanges) != 1 || len(r.ModifiedNodes) != 1 {
		t.Fatalf("Compare(reused ID across import) = %+v", r)
	}
	if !strings.Contains(string(bytes), "trusted_cross_import_lineage_unverified") {
		t.Errorf("Compare(reused ID) missing trust warning")
	}
}

func TestInvalidGraphsFailBeforeReporting(t *testing.T) {
	good := fixture(t)
	cases := []struct {
		name string
		edit func(*phasea.Document)
	}{
		{"duplicate ID", func(d *phasea.Document) { d.Nodes = append(d.Nodes, d.Nodes[0]) }},
		{"missing parent", func(d *phasea.Document) { d.Nodes[0].ParentID = "nonexistent" }},
		{"cycle", func(d *phasea.Document) { d.Nodes[1].ParentID = "child" }},
		{"missing edge endpoint", func(d *phasea.Document) { d.Edges[0].To = "missing" }},
		{"duplicate edge", func(d *phasea.Document) { d.Edges = append(d.Edges, d.Edges[0]) }},
		{"invalid ordinal", func(d *phasea.Document) { d.Nodes[0].Order = 42 }},
		{"excess nodes", func(d *phasea.Document) { d.Nodes = make([]phasea.Node, phasea.MaxNodes+1) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			invalid := cloned(t, good)
			tc.edit(&invalid)
			if r, err := Compare(good, invalid); err == nil {
				t.Errorf("Compare(good, %s) = %+v, want error", tc.name, r)
			}
			if r, err := Compare(invalid, good); err == nil {
				t.Errorf("Compare(%s, good) = %+v, want error", tc.name, r)
			}
		})
	}
}

func TestLargeGraphAndInputImmutability(t *testing.T) {
	before := fixture(t)
	before.Edges = []phasea.Edge{}
	before.Nodes = []phasea.Node{before.Nodes[1]}
	const count = 4000
	for i := 0; i < count; i++ {
		before.Nodes = append(before.Nodes, phasea.Node{
			ID: fmtID(i), ParentID: "root", Order: i, Title: "fixture", SourceRef: "fixture.md", SourceDigestClaim: strings.Repeat("a", 64),
		})
	}
	after := cloned(t, before)
	// Valid reverse order: the identity of every child remains synthetic.
	for i := 1; i < len(after.Nodes); i++ {
		after.Nodes[i].Order = count - i
	}
	oldBytes, _ := json.Marshal(before)
	newBytes, _ := json.Marshal(after)
	r, result := diffAndEncode(t, before, after)
	if r.Identical || len(r.ChildOrderChanges) != 1 || len(r.ChildOrderChanges[0].After) != count || len(result) > MaxOutputBytes {
		t.Errorf("Compare(large graph) = %+v / %d bytes", r.IdentityBoundary, len(result))
	}
	currentBefore, _ := json.Marshal(before)
	currentAfter, _ := json.Marshal(after)
	if !bytes.Equal(oldBytes, currentBefore) || !bytes.Equal(newBytes, currentAfter) {
		t.Error("Compare mutated input document")
	}
}

func fmtID(i int) string { return "id-" + strconv.Itoa(i) }

func TestEncodedOutputLimit(t *testing.T) {
	r := Report{AddedNodes: []string{strings.Repeat("x", MaxOutputBytes)}}
	if b, err := Encode(r); err == nil || b != nil {
		t.Fatalf("Encode(oversize diff) = %d bytes, %v; want nil, error", len(b), err)
	}
}

// FuzzCompare checks that accepted raw graph dialects stay deterministic and
// that syntactically different inputs are never promoted to trusted lineage.
func FuzzCompare(f *testing.F) {
	a, _ := os.ReadFile("../../testdata/tiny_graph.json")
	b, _ := os.ReadFile("../../testdata/ambiguous_reused_node_id.json")
	f.Add(a, a)
	f.Add(a, b)
	f.Add([]byte("{}"), a)
	f.Fuzz(func(t *testing.T, rawBefore, rawAfter []byte) {
		if len(rawBefore) > phasea.MaxInputBytes || len(rawAfter) > phasea.MaxInputBytes {
			return
		}
		before, err := phasea.DecodeStrict(rawBefore)
		if err != nil {
			return
		}
		after, err := phasea.DecodeStrict(rawAfter)
		if err != nil {
			return
		}
		r1, err := Compare(before, after)
		if err != nil {
			t.Fatal(err)
		}
		r2, err := Compare(before, after)
		if err != nil {
			t.Fatal(err)
		}
		x, err := Encode(r1)
		if err != nil {
			t.Fatal(err)
		}
		y, err := Encode(r2)
		if err != nil || !bytes.Equal(x, y) || r1.IdentityBoundary != IdentityBoundary {
			t.Fatalf("Compare(non-mutated inputs) non-deterministic or lost trust warning")
		}
	})
}

func TestReparentingBetweenExistingParents(t *testing.T) {
	before := fixture(t)
	before.Nodes = append(before.Nodes, phasea.Node{
		ID: "other", ParentID: "root", Order: 1, Title: "Other", SourceRef: "fixture.md", SourceDigestClaim: strings.Repeat("a", 64),
	})
	after := cloned(t, before)
	for i := range after.Nodes {
		switch after.Nodes[i].ID {
		case "child":
			after.Nodes[i].ParentID = "other"
		case "other":
			after.Nodes[i].Order = 0
		}
	}
	r, _ := diffAndEncode(t, before, after)
	if !reflect.DeepEqual(r.ReparentedNodes, []ReparentedNode{{ID: "child", BeforeParentID: "root", AfterParentID: "other"}}) {
		t.Errorf("Compare(reparent child) = %+v", r.ReparentedNodes)
	}
	if len(r.ChildOrderChanges) != 2 || r.ChildOrderChanges[0].ParentID != "other" || r.ChildOrderChanges[1].ParentID != "root" {
		t.Errorf("Compare(reparent) child lists = %+v, want other then root", r.ChildOrderChanges)
	}
}

func FuzzGeneratedValidGraphDiff(f *testing.F) {
	f.Add(uint8(0), uint8(0), false)
	f.Add(uint8(3), uint8(2), true)
	f.Add(uint8(12), uint8(12), false)
	f.Fuzz(func(t *testing.T, a, b uint8, differentSource bool) {
		before := phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "s", Nodes: []phasea.Node{}, Edges: []phasea.Edge{}}
		before.Nodes = append(before.Nodes, phasea.Node{ID: "root", Order: 0, SourceRef: "fixture", SourceDigestClaim: strings.Repeat("a", 64)})
		for i := 0; i < int(a)%32; i++ {
			before.Nodes = append(before.Nodes, phasea.Node{ID: fmtID(i), ParentID: "root", Order: i, SourceRef: "fixture", SourceDigestClaim: strings.Repeat("a", 64)})
		}
		after := cloned(t, before)
		want := int(b) % 32
		after.Nodes = after.Nodes[:1]
		for i := 0; i < want; i++ {
			after.Nodes = append(after.Nodes, phasea.Node{ID: fmtID(i), ParentID: "root", Order: i, SourceRef: "fixture", SourceDigestClaim: strings.Repeat("a", 64)})
		}
		if differentSource {
			after.SourceID = "another"
		}
		r, err := Compare(before, after)
		if err != nil {
			t.Fatal(err)
		}
		if len(r.AddedNodes) != max(0, want-(int(a)%32)) || len(r.RemovedNodes) != max(0, (int(a)%32)-want) || r.IdentityBoundary != IdentityBoundary {
			t.Errorf("Compare(generated %d/%d): %+v, want accurate adds/removes and trust warning", int(a)%32, want, r)
		}
		out, err := Encode(r)
		if err != nil || !json.Valid(out) {
			t.Fatalf("Encode(Compare(generated)): %v", err)
		}
	})
}
