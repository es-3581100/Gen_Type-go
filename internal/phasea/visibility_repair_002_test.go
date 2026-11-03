package phasea

import (
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"testing"
)

// Independently compare the optimized probe against the unmodified,
// validated Authorize contract. This tests parity, not permission to admit.
func TestRepair002ProjectionParityAgainstAuthorize(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		rng := rand.New(rand.NewSource(seed))
		before := randomTreeForProbe(rng, 24)
		after := randomTreeForProbe(rng, 24)
		policy := Policy{ProjectID: before.ProjectID, Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
		principals := []Principal{{Kind: "agent", ID: "worker"}, {Kind: "human", ID: "worker"}, {Kind: "agent", ID: "other"}}
		seen := map[Rule]bool{}
		for j := 0; j < 90; j++ {
			p := principals[rng.Intn(len(principals))]
			r := Rule{
				PrincipalKind: p.Kind, PrincipalID: p.ID,
				NodeID: fmt.Sprintf("node-%02d", rng.Intn(24)),
				Action: []string{"read", "write"}[rng.Intn(2)],
				Effect: []string{"grant", "deny"}[rng.Intn(2)], Inherit: rng.Intn(2) == 1,
			}
			if !seen[r] {
				policy.Rules = append(policy.Rules, r)
				seen[r] = true
			}
		}
		if err := ValidatePolicy(before, policy); err != nil {
			t.Fatalf("seed=%d before: %v", seed, err)
		}
		if err := ValidatePolicy(after, policy); err != nil {
			t.Fatalf("seed=%d after: %v", seed, err)
		}
		expected := make([]string, 0)
		for _, n := range after.Nodes {
			gain := false
			for _, p := range principals {
				old := Authorize(before, policy, p, n.ID, "read")
				newD := Authorize(after, policy, p, n.ID, "read")
				if !old.Allowed && newD.Allowed {
					gain = true
				}
			}
			if gain {
				expected = append(expected, n.ID)
			}
		}
		sort.Strings(expected)
		got, err := ProbeVisibilityGains(before, after, policy, principals)
		if err != nil {
			t.Fatalf("seed=%d probe: %v", seed, err)
		}
		if !reflect.DeepEqual(got.NewlyVisible, expected) {
			t.Fatalf("seed=%d gains mismatch: expected %v, got %v", seed, expected, got.NewlyVisible)
		}
		if got.Admission != "blocked" || got.Identity != "unresolved_cross_import_identity" || got.ReviewCompleteness != "sampled_principals_only_not_exhaustive" {
			t.Fatalf("seed=%d unsafe approval/completeness: %+v", seed, got)
		}
	}
}

func randomTreeForProbe(rng *rand.Rand, n int) Document {
	nodes := make([]Node, n)
	childCount := make(map[string]int)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("node-%02d", i)
		parent := ""
		if i > 0 {
			parent = fmt.Sprintf("node-%02d", rng.Intn(i))
		}
		nodes[i] = Node{ID: id, ParentID: parent, Order: childCount[parent], Title: id, Content: "content", SourceRef: "source.md", SourceDigestClaim: shaClaim}
		childCount[parent]++
	}
	rng.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	return Document{Schema: Schema, ProjectID: "project-1", SourceID: "source-1", Nodes: nodes, Edges: []Edge{}}
}

func chainForProbe(n int) (Document, Policy) {
	nodes := make([]Node, n)
	for i := 0; i < n; i++ {
		parent := ""
		if i > 0 {
			parent = fmt.Sprintf("node-%04d", i-1)
		}
		nodes[i] = Node{ID: fmt.Sprintf("node-%04d", i), ParentID: parent, Order: 0, Title: "node", Content: "example", SourceRef: "source.md", SourceDigestClaim: shaClaim}
	}
	d := Document{Schema: Schema, ProjectID: "project-1", SourceID: "source-1", Nodes: nodes, Edges: []Edge{}}
	p := Policy{ProjectID: d.ProjectID, Epoch: 1, HumanRootID: "owner", Rules: []Rule{
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "node-0000", Action: "read", Effect: "grant", Inherit: true},
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: fmt.Sprintf("node-%04d", n/2), Action: "read", Effect: "deny", Inherit: true},
	}}
	return d, p
}

func TestRepair002DeepChainNoUnintendedAdmission(t *testing.T) {
	d, p := chainForProbe(2500)
	r, err := ProbeVisibilityGains(d, d, p, []Principal{{Kind: "agent", ID: "worker"}})
	if err != nil || len(r.NewlyVisible) != 0 || r.Admission != "blocked" {
		t.Fatalf("deep chain: %+v err=%v", r, err)
	}
}

func TestRepair002InvalidPolicyAndPrincipalFailClosed(t *testing.T) {
	d, p := chainForProbe(8)
	p.Rules = append(p.Rules, Rule{PrincipalKind: "bot", PrincipalID: "guest", NodeID: "node-0000", Action: "read", Effect: "grant"})
	if _, err := ProbeVisibilityGains(d, d, p, []Principal{{Kind: "agent", ID: "worker"}}); err == nil {
		t.Fatal("malformed unrelated policy accepted")
	}
	p.Rules = p.Rules[:len(p.Rules)-1]
	if _, err := ProbeVisibilityGains(d, d, p, []Principal{{Kind: "agent", ID: "../escape"}}); err == nil {
		t.Fatal("invalid principal accepted")
	}
}

func TestRepair002ExplicitDenyAndExactRuleParity(t *testing.T) {
	d, p, _ := authFixture()
	p.Rules = []Rule{
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "root", Action: "read", Effect: "grant", Inherit: true},
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "root", Action: "read", Effect: "deny", Inherit: false},
	}
	got := projectedReadVisibility(d, p, Principal{Kind: "agent", ID: "worker"})
	if got["root"] || !got["child"] || !got["peer"] {
		t.Fatalf("exact-only deny incorrectly inherited: %v", got)
	}
	for _, n := range d.Nodes {
		if got[n.ID] != Authorize(d, p, Principal{Kind: "agent", ID: "worker"}, n.ID, "read").Allowed {
			t.Fatalf("incorrect projection for %s", n.ID)
		}
	}
}

func TestRepair002EmptyAndDuplicatePrincipalSets(t *testing.T) {
	d, p, principal := authFixture()
	empty, err := ProbeVisibilityGains(d, d, p, nil)
	if err != nil || !slices.Equal(empty.NewlyVisible, []string{}) || empty.Admission != "blocked" {
		t.Fatalf("nil principals: %+v err=%v", empty, err)
	}
	duplicate, err := ProbeVisibilityGains(d, d, p, []Principal{principal, principal})
	if err != nil || len(duplicate.NewlyVisible) != 0 || duplicate.Admission != "blocked" {
		t.Fatalf("duplicate principals: %+v err=%v", duplicate, err)
	}
}

func BenchmarkRepair002VisibilityChain(b *testing.B) {
	for _, n := range []int{100, 200, 400, 800, 1600} {
		b.Run(fmt.Sprintf("nodes_%d", n), func(b *testing.B) {
			d, p := chainForProbe(n)
			principal := []Principal{{Kind: "agent", ID: "worker"}}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				report, err := ProbeVisibilityGains(d, d, p, principal)
				if err != nil || report.Admission != "blocked" {
					b.Fatalf("invalid probe result %v %+v", err, report)
				}
			}
		})
	}
}
