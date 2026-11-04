// Package graphinspect provides deterministic read-only navigation of local graphs.
// Source content and digest claims are input data, not proof of authority.
package graphinspect

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

const Schema = "gen-type-go/graph-inspection-v0"
const IdentityBoundary = "local_graph_structure_only;source_claims_unverified;no_lineage_or_permission_inference"
const MaxOutputBytes = 4 << 20

var ErrNodeNotFound = errors.New("node not found")

type Report struct {
	Schema           string        `json:"schema"`
	CanonicalSHA256  string        `json:"canonical_sha256"`
	IdentityBoundary string        `json:"identity_boundary"`
	Node             phasea.Node   `json:"node"`
	Parent           *phasea.Node  `json:"parent"`
	Children         []string      `json:"children"`
	Ancestors        []string      `json:"ancestors"`
	Descendants      []string      `json:"descendants"`
	Roots            []string      `json:"roots"`
	Leaves           []string      `json:"leaves"`
	IncomingEdges    []phasea.Edge `json:"incoming_edges"`
	OutgoingEdges    []phasea.Edge `json:"outgoing_edges"`
}

func Inspect(doc phasea.Document, id string) (Report, error) {
	d, err := phasea.Canonicalize(doc)
	if err != nil {
		return Report{}, fmt.Errorf("invalid graph: %w", err)
	}
	_, digest, err := phasea.EncodeCanonical(d)
	if err != nil {
		return Report{}, err
	}
	byID := make(map[string]phasea.Node, len(d.Nodes))
	kids := make(map[string][]string, len(d.Nodes))
	roots := make([]string, 0)
	leaves := make([]string, 0)
	for _, node := range d.Nodes {
		byID[node.ID] = node
		kids[node.ParentID] = append(kids[node.ParentID], node.ID)
		if node.ParentID == "" {
			roots = append(roots, node.ID)
		}
	}
	node, ok := byID[id]
	if !ok {
		return Report{}, fmt.Errorf("%w: %q", ErrNodeNotFound, id)
	}
	for _, node := range d.Nodes {
		if len(kids[node.ID]) == 0 {
			leaves = append(leaves, node.ID)
		}
	}
	r := Report{
		Schema: Schema, CanonicalSHA256: digest, IdentityBoundary: IdentityBoundary, Node: node,
		Children: append([]string{}, kids[id]...), Ancestors: make([]string, 0), Descendants: make([]string, 0),
		Roots: roots, Leaves: leaves, IncomingEdges: make([]phasea.Edge, 0), OutgoingEdges: make([]phasea.Edge, 0),
	}
	if node.ParentID != "" {
		p := byID[node.ParentID]
		r.Parent = &p
	}
	for parent := node.ParentID; parent != ""; parent = byID[parent].ParentID {
		r.Ancestors = append(r.Ancestors, parent)
	}
	for i, j := 0, len(r.Ancestors)-1; i < j; i, j = i+1, j-1 {
		r.Ancestors[i], r.Ancestors[j] = r.Ancestors[j], r.Ancestors[i]
	}
	stack := make([]string, 0)
	for i := len(kids[id]) - 1; i >= 0; i-- {
		stack = append(stack, kids[id][i])
	}
	for len(stack) > 0 {
		k := len(stack) - 1
		current := stack[k]
		stack = stack[:k]
		r.Descendants = append(r.Descendants, current)
		for i := len(kids[current]) - 1; i >= 0; i-- {
			stack = append(stack, kids[current][i])
		}
	}
	for _, edge := range d.Edges {
		if edge.From == id {
			r.OutgoingEdges = append(r.OutgoingEdges, edge)
		}
		if edge.To == id {
			r.IncomingEdges = append(r.IncomingEdges, edge)
		}
	}
	return r, nil
}
func Encode(r Report) ([]byte, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode inspection: %w", err)
	}
	if len(b)+1 > MaxOutputBytes {
		return nil, errors.New("inspection output exceeds byte budget")
	}
	return append(b, '\n'), nil
}
