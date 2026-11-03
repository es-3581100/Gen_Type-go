package phasea

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var identityPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@/-]{0,127}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
var edgeKinds = map[string]bool{"links_to": true, "depends_on": true, "related_to": true}

// Validate enforces the structural representation contract, not a trust claim.
func Validate(d Document) error {
	if d.Schema != Schema {
		return fmt.Errorf("unexpected schema %q", d.Schema)
	}
	if !identityPattern.MatchString(d.ProjectID) || !identityPattern.MatchString(d.SourceID) {
		return errors.New("invalid project/source identifier")
	}
	if d.Nodes == nil || d.Edges == nil {
		return errors.New("nodes/edges must be explicit arrays")
	}
	if len(d.Nodes) > MaxNodes || len(d.Edges) > MaxEdges {
		return errors.New("graph exceeds record budget")
	}
	if len(d.Nodes) == 0 {
		return errors.New("graph needs a root node")
	}
	byID := make(map[string]Node, len(d.Nodes))
	children := make(map[string][]Node)
	for _, n := range d.Nodes {
		if !identityPattern.MatchString(n.ID) || n.ParentID != "" && !identityPattern.MatchString(n.ParentID) {
			return fmt.Errorf("invalid node identifier %q", n.ID)
		}
		if _, ok := byID[n.ID]; ok {
			return fmt.Errorf("duplicate node %q", n.ID)
		}
		if n.Order < 0 || n.Order > MaxNodes {
			return fmt.Errorf("bad ordinal %q", n.ID)
		}
		if len(n.Title) > 4096 || len(n.Content) > 262144 || len(n.SourceRef) > 4096 {
			return fmt.Errorf("node exceeds field budget %q", n.ID)
		}
		if !utf8.ValidString(n.Title + n.Content + n.SourceRef) {
			return fmt.Errorf("invalid text in node %q", n.ID)
		}
		if n.SourceRef == "" || !digestPattern.MatchString(n.SourceDigestClaim) {
			return fmt.Errorf("invalid source provenance fields %q", n.ID)
		}
		byID[n.ID] = n
		children[n.ParentID] = append(children[n.ParentID], n)
	}
	for _, n := range d.Nodes {
		if n.ParentID == n.ID {
			return fmt.Errorf("node %q self-parent", n.ID)
		}
		if n.ParentID != "" {
			if _, ok := byID[n.ParentID]; !ok {
				return fmt.Errorf("dangling parent of %q", n.ID)
			}
		}
	}
	if len(children[""]) == 0 {
		return errors.New("graph has no root")
	}
	for parent, ns := range children {
		seen := make(map[int]bool, len(ns))
		for _, n := range ns {
			if seen[n.Order] {
				return fmt.Errorf("reused sibling ordinal at parent %q", parent)
			}
			seen[n.Order] = true
		}
		for i := 0; i < len(ns); i++ {
			if !seen[i] {
				return fmt.Errorf("non-contiguous ordinal at parent %q", parent)
			}
		}
	}
	state := make(map[string]uint8)
	var walk func(string) error
	walk = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("cycle at %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, n := range children[id] {
			if err := walk(n.ID); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for _, root := range children[""] {
		if err := walk(root.ID); err != nil {
			return err
		}
	}
	if len(state) != len(byID) {
		return errors.New("disconnected or cyclic hierarchy")
	}
	seenEdges := make(map[string]bool, len(d.Edges))
	for _, e := range d.Edges {
		if !edgeKinds[e.Kind] {
			return fmt.Errorf("unsupported edge kind %q", e.Kind)
		}
		if _, ok := byID[e.From]; !ok {
			return fmt.Errorf("edge from missing node %q", e.From)
		}
		if _, ok := byID[e.To]; !ok {
			return fmt.Errorf("edge to missing node %q", e.To)
		}
		key := e.From + "\x00" + e.Kind + "\x00" + e.To
		if seenEdges[key] {
			return fmt.Errorf("duplicate edge %q", key)
		}
		seenEdges[key] = true
	}
	return nil
}

// Canonicalize orders sibling nodes in deterministic preorder, and links in
// lexicographic order. It does not infer or reconcile cross-import identities.
func Canonicalize(d Document) (Document, error) {
	if err := Validate(d); err != nil {
		return Document{}, err
	}
	out := Document{Schema: d.Schema, ProjectID: d.ProjectID, SourceID: d.SourceID, Nodes: make([]Node, 0, len(d.Nodes)), Edges: append([]Edge{}, d.Edges...)}
	kids := make(map[string][]Node)
	for _, n := range d.Nodes {
		kids[n.ParentID] = append(kids[n.ParentID], n)
	}
	for k := range kids {
		sort.Slice(kids[k], func(i, j int) bool { return kids[k][i].Order < kids[k][j].Order })
	}
	var emit func(string)
	emit = func(parent string) {
		for _, n := range kids[parent] {
			out.Nodes = append(out.Nodes, n)
			emit(n.ID)
		}
	}
	emit("")
	sort.Slice(out.Edges, func(i, j int) bool {
		a, b := out.Edges[i], out.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.To < b.To
	})
	return out, nil
}

// EncodeCanonical returns exactly one deterministic JSON encoding with LF.
func EncodeCanonical(d Document) ([]byte, string, error) {
	canon, err := Canonicalize(d)
	if err != nil {
		return nil, "", err
	}
	b, err := json.Marshal(canon)
	if err != nil {
		return nil, "", err
	}
	b = append(b, '\n')
	digest := sha256.Sum256(b)
	return b, hex.EncodeToString(digest[:]), nil
}

func ReconstructCanonical(raw []byte) (Document, error) {
	d, err := DecodeStrict(raw)
	if err != nil {
		return Document{}, err
	}
	b, _, err := EncodeCanonical(d)
	if err != nil {
		return Document{}, err
	}
	if !bytes.Equal(raw, b) {
		return Document{}, errors.New("wire representation not canonical")
	}
	return d, nil
}

func EqualGraphs(a, b Document) (bool, error) {
	ab, _, err := EncodeCanonical(a)
	if err != nil {
		return false, err
	}
	bb, _, err := EncodeCanonical(b)
	if err != nil {
		return false, err
	}
	return bytes.Equal(ab, bb), nil
}

// AssessCrossImport never grants identity continuity or admission. A digest is
// an integrity marker, not an external principal or source attestation.
func AssessCrossImport(before, after Document) (ChangeAssessment, error) {
	_, beforeHash, err := EncodeCanonical(before)
	if err != nil {
		return ChangeAssessment{}, err
	}
	_, afterHash, err := EncodeCanonical(after)
	if err != nil {
		return ChangeAssessment{}, err
	}
	return ChangeAssessment{
		BeforeSHA256: beforeHash, AfterSHA256: afterHash,
		Identity: "unresolved_cross_import_identity", NewlyVisible: []string{},
		ReviewCompleteness: "not_exhaustive", Admission: "blocked",
		Reason: "no independently protected identity mapping, principal issuer, revision approval, or policy publication",
	}, nil
}

// Ordinal reordering of a sibling is representation-preserving only when
// provided as a complete, valid new graph. This function never mutates input.
func SupportsOnlyLocalIdentity(d Document) bool {
	for _, n := range d.Nodes {
		if strings.TrimSpace(n.ID) == "" {
			return false
		}
	}
	return true
}
