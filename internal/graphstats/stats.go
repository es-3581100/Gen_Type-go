// Package graphstats computes deterministic, read-only measurements of a
// validated local graph. It does not establish trusted identity or authority.
package graphstats

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

const Schema = "gen-type-go/graph-stats-v0"
const IdentityBoundary = "local_graph_structure_only;source_claims_unverified;no_lineage_or_permission_inference"
const MaxOutputBytes = 4 << 20

// Bucket is a count of nodes with a particular depth or number of children.
// Buckets are sorted by Value in ascending order.
type Bucket struct {
	Value int `json:"value"`
	Count int `json:"count"`
}

// KindCount counts an explicit edge kind recorded by the input graph.
type KindCount struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// Ratio preserves the exact average branching factor without floating-point
// formatting or rounding. Denominator is always the nonzero node count.
type Ratio struct {
	Numerator   int `json:"numerator"`
	Denominator int `json:"denominator"`
}

// Report contains deterministic graph measurements, not verified provenance.
// Explicit edges are the typed edges in Document.Edges. Hierarchy edges are
// parent-child relationships, counted separately even when an explicit edge
// happens to connect the same two nodes. All slices are non-nil JSON arrays.
type Report struct {
	Schema                      string      `json:"schema"`
	CanonicalSHA256             string      `json:"canonical_sha256"`
	IdentityBoundary            string      `json:"identity_boundary"`
	NodeCount                   int         `json:"node_count"`
	ExplicitEdgeCount           int         `json:"explicit_edge_count"`
	HierarchyEdgeCount          int         `json:"hierarchy_edge_count"`
	RootCount                   int         `json:"root_count"`
	LeafCount                   int         `json:"leaf_count"`
	MaxDepth                    int         `json:"max_depth"`
	DepthDistribution           []Bucket    `json:"depth_distribution"`
	MaxBranchingFactor          int         `json:"max_branching_factor"`
	BranchingFactorDistribution []Bucket    `json:"branching_factor_distribution"`
	MeanBranchingFactor         Ratio       `json:"mean_branching_factor"`
	WeakComponentCount          int         `json:"weak_component_count"`
	WeakComponentSizes          []int       `json:"weak_component_sizes"`
	ExplicitEdgeKinds           []KindCount `json:"explicit_edge_kinds"`
	IsolatedNodeCount           int         `json:"isolated_node_count"`
	ExplicitSelfLoopCount       int         `json:"explicit_self_loop_count"`
}

// Analyze validates the supplied graph using the existing Phase A contract,
// computes its canonical digest, then measures only the supplied structure.
// It does not mutate the input, infer permissions, or infer cross-import links.
func Analyze(d phasea.Document) (Report, error) {
	_, digest, err := phasea.EncodeCanonical(d)
	if err != nil {
		return Report{}, fmt.Errorf("invalid graph: %w", err)
	}

	n := len(d.Nodes) // Validate guarantees n > 0 and bounded record counts.
	index := make(map[string]int, n)
	for i, node := range d.Nodes {
		index[node.ID] = i
	}

	children := make([][]int, n)
	roots := make([]int, 0)
	connected := newDisjointSet(n)
	for i, node := range d.Nodes {
		if node.ParentID == "" {
			roots = append(roots, i)
			continue
		}
		parent := index[node.ParentID]
		children[parent] = append(children[parent], i)
		connected.union(i, parent)
	}

	// Iterative traversal handles the full legal depth without recursion in
	// the statistics computation, including long single-child chains.
	depths := make([]int, n)
	queue := append([]int(nil), roots...)
	depthCounts := make(map[int]int)
	maxDepth := 0
	for cursor := 0; cursor < len(queue); cursor++ {
		id := queue[cursor]
		depth := depths[id]
		depthCounts[depth]++
		if depth > maxDepth {
			maxDepth = depth
		}
		for _, child := range children[id] {
			depths[child] = depth + 1
			queue = append(queue, child)
		}
	}

	branchCounts := make(map[int]int)
	leaves := 0
	maxBranching := 0
	for _, childIDs := range children {
		degree := len(childIDs)
		branchCounts[degree]++
		if degree == 0 {
			leaves++
		}
		if degree > maxBranching {
			maxBranching = degree
		}
	}

	incident := make([]bool, n)
	kinds := make(map[string]int)
	selfLoops := 0
	for _, edge := range d.Edges {
		from, to := index[edge.From], index[edge.To]
		connected.union(from, to) // Ignore direction for weak connectivity.
		incident[from], incident[to] = true, true
		kinds[edge.Kind]++
		if from == to {
			selfLoops++
		}
	}
	isolated := 0
	for i, node := range d.Nodes {
		if node.ParentID == "" && len(children[i]) == 0 && !incident[i] {
			isolated++
		}
	}

	componentCounts := make(map[int]int)
	for i := range d.Nodes {
		componentCounts[connected.find(i)]++
	}
	sizes := make([]int, 0, len(componentCounts))
	for _, size := range componentCounts {
		sizes = append(sizes, size)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))
	edgeKinds := make([]KindCount, 0, len(kinds))
	for kind, count := range kinds {
		edgeKinds = append(edgeKinds, KindCount{Kind: kind, Count: count})
	}
	sort.Slice(edgeKinds, func(i, j int) bool { return edgeKinds[i].Kind < edgeKinds[j].Kind })

	return Report{
		Schema:                      Schema,
		CanonicalSHA256:             digest,
		IdentityBoundary:            IdentityBoundary,
		NodeCount:                   n,
		ExplicitEdgeCount:           len(d.Edges),
		HierarchyEdgeCount:          n - len(roots),
		RootCount:                   len(roots),
		LeafCount:                   leaves,
		MaxDepth:                    maxDepth,
		DepthDistribution:           buckets(depthCounts),
		MaxBranchingFactor:          maxBranching,
		BranchingFactorDistribution: buckets(branchCounts),
		MeanBranchingFactor:         Ratio{Numerator: n - len(roots), Denominator: n},
		WeakComponentCount:          len(sizes),
		WeakComponentSizes:          sizes,
		ExplicitEdgeKinds:           edgeKinds,
		IsolatedNodeCount:           isolated,
		ExplicitSelfLoopCount:       selfLoops,
	}, nil
}

// Encode produces a bounded JSON object and one terminating LF. No partial
// JSON can escape before all encoding and size checks have succeeded.
func Encode(r Report) ([]byte, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode graph statistics: %w", err)
	}
	if len(b)+1 > MaxOutputBytes {
		return nil, errors.New("graph statistics exceed output byte budget")
	}
	return append(b, '\n'), nil
}

func buckets(counts map[int]int) []Bucket {
	result := make([]Bucket, 0, len(counts))
	for value, count := range counts {
		result = append(result, Bucket{Value: value, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Value < result[j].Value })
	return result
}

type disjointSet struct {
	parent []int
	size   []int
}

func newDisjointSet(n int) disjointSet {
	s := disjointSet{parent: make([]int, n), size: make([]int, n)}
	for i := range s.parent {
		s.parent[i], s.size[i] = i, 1
	}
	return s
}

func (s disjointSet) find(i int) int {
	for s.parent[i] != i {
		s.parent[i] = s.parent[s.parent[i]]
		i = s.parent[i]
	}
	return i
}

func (s disjointSet) union(a, b int) {
	a, b = s.find(a), s.find(b)
	if a == b {
		return
	}
	if s.size[a] < s.size[b] {
		a, b = b, a
	}
	s.parent[b] = a
	s.size[a] += s.size[b]
}
