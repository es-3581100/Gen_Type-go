// Package graphdiagnostics wraps Phase A's strict decoder with a read-only,
// stable first-error report. It never substitutes for graph acceptance.
package graphdiagnostics

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
	"regexp"
	"strconv"
	"strings"
)

const Schema = "gen-type-go/graph-diagnostics-v0"
const IdentityBoundary = "local_graph_structure_only;no_trusted_provenance_or_authority"
const MaxOutputBytes = 4 << 20

type Diagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Action  string `json:"action"`
}
type Report struct {
	Schema           string       `json:"schema"`
	Valid            bool         `json:"valid"`
	CanonicalSHA256  string       `json:"canonical_sha256,omitempty"`
	IdentityBoundary string       `json:"identity_boundary"`
	Diagnostics      []Diagnostic `json:"diagnostics"`
}

func Diagnose(raw []byte) Report {
	r := Report{Schema: Schema, IdentityBoundary: IdentityBoundary, Diagnostics: make([]Diagnostic, 0)}
	d, err := phasea.DecodeStrict(raw)
	if err != nil {
		r.Diagnostics = append(r.Diagnostics, classify(raw, err.Error()))
		return r
	}
	_, digest, err := phasea.EncodeCanonical(d)
	if err != nil {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{Code: "INTERNAL_CANONICAL_FAILURE", Path: "$", Message: "canonicalization failed", Action: "report an internal decoder/encoder mismatch"})
		return r
	}
	r.Valid = true
	r.CanonicalSHA256 = digest
	return r
}
func Encode(r Report) ([]byte, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("encode diagnostics: %w", err)
	}
	if len(b)+1 > MaxOutputBytes {
		return nil, errors.New("diagnostics output exceeds byte budget")
	}
	return append(b, '\n'), nil
}

var indexed = regexp.MustCompile(`^(node|edge) ([0-9]+)(?: field ([a-z_]+))?:`)
var quote = regexp.MustCompile(`"([^"]+)"`)

// Location hints are best effort; JSON re-decoding here is diagnostic-only
// and cannot accept a graph rejected by the strict decoder.
func path(raw []byte, msg string) string {
	if m := indexed.FindStringSubmatch(msg); m != nil {
		if _, err := strconv.Atoi(m[2]); err != nil {
			return "$"
		}
		prefix := "nodes"
		if m[1] == "edge" {
			prefix = "edges"
		}
		s := "$." + prefix + "[" + m[2] + "]"
		if m[3] != "" {
			s += "." + m[3]
		}
		if strings.Contains(msg, "missing field ") || strings.Contains(msg, "null field ") || strings.Contains(msg, "undeclared or case-aliased JSON field ") {
			q := quote.FindStringSubmatch(msg)
			if q != nil {
				s += "." + q[1]
			}
		}
		return s
	}
	var graph struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
		Edges []struct {
			From string `json:"from"`
			To   string `json:"to"`
			Kind string `json:"kind"`
		} `json:"edges"`
	}
	if json.Unmarshal(raw, &graph) == nil {
		q := quote.FindStringSubmatch(msg)
		if q != nil {
			for _, it := range []struct{ prefix, field string }{
				{"duplicate node ", "id"}, {"bad ordinal ", "order"},
				{"invalid node identifier ", "id"}, {"dangling parent of ", "parent_id"},
				{"invalid source provenance fields ", ""},
				{"node ", "parent_id"},
			} {
				if strings.HasPrefix(msg, it.prefix) {
					if it.prefix == "node " && !strings.HasSuffix(msg, " self-parent") {
						continue
					}
					ix, matches := -1, 0
					for i, n := range graph.Nodes {
						if n.ID == q[1] {
							ix = i
							matches++
							if it.prefix == "duplicate node " && matches == 2 {
								break
							}
						}
					}
					if ix >= 0 {
						if it.field == "" {
							return fmt.Sprintf("$.nodes[%d]", ix)
						}
						return fmt.Sprintf("$.nodes[%d].%s", ix, it.field)
					}
				}
			}
			for i, e := range graph.Edges {
				if strings.HasPrefix(msg, "edge from missing node ") && e.From == q[1] {
					return fmt.Sprintf("$.edges[%d].from", i)
				}
				if strings.HasPrefix(msg, "edge to missing node ") && e.To == q[1] {
					return fmt.Sprintf("$.edges[%d].to", i)
				}
				if strings.HasPrefix(msg, "unsupported edge kind ") && e.Kind == q[1] {
					return fmt.Sprintf("$.edges[%d].kind", i)
				}
			}
		}
	}
	if strings.HasPrefix(msg, "unexpected schema ") {
		return "$.schema"
	}
	if strings.HasPrefix(msg, "graph: undeclared or case-aliased JSON field ") {
		if q := quote.FindStringSubmatch(msg); q != nil {
			return "$." + q[1]
		}
	}
	if strings.HasPrefix(msg, "nodes must be array") {
		return "$.nodes"
	}
	if strings.HasPrefix(msg, "edges must be array") {
		return "$.edges"
	}
	if strings.HasPrefix(msg, "missing field ") || strings.HasPrefix(msg, "null field ") {
		q := quote.FindStringSubmatch(msg)
		if q != nil {
			return "$." + q[1]
		}
	}
	return "$"
}
func classify(raw []byte, msg string) Diagnostic {
	d := Diagnostic{Code: "INVALID_GRAPH", Path: path(raw, msg), Message: msg, Action: "correct the input and rerun diagnose"}
	switch {
	case strings.Contains(msg, "byte budget"):
		d.Code = "INPUT_TOO_LARGE"
		d.Action = "reduce input below 2 MiB"
	case strings.Contains(msg, "not UTF-8"), strings.Contains(msg, "surrogate"), strings.Contains(msg, "unicode escape"):
		d.Code = "INVALID_UTF8"
		d.Action = "provide valid UTF-8 and paired escapes"
	case strings.Contains(msg, "duplicate JSON key"):
		d.Code = "DUPLICATE_JSON_KEY"
		d.Action = "remove duplicate JSON keys"
	case strings.Contains(msg, "undeclared or case-aliased"):
		d.Code = "UNKNOWN_FIELD"
		d.Action = "use exact schema field names"
	case strings.Contains(msg, "missing field"), strings.Contains(msg, "must be explicit arrays"), strings.Contains(msg, "nodes and edges must be JSON arrays"):
		d.Code = "MISSING_FIELD"
		d.Action = "provide required fields and explicit arrays"
	case strings.Contains(msg, "null field"):
		d.Code = "NULL_FIELD"
		d.Action = "provide non-null field"
	case strings.Contains(msg, "must be array"), strings.Contains(msg, "must be integer"), strings.Contains(msg, "cannot unmarshal"), strings.Contains(msg, "decode graph:"):
		d.Code = "FIELD_TYPE"
		d.Action = "use required JSON types"
	case strings.Contains(msg, "unexpected schema"):
		d.Code = "BAD_SCHEMA"
		d.Action = "use gen-type-go/phase-a-graph-v0"
	case strings.Contains(msg, "duplicate node "):
		d.Code = "DUPLICATE_NODE"
		d.Action = "use distinct node IDs"
	case strings.Contains(msg, "invalid node identifier"), strings.Contains(msg, "invalid project/source identifier"):
		d.Code = "INVALID_ID"
		d.Action = "correct local identifiers"
	case strings.Contains(msg, "ordinal"):
		d.Code = "INVALID_ORDER"
		d.Action = "number siblings uniquely from zero with no gaps"
	case strings.Contains(msg, "dangling parent"), strings.Contains(msg, "self-parent"):
		d.Code = "INVALID_PARENT"
		d.Action = "reference an existing different parent"
	case strings.Contains(msg, "cycle"), strings.Contains(msg, "no root"), strings.Contains(msg, "disconnected"):
		d.Code = "INVALID_HIERARCHY"
		d.Action = "repair rooted acyclic hierarchy"
	case strings.Contains(msg, "invalid source provenance fields"):
		d.Code = "INVALID_SOURCE_CLAIM"
		d.Action = "provide source_ref and a 64-hex unverified digest claim"
	case strings.Contains(msg, "edge from missing node"), strings.Contains(msg, "edge to missing node"):
		d.Code = "INVALID_EDGE_ENDPOINT"
		d.Action = "reference existing node IDs"
	case strings.Contains(msg, "unsupported edge kind"):
		d.Code = "INVALID_EDGE_KIND"
		d.Action = "use links_to, related_to or depends_on"
	case strings.Contains(msg, "duplicate edge"):
		d.Code = "DUPLICATE_EDGE"
		d.Action = "remove repeated edge"
	case strings.Contains(msg, "JSON token:"), strings.Contains(msg, "trailing JSON"), strings.Contains(msg, "malformed"), strings.Contains(msg, "unexpected JSON"), strings.Contains(msg, "unterminated"), strings.Contains(msg, "invalid character"):
		d.Code = "MALFORMED_JSON"
		d.Action = "provide one complete JSON graph object"
	}
	return d
}
