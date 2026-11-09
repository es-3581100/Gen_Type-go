// Package graphdiagnostics reports deterministic, read-only error information.
// Phase A's strict decoder remains the sole acceptance oracle.
package graphdiagnostics

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
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
		// Decoder error strings can be nondeterministic when multiple wire or
		// semantic errors coexist. Never copy their map-iteration order to output.
		r.Diagnostics = append(r.Diagnostics, stableDiagnostic(raw))
		return r
	}
	_, digest, err := phasea.EncodeCanonical(d)
	if err != nil {
		r.Diagnostics = append(r.Diagnostics, Diagnostic{Code: "INTERNAL_CANONICAL_FAILURE", Path: "$", Message: "canonical encoding failed", Action: "report the decoder/encoder contract mismatch"})
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
		return nil, errors.New("diagnostics exceed output byte budget")
	}
	return append(b, '\n'), nil
}
