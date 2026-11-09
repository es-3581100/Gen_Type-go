package graphinspect

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

const claim = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func forest() phasea.Document {
	return phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "unverified", Nodes: []phasea.Node{
		{ID: "c", ParentID: "a", Order: 0, Title: "C", SourceRef: "local", SourceDigestClaim: claim},
		{ID: "r2", ParentID: "", Order: 1, Title: "R2", SourceRef: "local", SourceDigestClaim: claim},
		{ID: "b", ParentID: "r1", Order: 1, Title: "B", SourceRef: "local", SourceDigestClaim: claim},
		{ID: "a", ParentID: "r1", Order: 0, Title: "A", SourceRef: "local", SourceDigestClaim: claim},
		{ID: "r1", ParentID: "", Order: 0, Title: "R1", SourceRef: "local", SourceDigestClaim: claim},
	}, Edges: []phasea.Edge{{From: "r1", To: "r2", Kind: "depends_on"}, {From: "c", To: "c", Kind: "related_to"}, {From: "a", To: "c", Kind: "links_to"}}}
}
func TestInspectNavigation(t *testing.T) {
	d := forest()
	r, err := Inspect(d, "c")
	if err != nil {
		t.Fatal(err)
	}
	if r.Node.ID != "c" || r.Parent == nil || r.Parent.ID != "a" {
		t.Fatalf("node/parent: %+v", r)
	}
	if !reflect.DeepEqual(r.Ancestors, []string{"r1", "a"}) || len(r.Descendants) != 0 || len(r.Children) != 0 {
		t.Fatalf("ancestors: %+v", r)
	}
	if !reflect.DeepEqual(r.Roots, []string{"r1", "r2"}) || !reflect.DeepEqual(r.Leaves, []string{"c", "b", "r2"}) {
		t.Fatalf("roots/leaves: %+v", r)
	}
	if len(r.IncomingEdges) != 2 || len(r.OutgoingEdges) != 1 || r.IncomingEdges[0].From != "a" || r.IncomingEdges[1].From != "c" {
		t.Fatalf("edges: %+v", r)
	}
	root, err := Inspect(d, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if root.Parent != nil || !reflect.DeepEqual(root.Children, []string{"a", "b"}) || !reflect.DeepEqual(root.Descendants, []string{"a", "c", "b"}) || len(root.OutgoingEdges) != 1 {
		t.Fatalf("root navigation: %+v", root)
	}
	if r.CanonicalSHA256 != root.CanonicalSHA256 || r.IdentityBoundary != IdentityBoundary {
		t.Fatal("digest or trust boundary drift")
	}
}
func TestInspectPermutationAndErrors(t *testing.T) {
	d := forest()
	original, _ := json.Marshal(d)
	r, err := Inspect(d, "c")
	if err != nil {
		t.Fatal(err)
	}
	want, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(want, []byte("\n")) {
		t.Fatal("no LF")
	}
	for i := 0; i < 100; i++ {
		shuffled := forest()
		rnd := rand.New(rand.NewSource(int64(i)))
		rnd.Shuffle(len(shuffled.Nodes), func(i, j int) { shuffled.Nodes[i], shuffled.Nodes[j] = shuffled.Nodes[j], shuffled.Nodes[i] })
		rnd.Shuffle(len(shuffled.Edges), func(i, j int) { shuffled.Edges[i], shuffled.Edges[j] = shuffled.Edges[j], shuffled.Edges[i] })
		again, err := Inspect(shuffled, "c")
		if err != nil {
			t.Fatal(err)
		}
		out, err := Encode(again)
		if err != nil || !bytes.Equal(out, want) {
			t.Fatalf("permutation %d drift: %v", i, err)
		}
	}
	after, _ := json.Marshal(d)
	if !bytes.Equal(original, after) {
		t.Fatal("input mutation")
	}
	if _, err := Inspect(d, "no"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("unknown node: %v", err)
	}
	d.Nodes[0].ParentID = "missing"
	if _, err := Inspect(d, "c"); err == nil {
		t.Fatal("accepted invalid graph")
	}
	large := r
	large.Node.Content = strings.Repeat("a", MaxOutputBytes)
	if _, err := Encode(large); err == nil {
		t.Fatal("accepted oversize output")
	}
}
func FuzzInspectPermutation(f *testing.F) {
	f.Add(uint8(0))
	f.Add(uint8(1))
	f.Add(uint8(255))
	f.Fuzz(func(t *testing.T, seed uint8) {
		a, b := forest(), forest()
		rnd := rand.New(rand.NewSource(int64(seed)))
		rnd.Shuffle(len(b.Nodes), func(i, j int) { b.Nodes[i], b.Nodes[j] = b.Nodes[j], b.Nodes[i] })
		rnd.Shuffle(len(b.Edges), func(i, j int) { b.Edges[i], b.Edges[j] = b.Edges[j], b.Edges[i] })
		x, err := Inspect(a, "r1")
		if err != nil {
			t.Fatal(err)
		}
		y, err := Inspect(b, "r1")
		if err != nil {
			t.Fatal(err)
		}
		xx, _ := Encode(x)
		yy, _ := Encode(y)
		if !bytes.Equal(xx, yy) {
			t.Fatal("non-determinism")
		}
	})
}
