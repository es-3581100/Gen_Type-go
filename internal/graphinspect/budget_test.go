package graphinspect

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

// wire encodes a document the way an attacker would send it: compact JSON with
// no HTML escaping, so '<' costs one input byte (the canonical encoding spends
// six). The strict decoder accepts it; only ReconstructCanonical would not.
func wire(t *testing.T, d phasea.Document) []byte {
	t.Helper()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(d); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

// TestInspectWorstCaseValidInputFitsBudget builds a graph the strict decoder
// accepts at the 2 MiB input cap and inspects its busiest node. The node and its
// parent each carry 256 KiB of '<' (escaped to six bytes on output) and every
// leaf ID is echoed three times (children, descendants, leaves). Under the former
// 4 MiB cap this valid input could not be inspected at all.
func TestInspectWorstCaseValidInputFitsBudget(t *testing.T) {
	hot := strings.Repeat("<", 262144)
	build := func(leaves int) phasea.Document {
		nodes := []phasea.Node{
			{ID: "p", Order: 0, Content: hot, SourceRef: "x", SourceDigestClaim: claim},
			{ID: "x", ParentID: "p", Order: 0, Content: hot, SourceRef: "x", SourceDigestClaim: claim},
		}
		for i := 0; i < leaves; i++ {
			id := fmt.Sprintf("L%05d%s", i, strings.Repeat("z", 122)) // 128 bytes, the ID maximum
			nodes = append(nodes, phasea.Node{ID: id, ParentID: "x", Order: i, SourceRef: "x", SourceDigestClaim: claim})
		}
		return phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "s", Nodes: nodes, Edges: []phasea.Edge{}}
	}
	lo, hi := 0, 8000
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if len(wire(t, build(mid))) <= phasea.MaxInputBytes {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	doc := build(lo)
	raw := wire(t, doc)
	if _, err := phasea.DecodeStrict(raw); err != nil {
		t.Fatalf("fixture must be valid input: %v", err)
	}
	r, err := Inspect(doc, "x")
	if err != nil {
		t.Fatal(err)
	}
	out, err := Encode(r)
	if err != nil {
		t.Fatalf("a valid %d-byte graph (%d leaves) cannot be inspected: %v", len(raw), lo, err)
	}
	t.Logf("input %d bytes (cap %d) -> output %d bytes (cap %d)", len(raw), phasea.MaxInputBytes, len(out), MaxOutputBytes)
	if len(out) <= 4<<20 {
		t.Fatalf("fixture no longer exercises the regression: %d bytes fits the old cap", len(out))
	}
}

func randomForest(rnd *rand.Rand, n int) phasea.Document {
	next := map[int]int{}
	nodes := make([]phasea.Node, n)
	for i := 0; i < n; i++ {
		p := -1
		if i > 0 && rnd.Intn(5) != 0 {
			p = rnd.Intn(i)
		}
		parent := ""
		if p >= 0 {
			parent = fmt.Sprintf("n%d", p)
		}
		nodes[i] = phasea.Node{ID: fmt.Sprintf("n%d", i), ParentID: parent, Order: next[p], Title: fmt.Sprintf("t%d", i), SourceRef: "local", SourceDigestClaim: claim}
		next[p]++
	}
	kinds := []string{"links_to", "related_to", "depends_on"}
	seen := map[phasea.Edge]bool{}
	edges := []phasea.Edge{}
	for k := rnd.Intn(2*n + 1); k > 0; k-- {
		e := phasea.Edge{From: fmt.Sprintf("n%d", rnd.Intn(n)), To: fmt.Sprintf("n%d", rnd.Intn(n)), Kind: kinds[rnd.Intn(len(kinds))]}
		if !seen[e] {
			seen[e] = true
			edges = append(edges, e)
		}
	}
	rnd.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	rnd.Shuffle(len(edges), func(i, j int) { edges[i], edges[j] = edges[j], edges[i] })
	return phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "s", Nodes: nodes, Edges: edges}
}

type navigation struct {
	children, ancestors, descendants, roots, leaves []string
	incoming, outgoing                              []phasea.Edge
}

// independentNavigation recomputes everything from parent pointers and sibling
// ordinals alone, sharing no code with Inspect or phasea.Canonicalize.
func independentNavigation(doc phasea.Document, id string) navigation {
	byID := map[string]phasea.Node{}
	kids := map[string][]phasea.Node{}
	for _, n := range doc.Nodes {
		byID[n.ID] = n
		kids[n.ParentID] = append(kids[n.ParentID], n)
	}
	for _, ks := range kids {
		ks := ks
		sort.Slice(ks, func(i, j int) bool { return ks[i].Order < ks[j].Order })
	}
	var preorder func(parent string, visit func(phasea.Node))
	preorder = func(parent string, visit func(phasea.Node)) {
		for _, n := range kids[parent] {
			visit(n)
			preorder(n.ID, visit)
		}
	}
	var nav navigation
	for _, n := range kids[id] {
		nav.children = append(nav.children, n.ID)
	}
	preorder(id, func(n phasea.Node) { nav.descendants = append(nav.descendants, n.ID) })
	for p := byID[id].ParentID; p != ""; p = byID[p].ParentID {
		nav.ancestors = append([]string{p}, nav.ancestors...)
	}
	for _, n := range kids[""] {
		nav.roots = append(nav.roots, n.ID)
	}
	preorder("", func(n phasea.Node) {
		if len(kids[n.ID]) == 0 {
			nav.leaves = append(nav.leaves, n.ID)
		}
	})
	for _, e := range doc.Edges {
		if e.From == id {
			nav.outgoing = append(nav.outgoing, e)
		}
		if e.To == id {
			nav.incoming = append(nav.incoming, e)
		}
	}
	order := func(es []phasea.Edge) {
		sort.Slice(es, func(i, j int) bool {
			a, b := es[i], es[j]
			if a.From != b.From {
				return a.From < b.From
			}
			if a.Kind != b.Kind {
				return a.Kind < b.Kind
			}
			return a.To < b.To
		})
	}
	order(nav.incoming)
	order(nav.outgoing)
	return nav
}

func sameStrings(a, b []string) bool {
	return len(a) == len(b) && (len(a) == 0 || reflect.DeepEqual(a, b))
}

func sameEdges(a, b []phasea.Edge) bool {
	return len(a) == len(b) && (len(a) == 0 || reflect.DeepEqual(a, b))
}

// TestInspectMatchesIndependentOracle replaces the one-seed permutation fuzz
// with a comparison against a from-scratch implementation over random forests.
func TestInspectMatchesIndependentOracle(t *testing.T) {
	rnd := rand.New(rand.NewSource(20261108))
	for round := 0; round < 200; round++ {
		doc := randomForest(rnd, 1+rnd.Intn(40))
		_, digest, err := phasea.EncodeCanonical(doc)
		if err != nil {
			t.Fatalf("round %d: generated an invalid forest: %v", round, err)
		}
		byID := map[string]phasea.Node{}
		for _, n := range doc.Nodes {
			byID[n.ID] = n
		}
		for _, n := range doc.Nodes {
			r, err := Inspect(doc, n.ID)
			if err != nil {
				t.Fatalf("round %d node %s: %v", round, n.ID, err)
			}
			want := independentNavigation(doc, n.ID)
			switch {
			case !reflect.DeepEqual(r.Node, byID[n.ID]):
				t.Fatalf("round %d node %s: node payload differs", round, n.ID)
			case !sameStrings(r.Children, want.children):
				t.Fatalf("round %d node %s: children %v want %v", round, n.ID, r.Children, want.children)
			case !sameStrings(r.Ancestors, want.ancestors):
				t.Fatalf("round %d node %s: ancestors %v want %v", round, n.ID, r.Ancestors, want.ancestors)
			case !sameStrings(r.Descendants, want.descendants):
				t.Fatalf("round %d node %s: descendants %v want %v", round, n.ID, r.Descendants, want.descendants)
			case !sameStrings(r.Roots, want.roots):
				t.Fatalf("round %d node %s: roots %v want %v", round, n.ID, r.Roots, want.roots)
			case !sameStrings(r.Leaves, want.leaves):
				t.Fatalf("round %d node %s: leaves %v want %v", round, n.ID, r.Leaves, want.leaves)
			case !sameEdges(r.IncomingEdges, want.incoming), !sameEdges(r.OutgoingEdges, want.outgoing):
				t.Fatalf("round %d node %s: edges differ", round, n.ID)
			case r.CanonicalSHA256 != digest:
				t.Fatalf("round %d node %s: digest drift", round, n.ID)
			}
			if (n.ParentID == "") != (r.Parent == nil) || (r.Parent != nil && !reflect.DeepEqual(*r.Parent, byID[n.ParentID])) {
				t.Fatalf("round %d node %s: parent payload differs", round, n.ID)
			}
			if r.Children == nil || r.Ancestors == nil || r.Descendants == nil || r.Roots == nil || r.Leaves == nil || r.IncomingEdges == nil || r.OutgoingEdges == nil {
				t.Fatalf("round %d node %s: a report array is null", round, n.ID)
			}
		}
	}
}
