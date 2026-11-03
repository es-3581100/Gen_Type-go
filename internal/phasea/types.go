// Package phasea contains an isolated, in-memory proof of canonical graph
// representation and pure policy evaluation. It has no trusted identity issuer,
// storage, server, or revision-admission authority. It is NOT production ready.
package phasea

type Node struct {
	ID        string `json:"id"`
	ParentID  string `json:"parent_id"`
	Order     int    `json:"order"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	SourceRef string `json:"source_ref"`
	// SourceDigestClaim is supplied by an untrusted importer; never an attestation.
	SourceDigestClaim string `json:"source_digest_claim"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

type Document struct {
	Schema    string `json:"schema"`
	ProjectID string `json:"project_id"`
	SourceID  string `json:"source_id"`
	Nodes     []Node `json:"nodes"`
	Edges     []Edge `json:"edges"`
}

const Schema = "gen-type-go/phase-a-graph-v0"

const MaxInputBytes = 2 << 20
const MaxNodes = 10_000
const MaxEdges = 30_000

// Principal is a pure evaluator input ONLY. Its kind and ID MUST be supplied
// by a separate, independently authenticated issuer before any future use.
// Do not construct one from request JSON, headers, paths, or imported data.
type Principal struct {
	Kind string
	ID   string
}

type Rule struct {
	PrincipalKind string
	PrincipalID   string
	NodeID        string
	Action        string
	Effect        string
	Inherit       bool
}

type Policy struct {
	ProjectID   string
	Epoch       uint64
	HumanRootID string
	Rules       []Rule
}

type Decision struct {
	Allowed bool   `json:"allowed"`
	Reason  string `json:"reason"`
}

type ChangeAssessment struct {
	BeforeSHA256       string   `json:"before_sha256"`
	AfterSHA256        string   `json:"after_sha256"`
	Identity           string   `json:"identity"`
	NewlyVisible       []string `json:"newly_visible"`
	ReviewCompleteness string   `json:"review_completeness"`
	Admission          string   `json:"admission"`
	Reason             string   `json:"reason"`
}
