// Package graphdiff compares two validated local graph snapshots. It never
// establishes node identity continuity, permissions, or structural admission.
package graphdiff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

const Schema = "gen-type-go/graph-diff-v0"
const MaxOutputBytes = 16 << 20
const IdentityBoundary = "node_id_equality_is_syntactic_only;trusted_cross_import_lineage_unverified"

// MetadataChange describes a changed document-level string field.
type MetadataChange struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// ModifiedNode identifies changed content or source-claim field names, not
// their contents. Neither a source reference nor digest claim is verified.
type ModifiedNode struct {
	ID     string   `json:"id"`
	Fields []string `json:"fields"`
}

// ReparentedNode compares syntactically matching IDs, without asserting that
// the IDs refer to the same real-world object across imports.
type ReparentedNode struct {
	ID             string `json:"id"`
	BeforeParentID string `json:"before_parent_id"`
	AfterParentID  string `json:"after_parent_id"`
}

// ChildOrderChange captures the complete ordered child-ID list on each side.
// The empty parent ID represents the ordered root list.
type ChildOrderChange struct {
	ParentID string   `json:"parent_id"`
	Before   []string `json:"before"`
	After    []string `json:"after"`
}

// Report is a bounded, deterministic, non-authoritative local data diff.
// Every slice encodes as an array, including when no changes were found.
type Report struct {
	Schema            string             `json:"schema"`
	BeforeSHA256      string             `json:"before_sha256"`
	AfterSHA256       string             `json:"after_sha256"`
	Identical         bool               `json:"identical"`
	IdentityBoundary  string             `json:"identity_boundary"`
	MetadataChanges   []MetadataChange   `json:"metadata_changes"`
	AddedNodes        []string           `json:"added_nodes"`
	RemovedNodes      []string           `json:"removed_nodes"`
	ModifiedNodes     []ModifiedNode     `json:"modified_nodes"`
	ReparentedNodes   []ReparentedNode   `json:"reparented_nodes"`
	ChildOrderChanges []ChildOrderChange `json:"child_order_changes"`
	AddedEdges        []phasea.Edge      `json:"added_edges"`
	RemovedEdges      []phasea.Edge      `json:"removed_edges"`
}

// Compare validates and canonicalizes both graphs before computing any change.
// It never mutates either input or interprets matching IDs as trusted lineage.
func Compare(before, after phasea.Document) (Report, error) {
	beforeBytes, beforeHash, err := phasea.EncodeCanonical(before)
	if err != nil {
		return Report{}, fmt.Errorf("before graph: %w", err)
	}
	afterBytes, afterHash, err := phasea.EncodeCanonical(after)
	if err != nil {
		return Report{}, fmt.Errorf("after graph: %w", err)
	}
	// Decode only our own already-validated canonical encoder output.
	// This avoids a second full graph validation and canonicalization pass.
	var beforeCanon, afterCanon phasea.Document
	if err := json.Unmarshal(beforeBytes, &beforeCanon); err != nil {
		return Report{}, fmt.Errorf("decode before canonical bytes: %w", err)
	}
	if err := json.Unmarshal(afterBytes, &afterCanon); err != nil {
		return Report{}, fmt.Errorf("decode after canonical bytes: %w", err)
	}

	r := Report{
		Schema: Schema, BeforeSHA256: beforeHash, AfterSHA256: afterHash,
		Identical: bytes.Equal(beforeBytes, afterBytes), IdentityBoundary: IdentityBoundary,
		MetadataChanges: []MetadataChange{}, AddedNodes: []string{}, RemovedNodes: []string{},
		ModifiedNodes: []ModifiedNode{}, ReparentedNodes: []ReparentedNode{},
		ChildOrderChanges: []ChildOrderChange{}, AddedEdges: []phasea.Edge{}, RemovedEdges: []phasea.Edge{},
	}
	if beforeCanon.ProjectID != afterCanon.ProjectID {
		r.MetadataChanges = append(r.MetadataChanges, MetadataChange{Field: "project_id", Before: beforeCanon.ProjectID, After: afterCanon.ProjectID})
	}
	if beforeCanon.SourceID != afterCanon.SourceID {
		r.MetadataChanges = append(r.MetadataChanges, MetadataChange{Field: "source_id", Before: beforeCanon.SourceID, After: afterCanon.SourceID})
	}

	oldNodes := make(map[string]phasea.Node, len(beforeCanon.Nodes))
	newNodes := make(map[string]phasea.Node, len(afterCanon.Nodes))
	for _, n := range beforeCanon.Nodes {
		oldNodes[n.ID] = n
	}
	for _, n := range afterCanon.Nodes {
		newNodes[n.ID] = n
	}
	for id, old := range oldNodes {
		current, ok := newNodes[id]
		if !ok {
			r.RemovedNodes = append(r.RemovedNodes, id)
			continue
		}
		var fields []string
		if old.Title != current.Title {
			fields = append(fields, "title")
		}
		if old.Content != current.Content {
			fields = append(fields, "content")
		}
		if old.SourceRef != current.SourceRef {
			fields = append(fields, "source_ref")
		}
		if old.SourceDigestClaim != current.SourceDigestClaim {
			fields = append(fields, "source_digest_claim")
		}
		if len(fields) != 0 {
			r.ModifiedNodes = append(r.ModifiedNodes, ModifiedNode{ID: id, Fields: fields})
		}
		if old.ParentID != current.ParentID {
			r.ReparentedNodes = append(r.ReparentedNodes, ReparentedNode{ID: id, BeforeParentID: old.ParentID, AfterParentID: current.ParentID})
		}
	}
	for id := range newNodes {
		if _, ok := oldNodes[id]; !ok {
			r.AddedNodes = append(r.AddedNodes, id)
		}
	}
	sort.Strings(r.AddedNodes)
	sort.Strings(r.RemovedNodes)
	sort.Slice(r.ModifiedNodes, func(i, j int) bool { return r.ModifiedNodes[i].ID < r.ModifiedNodes[j].ID })
	sort.Slice(r.ReparentedNodes, func(i, j int) bool { return r.ReparentedNodes[i].ID < r.ReparentedNodes[j].ID })

	oldChildren := childLists(beforeCanon.Nodes)
	newChildren := childLists(afterCanon.Nodes)
	parents := make(map[string]bool, len(oldChildren)+len(newChildren))
	for parent := range oldChildren {
		parents[parent] = true
	}
	for parent := range newChildren {
		parents[parent] = true
	}
	for parent := range parents {
		oldList, newList := oldChildren[parent], newChildren[parent]
		if !slices.Equal(oldList, newList) {
			if oldList == nil {
				oldList = []string{}
			}
			if newList == nil {
				newList = []string{}
			}
			r.ChildOrderChanges = append(r.ChildOrderChanges, ChildOrderChange{ParentID: parent, Before: oldList, After: newList})
		}
	}
	sort.Slice(r.ChildOrderChanges, func(i, j int) bool { return r.ChildOrderChanges[i].ParentID < r.ChildOrderChanges[j].ParentID })

	oldEdges := make(map[phasea.Edge]bool, len(beforeCanon.Edges))
	newEdges := make(map[phasea.Edge]bool, len(afterCanon.Edges))
	for _, e := range beforeCanon.Edges {
		oldEdges[e] = true
	}
	for _, e := range afterCanon.Edges {
		newEdges[e] = true
	}
	for e := range oldEdges {
		if !newEdges[e] {
			r.RemovedEdges = append(r.RemovedEdges, e)
		}
	}
	for e := range newEdges {
		if !oldEdges[e] {
			r.AddedEdges = append(r.AddedEdges, e)
		}
	}
	sortEdges(r.AddedEdges)
	sortEdges(r.RemovedEdges)
	return r, nil
}

// Encode emits compact JSON followed by one LF. It rejects outputs exceeding
// the fixed output budget before emitting any bytes to a caller-owned writer.
func Encode(r Report) ([]byte, error) {
	encoded, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode graph diff: %w", err)
	}
	if len(encoded)+1 > MaxOutputBytes {
		return nil, errors.New("graph diff exceeds output byte budget")
	}
	return append(encoded, '\n'), nil
}

func childLists(nodes []phasea.Node) map[string][]string {
	// The supplied documents have already been canonicalized, so the
	// preorder node order preserves the sibling-order contract.
	children := make(map[string][]string)
	for _, n := range nodes {
		children[n.ParentID] = append(children[n.ParentID], n.ID)
	}
	return children
}

func sortEdges(edges []phasea.Edge) {
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.To < b.To
	})
}
