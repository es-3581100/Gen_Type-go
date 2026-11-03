package phasea

import (
	"fmt"
	"strings"
	"testing"
)

func expectBlockedProbeError(t *testing.T, result ChangeAssessment, err error, errorContains string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), errorContains) {
		t.Fatalf("error = %v; wanted containing %q", err, errorContains)
	}
	if result.Admission != "blocked" || result.Reason == "" || result.Identity != "unresolved_cross_import_identity" || result.ReviewCompleteness == "" || result.NewlyVisible == nil || len(result.NewlyVisible) != 0 {
		t.Fatalf("incomplete fail-closed assessment: %+v (error %v)", result, err)
	}
}

// R003-I01: corrupted graph inputs previously returned a zero assessment
// from the nested canonicalizer, even though they returned an error.
func TestRepair004InvalidBeforeOrAfterAlwaysReturnsBlockedDisposition(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(3)
	cases := []struct {
		label  string
		mutate func(*Document, *Document)
	}{
		{"nil_before_nodes", func(b, a *Document) { b.Nodes = nil }},
		{"nil_after_nodes", func(b, a *Document) { a.Nodes = nil }},
		{"nil_before_edges", func(b, a *Document) { b.Edges = nil }},
		{"nil_after_edges", func(b, a *Document) { a.Edges = nil }},
		{"bad_schema", func(b, a *Document) { b.Schema = "not-graph" }},
		{"invalid_parent", func(b, a *Document) { a.Nodes[1].ParentID = "missing" }},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			b, a := before, after
			b.Nodes = append([]Node(nil), before.Nodes...)
			a.Nodes = append([]Node(nil), after.Nodes...)
			tc.mutate(&b, &a)
			result, err := ProbeVisibilityGains(b, a, Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}, nil)
			expectBlockedProbeError(t, result, err, "")
			if result.BeforeSHA256 != "" || result.AfterSHA256 != "" {
				t.Fatalf("invalid graph emitted digest claims: %+v", result)
			}
		})
	}
}

// R003-I02: reject cheap malformed/oversized samples before computing
// any canonical graph digests; deliberately use invalid graph to test order
// without machine-dependent timing thresholds.
func TestRepair004CheapBudgetsPrecedeGraphCanonicalization(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(3)
	before.Nodes = nil
	policy := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
	huge := make([]Principal, MaxProbePrincipalInputs+1)
	for i := range huge {
		huge[i] = Principal{Kind: "agent", ID: "same"}
	}
	result, err := ProbeVisibilityGains(before, after, policy, huge)
	expectBlockedProbeError(t, result, err, "sampled principal input budget exceeded")

	// The rule-count cap must also reject prior to invalid graph checks.
	policy.Rules = make([]Rule, MaxProbePolicyRules+1)
	result, err = ProbeVisibilityGains(before, after, policy, nil)
	expectBlockedProbeError(t, result, err, "sampled policy rule budget exceeded")

	policy.Rules = []Rule{}
	distinct := make([]Principal, MaxProbeUniquePrincipals+1)
	for i := range distinct {
		distinct[i] = Principal{Kind: "agent", ID: fmt.Sprintf("user-%d", i)}
	}
	result, err = ProbeVisibilityGains(before, after, policy, distinct)
	expectBlockedProbeError(t, result, err, "unique sampled principal budget exceeded")
}

func TestRepair004BoundariesAndValidAdmissionUnchanged(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(4)
	policy := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
	exactly4096 := make([]Principal, MaxProbePrincipalInputs)
	for i := range exactly4096 {
		exactly4096[i] = Principal{Kind: "agent", ID: "same"}
	}
	out, err := ProbeVisibilityGains(before, after, policy, exactly4096)
	if err != nil || out.Admission != "blocked" || out.ReviewCompleteness != "sampled_principals_only_not_exhaustive" {
		t.Fatalf("valid request changed: %v %+v", err, out)
	}
	// Invalid principal entries, including duplicate-position entries, stay invalid.
	exactly4096[len(exactly4096)-1] = Principal{Kind: "agent", ID: ".."}
	out, err = ProbeVisibilityGains(before, after, policy, exactly4096)
	expectBlockedProbeError(t, out, err, "invalid principal id")
}

func TestRepair004ErrorPathsStillFailClosedAfterGraphValidation(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(3)
	cases := []struct {
		label  string
		policy Policy
		people []Principal
		mutate func(*Document, *Document)
	}{
		{label: "invalid_policy_project", policy: Policy{ProjectID: "other", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}},
		{label: "missing_policy_rules", policy: Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner"}},
		{label: "malformed_rule", policy: Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "root", Action: "unknown", Effect: "grant"}}}},
		{label: "different_project", policy: Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}, mutate: func(b, a *Document) { a.ProjectID = "other" }},
		{label: "invalid_principal", policy: Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}, people: []Principal{{Kind: "agent", ID: "."}}},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			b, a := before, after
			if tc.mutate != nil {
				tc.mutate(&b, &a)
			}
			report, err := ProbeVisibilityGains(b, a, tc.policy, tc.people)
			expectBlockedProbeError(t, report, err, "")
		})
	}
}
