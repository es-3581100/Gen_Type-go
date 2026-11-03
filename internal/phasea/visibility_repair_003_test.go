package phasea

import (
	"fmt"
	"reflect"
	"testing"
)

// The independent Repair 002 audit used 2,000 duplicate sampled actors.
// Repeating the same authenticated-identity *value* in this pure probe should
// not change the reported gain set; this is still not identity authentication.
func TestRepair003DuplicatePrincipalsShareOneProjection(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(200)
	p := Policy{ProjectID: before.ProjectID, Epoch: 1, HumanRootID: "owner", Rules: []Rule{
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "root", Action: "read", Effect: "grant", Inherit: true},
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "restricted", Action: "read", Effect: "deny", Inherit: true},
	}}
	actor := Principal{Kind: "agent", ID: "worker"}
	single, err := ProbeVisibilityGains(before, after, p, []Principal{actor})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{40, 400, 2000} {
		duplicate := make([]Principal, n)
		for i := range duplicate {
			duplicate[i] = actor
		}
		result, err := ProbeVisibilityGains(before, after, p, duplicate)
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if !reflect.DeepEqual(result, single) {
			t.Fatalf("duplicate drift n=%d: %+v != %+v", n, result, single)
		}
		if len(result.NewlyVisible) != 200 || result.Admission != "blocked" {
			t.Fatal("sample was accidentally authoritative")
		}
	}
}

func repair003FixtureMovedLeaves(count int) (Document, Document) {
	nodes := make([]Node, 0, count+2)
	mk := func(id, parent string, order int) Node {
		return Node{ID: id, ParentID: parent, Order: order, Title: id, Content: "data", SourceRef: "source.md", SourceDigestClaim: shaClaim}
	}
	nodes = append(nodes, mk("root", "", 0), mk("restricted", "root", 0))
	for i := 0; i < count; i++ {
		nodes = append(nodes, mk(fmt.Sprintf("leaf-%04d", i), "restricted", i))
	}
	before := Document{Schema: Schema, ProjectID: "proj", SourceID: "source", Nodes: nodes, Edges: []Edge{}}
	after := before
	after.Nodes = append([]Node(nil), nodes...)
	for i := 2; i < len(after.Nodes); i++ {
		after.Nodes[i].ParentID = "root"
		after.Nodes[i].Order = i - 1
	}
	return before, after
}

func TestRepair003NamespacedPrincipalAndUnrelatedRules(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(10)
	human := Principal{Kind: "human", ID: "worker"}
	agent := Principal{Kind: "agent", ID: "worker"}
	p := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{
		{PrincipalKind: "human", PrincipalID: "worker", NodeID: "root", Action: "read", Effect: "grant", Inherit: true},
		{PrincipalKind: "human", PrincipalID: "worker", NodeID: "restricted", Action: "read", Effect: "deny", Inherit: true},
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "root", Action: "write", Effect: "grant", Inherit: true},
	}}
	// Agent with only write authority cannot inherit the human read grant.
	got, err := ProbeVisibilityGains(before, after, p, []Principal{agent})
	if err != nil || len(got.NewlyVisible) != 0 || got.Admission != "blocked" {
		t.Fatalf("agent %v %+v", err, got)
	}
	got, err = ProbeVisibilityGains(before, after, p, []Principal{human, agent, human})
	if err != nil || len(got.NewlyVisible) != 10 || got.Admission != "blocked" {
		t.Fatalf("mixed %v %+v", err, got)
	}
	original := got
	for i := 0; i < 4000; i++ {
		p.Rules = append(p.Rules, Rule{PrincipalKind: "human", PrincipalID: fmt.Sprintf("other-%d", i/80), NodeID: fmt.Sprintf("leaf-%04d", i%10), Action: []string{"read", "write"}[i%2], Effect: []string{"grant", "deny"}[(i/2)%2], Inherit: (i/4)%2 == 1})
	}
	// Duplicate rule tuples are forbidden by ValidatePolicy. Ensure the
	// unrelated corpus is unique without relaxing that validator.
	for i := 3; i < len(p.Rules); i++ {
		p.Rules[i].PrincipalID = fmt.Sprintf("other-%d", i)
	}
	got, err = ProbeVisibilityGains(before, after, p, []Principal{human, agent, human})
	if err != nil || !reflect.DeepEqual(got, original) {
		t.Fatalf("unrelated rules changed outcomes: %v %+v != %+v", err, got, original)
	}
}

func TestRepair003ExplicitBudgetsFailClosed(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(4)
	p := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
	duplicate := make([]Principal, MaxProbePrincipalInputs+1)
	for i := range duplicate {
		duplicate[i] = Principal{Kind: "agent", ID: "worker"}
	}
	report, err := ProbeVisibilityGains(before, after, p, duplicate)
	if err == nil || report.Admission != "blocked" {
		t.Fatalf("raw principal cap: %v %+v", err, report)
	}
	unique := make([]Principal, MaxProbeUniquePrincipals+1)
	for i := range unique {
		unique[i] = Principal{Kind: "agent", ID: fmt.Sprintf("p-%d", i)}
	}
	report, err = ProbeVisibilityGains(before, after, p, unique)
	if err == nil || report.Admission != "blocked" {
		t.Fatalf("unique principal cap: %v %+v", err, report)
	}
	p.Rules = make([]Rule, MaxProbePolicyRules+1)
	report, err = ProbeVisibilityGains(before, after, p, nil)
	if err == nil || report.Admission != "blocked" {
		t.Fatalf("rule cap: %v %+v", err, report)
	}
}

func TestRepair003RejectsInvalidUnrelatedRulesAndPrincipals(t *testing.T) {
	before, after := repair003FixtureMovedLeaves(4)
	p := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{{PrincipalKind: "agent", PrincipalID: "bystander", NodeID: "root", Action: "DELETE", Effect: "grant"}}}
	if _, err := ProbeVisibilityGains(before, after, p, []Principal{{Kind: "agent", ID: "worker"}}); err == nil {
		t.Fatal("malformed unrelated rule accepted")
	}
	p.Rules = []Rule{}
	if _, err := ProbeVisibilityGains(before, after, p, []Principal{{Kind: "agent", ID: "worker"}, {Kind: "agent", ID: "../evil"}}); err == nil {
		t.Fatal("invalid trailing principal accepted")
	}
}

func BenchmarkRepair003DeduplicatedPrincipals(b *testing.B) {
	before, after := repair003FixtureMovedLeaves(200)
	p := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
	for _, size := range []int{1, 40, 400, 2000} {
		b.Run(fmt.Sprintf("duplicates_%d", size), func(b *testing.B) {
			principals := make([]Principal, size)
			for i := range principals {
				principals[i] = Principal{Kind: "agent", ID: "worker"}
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result, err := ProbeVisibilityGains(before, after, p, principals)
				if err != nil || result.Admission != "blocked" {
					b.Fatalf("%v %+v", err, result)
				}
			}
		})
	}
}
