package graphstats

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

const claim = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func node(id, parent string, order int) phasea.Node {
	return phasea.Node{ID: id, ParentID: parent, Order: order, Title: id, Content: "text", SourceRef: "unverified.md", SourceDigestClaim: claim}
}

func sample() phasea.Document {
	return phasea.Document{
		Schema: phasea.Schema, ProjectID: "local-demo", SourceID: "unverified-source",
		Nodes: []phasea.Node{
			node("d", "r2", 0), node("c", "a", 0), node("r3", "", 2),
			node("a", "r1", 0), node("r2", "", 1), node("b", "r1", 1), node("r1", "", 0),
		},
		Edges: []phasea.Edge{
			{From: "c", To: "c", Kind: "related_to"},
			{From: "r1", To: "r2", Kind: "depends_on"},
		},
	}
}

func analyzeAndEncode(t *testing.T, d phasea.Document) (Report, []byte) {
	t.Helper()
	r, err := Analyze(d)
	if err != nil {
		t.Fatalf("Analyze(%d nodes): %v", len(d.Nodes), err)
	}
	b, err := Encode(r)
	if err != nil {
		t.Fatalf("Encode(Analyze(...)): %v", err)
	}
	if !json.Valid(b) || !bytes.HasSuffix(b, []byte("\n")) {
		t.Fatalf("Encode(Analyze(...)) = %q, want valid JSON and LF", b)
	}
	for _, v := range []string{"depth_distribution", "branching_factor_distribution", "weak_component_sizes", "explicit_edge_kinds"} {
		if bytes.Contains(b, []byte(`"`+v+`":null`)) {
			t.Errorf("Encode(Analyze(...)) %s = null, want JSON array", v)
		}
	}
	return r, b
}

func TestMultiRootAndWeakConnectivity(t *testing.T) {
	d := sample()
	r, data := analyzeAndEncode(t, d)
	if r.NodeCount != 7 || r.ExplicitEdgeCount != 2 || r.HierarchyEdgeCount != 4 || r.RootCount != 3 || r.LeafCount != 4 {
		t.Errorf("Analyze(multi-root) counts = %+v, want (7,2,4,3,4)", r)
	}
	if r.MaxDepth != 2 || r.MaxBranchingFactor != 2 || r.IsolatedNodeCount != 1 || r.ExplicitSelfLoopCount != 1 {
		t.Errorf("Analyze(multi-root) tree metrics = %+v, want (2,2,1,1)", r)
	}
	if !reflect.DeepEqual(r.DepthDistribution, []Bucket{{0, 3}, {1, 3}, {2, 1}}) {
		t.Errorf("Analyze(multi-root).depth_distribution = %v, want [0:3,1:3,2:1]", r.DepthDistribution)
	}
	if !reflect.DeepEqual(r.BranchingFactorDistribution, []Bucket{{0, 4}, {1, 2}, {2, 1}}) {
		t.Errorf("Analyze(multi-root).branching_factor_distribution = %v, want [0:4,1:2,2:1]", r.BranchingFactorDistribution)
	}
	if r.MeanBranchingFactor != (Ratio{4, 7}) {
		t.Errorf("Analyze(multi-root).mean_branching_factor = %+v, want 4/7", r.MeanBranchingFactor)
	}
	if r.WeakComponentCount != 2 || !slices.Equal(r.WeakComponentSizes, []int{6, 1}) {
		t.Errorf("Analyze(multi-root).weak_components = (%d,%v), want (2,[6,1])", r.WeakComponentCount, r.WeakComponentSizes)
	}
	if !reflect.DeepEqual(r.ExplicitEdgeKinds, []KindCount{{"depends_on", 1}, {"related_to", 1}}) {
		t.Errorf("Analyze(multi-root).explicit_edge_kinds = %v, want sorted counts", r.ExplicitEdgeKinds)
	}
	if r.Schema != Schema || r.IdentityBoundary != IdentityBoundary || !bytes.Contains(data, []byte(`"identity_boundary":"local_graph_structure_only`)) {
		t.Errorf("Analyze(multi-root) identity boundary missing: %+v", r)
	}
	if bytes.Contains(data, []byte("unverified.md")) || bytes.Contains(data, []byte("text")) || bytes.Contains(data, []byte("local-demo")) {
		t.Errorf("Encode(Analyze(multi-root)) exposed source or node field content: %s", data)
	}
}

func TestAllIsolatedRootsAndSingleNode(t *testing.T) {
	d := phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "s", Nodes: []phasea.Node{node("a", "", 0), node("b", "", 1), node("c", "", 2)}, Edges: []phasea.Edge{}}
	r, _ := analyzeAndEncode(t, d)
	if r.IsolatedNodeCount != 3 || r.WeakComponentCount != 3 || !slices.Equal(r.WeakComponentSizes, []int{1, 1, 1}) || r.HierarchyEdgeCount != 0 || r.LeafCount != 3 || r.MaxDepth != 0 {
		t.Errorf("Analyze(three isolated roots) = %+v, want three singleton components", r)
	}
	if len(r.ExplicitEdgeKinds) != 0 || !reflect.DeepEqual(r.DepthDistribution, []Bucket{{0, 3}}) || !reflect.DeepEqual(r.BranchingFactorDistribution, []Bucket{{0, 3}}) {
		t.Errorf("Analyze(three isolated roots) distributions = %+v", r)
	}
	d.Nodes = d.Nodes[:1]
	r, _ = analyzeAndEncode(t, d)
	if r.MeanBranchingFactor != (Ratio{0, 1}) || r.RootCount != 1 || r.LeafCount != 1 {
		t.Errorf("Analyze(single root) = %+v", r)
	}
}

func TestInputPermutationAndMutationSafety(t *testing.T) {
	d := sample()
	original, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	r, want := analyzeAndEncode(t, d)
	current, _ := json.Marshal(d)
	if !bytes.Equal(current, original) {
		t.Fatal("Analyze modified its input")
	}
	for i := 0; i < 48; i++ {
		a := sample()
		rng := rand.New(rand.NewSource(int64(i)))
		rng.Shuffle(len(a.Nodes), func(i, j int) { a.Nodes[i], a.Nodes[j] = a.Nodes[j], a.Nodes[i] })
		rng.Shuffle(len(a.Edges), func(i, j int) { a.Edges[i], a.Edges[j] = a.Edges[j], a.Edges[i] })
		_, got := analyzeAndEncode(t, a)
		if !bytes.Equal(got, want) {
			t.Fatalf("Analyze(reordered nodes/edges, seed %d) differs:\ngot %s\nwant %s", i, got, want)
		}
	}
	changed := sample()
	changed.Nodes[0].Title = "<script>alert(1)</script>"
	r2, b := analyzeAndEncode(t, changed)
	if r.CanonicalSHA256 == r2.CanonicalSHA256 {
		t.Error("Analyze(content change) kept canonical SHA-256")
	}
	if bytes.Contains(b, []byte("<script>")) {
		t.Error("Analyze(content change) leaked text into stats output")
	}
	changed.Nodes[0].SourceDigestClaim = strings.Repeat("b", 64)
	_, _ = analyzeAndEncode(t, changed)
}

func TestInvalidDocumentsFailClosed(t *testing.T) {
	cases := []struct {
		name   string
		change func(*phasea.Document)
	}{
		{"duplicate id", func(d *phasea.Document) { d.Nodes[1].ID = d.Nodes[0].ID }},
		{"dangling parent", func(d *phasea.Document) { d.Nodes[0].ParentID = "missing" }},
		{"disconnected cycle", func(d *phasea.Document) {
			d.Nodes[0].ParentID = "a"
			d.Nodes[0].Order = 1
			d.Nodes[3].ParentID = "d"
			d.Nodes[5].Order = 0
		}},
		{"duplicate edge", func(d *phasea.Document) { d.Edges = append(d.Edges, d.Edges[0]) }},
		{"too many nodes", func(d *phasea.Document) { d.Nodes = make([]phasea.Node, phasea.MaxNodes+1) }},
		{"too many edges", func(d *phasea.Document) { d.Edges = make([]phasea.Edge, phasea.MaxEdges+1) }},
		{"nil nodes", func(d *phasea.Document) { d.Nodes = nil }},
		{"nil edges", func(d *phasea.Document) { d.Edges = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := sample()
			tc.change(&d)
			if r, err := Analyze(d); err == nil {
				t.Errorf("Analyze(%s) = %+v, want nonnil validation error", tc.name, r)
			}
		})
	}
}

func TestDeepChain(t *testing.T) {
	const n = 1200
	d := phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "s", Nodes: make([]phasea.Node, 0, n), Edges: []phasea.Edge{}}
	for i := 0; i < n; i++ {
		id := "n" + strconv.Itoa(i)
		parent := ""
		if i > 0 {
			parent = "n" + strconv.Itoa(i-1)
		}
		d.Nodes = append(d.Nodes, node(id, parent, 0))
	}
	r, _ := analyzeAndEncode(t, d)
	if r.NodeCount != n || r.MaxDepth != n-1 || r.RootCount != 1 || r.LeafCount != 1 || r.WeakComponentCount != 1 || r.MaxBranchingFactor != 1 {
		t.Errorf("Analyze(chain %d) = %+v, want depth %d", n, r, n-1)
	}
}

func TestIndependentReferenceOnGeneratedValidForests(t *testing.T) {
	for seed := int64(0); seed < 120; seed++ {
		rng := rand.New(rand.NewSource(seed))
		d := generatedForest(rng, rng.Intn(140)+1)
		r, _ := analyzeAndEncode(t, d)
		if got, want := r.WeakComponentSizes, referenceComponentSizes(d); !slices.Equal(got, want) {
			t.Fatalf("Analyze(seed=%d) weak component sizes = %v, want %v", seed, got, want)
		}
		if got, want := r.DepthDistribution, referenceDepthHistogram(d); !reflect.DeepEqual(got, want) {
			t.Fatalf("Analyze(seed=%d) depth histogram = %v, want %v", seed, got, want)
		}
		if got, want := r.BranchingFactorDistribution, referenceBranchingHistogram(d); !reflect.DeepEqual(got, want) {
			t.Fatalf("Analyze(seed=%d) branching histogram = %v, want %v", seed, got, want)
		}
		var totalDepth, totalBranch, totalComp int
		for _, b := range r.DepthDistribution {
			totalDepth += b.Count
		}
		for _, b := range r.BranchingFactorDistribution {
			totalBranch += b.Count
		}
		for _, s := range r.WeakComponentSizes {
			totalComp += s
		}
		if totalDepth != len(d.Nodes) || totalBranch != len(d.Nodes) || totalComp != len(d.Nodes) || r.HierarchyEdgeCount != len(d.Nodes)-r.RootCount {
			t.Fatalf("Analyze(seed=%d) accounting invariant: %+v", seed, r)
		}
	}
}

// Reference connectivity uses BFS on an independently assembled undirected
// adjacency list rather than graphstats' disjoint-set implementation.
func referenceComponentSizes(d phasea.Document) []int {
	neighbors := make(map[string][]string, len(d.Nodes))
	for _, n := range d.Nodes {
		neighbors[n.ID] = []string{}
	}
	connect := func(a, b string) { neighbors[a] = append(neighbors[a], b); neighbors[b] = append(neighbors[b], a) }
	for _, n := range d.Nodes {
		if n.ParentID != "" {
			connect(n.ID, n.ParentID)
		}
	}
	for _, e := range d.Edges {
		connect(e.From, e.To)
	}
	seen := map[string]bool{}
	sizes := []int{}
	for id := range neighbors {
		if seen[id] {
			continue
		}
		queue := []string{id}
		seen[id] = true
		for i := 0; i < len(queue); i++ {
			for _, next := range neighbors[queue[i]] {
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
		sizes = append(sizes, len(queue))
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))
	return sizes
}

func referenceDepthHistogram(d phasea.Document) []Bucket {
	parents := map[string]string{}
	for _, n := range d.Nodes {
		parents[n.ID] = n.ParentID
	}
	hist := map[int]int{}
	for _, n := range d.Nodes {
		depth := 0
		for p := n.ParentID; p != ""; p = parents[p] {
			depth++
		}
		hist[depth]++
	}
	return buckets(hist)
}

func referenceBranchingHistogram(d phasea.Document) []Bucket {
	degree := map[string]int{}
	for _, n := range d.Nodes {
		degree[n.ID] = 0
	}
	for _, n := range d.Nodes {
		if n.ParentID != "" {
			degree[n.ParentID]++
		}
	}
	hist := map[int]int{}
	for _, d := range degree {
		hist[d]++
	}
	return buckets(hist)
}

func generatedForest(rng *rand.Rand, n int) phasea.Document {
	d := phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "untrusted", Nodes: make([]phasea.Node, 0, n), Edges: []phasea.Edge{}}
	nextOrder := map[string]int{}
	for i := 0; i < n; i++ {
		parent := ""
		if i > 0 && rng.Intn(6) != 0 {
			parent = "n" + strconv.Itoa(rng.Intn(i))
		}
		id := "n" + strconv.Itoa(i)
		d.Nodes = append(d.Nodes, node(id, parent, nextOrder[parent]))
		nextOrder[parent]++
	}
	seen := map[phasea.Edge]bool{}
	kinds := []string{"links_to", "depends_on", "related_to"}
	for i := 0; i < min(2*n, 180); i++ {
		e := phasea.Edge{From: "n" + strconv.Itoa(rng.Intn(n)), To: "n" + strconv.Itoa(rng.Intn(n)), Kind: kinds[rng.Intn(len(kinds))]}
		if !seen[e] {
			seen[e] = true
			d.Edges = append(d.Edges, e)
		}
	}
	return d
}

func FuzzAnalyzeValidForest(f *testing.F) {
	f.Add(uint64(0), uint16(7))
	f.Add(uint64(10), uint16(40))
	f.Add(uint64(999), uint16(1))
	f.Fuzz(func(t *testing.T, seed uint64, size uint16) {
		n := int(size%100) + 1
		d := generatedForest(rand.New(rand.NewSource(int64(seed))), n)
		r, err := Analyze(d)
		if err != nil {
			t.Fatalf("Analyze(generated valid seed=%d n=%d): %v", seed, n, err)
		}
		if r.NodeCount != n || r.HierarchyEdgeCount != n-r.RootCount || r.WeakComponentCount <= 0 {
			t.Fatalf("Analyze(generated valid seed=%d n=%d): %+v", seed, n, r)
		}
		b, err := Encode(r)
		if err != nil || !json.Valid(b) {
			t.Fatalf("Encode(generated seed=%d n=%d): %v", seed, n, err)
		}
	})
}

func TestEncodeOutputBudget(t *testing.T) {
	_, err := Encode(Report{Schema: strings.Repeat("x", MaxOutputBytes)})
	if err == nil {
		t.Error("Encode(oversized report) = nil error, want byte-budget rejection")
	}
}
