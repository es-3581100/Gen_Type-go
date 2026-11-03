package phasea

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

func auditGraph(seed int64, n int, forest bool) Document {
	rng := rand.New(rand.NewSource(seed))
	nodes := make([]Node, 0, n)
	nextOrder := map[string]int{}
	for i := 0; i < n; i++ {
		parent := ""
		if i > 0 && (!forest || rng.Intn(5) != 0) {
			parent = fmt.Sprintf("n-%d", rng.Intn(i))
		}
		id := fmt.Sprintf("n-%d", i)
		nodes = append(nodes, Node{ID: id, ParentID: parent, Order: nextOrder[parent], Title: id, Content: "ok", SourceRef: "file", SourceDigestClaim: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
		nextOrder[parent]++
	}
	rng.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
	return Document{Schema: Schema, ProjectID: "proj", SourceID: "source", Nodes: nodes, Edges: []Edge{}}
}

func auditPolicy(seed int64, n int) Policy {
	rng := rand.New(rand.NewSource(seed))
	principals := []Principal{{Kind: "agent", ID: "p"}, {Kind: "human", ID: "p"}, {Kind: "agent", ID: "q"}, {Kind: "human", ID: "q"}}
	policy := Policy{ProjectID: "proj", Epoch: 3, HumanRootID: "owner", Rules: []Rule{}}
	seen := map[Rule]bool{}
	for i := 0; i < 180; i++ {
		pr := principals[rng.Intn(len(principals))]
		rule := Rule{PrincipalKind: pr.Kind, PrincipalID: pr.ID, NodeID: fmt.Sprintf("n-%d", rng.Intn(n)), Action: []string{"read", "write"}[rng.Intn(2)], Effect: []string{"grant", "deny"}[rng.Intn(2)], Inherit: rng.Intn(2) == 0}
		if !seen[rule] {
			seen[rule] = true
			policy.Rules = append(policy.Rules, rule)
		}
	}
	return policy
}

func TestAudit002ForestParityAndPermutations(t *testing.T) {
	// Independent source fixtures and expected calculation using the unchanged Authorize reference.
	principals := []Principal{{Kind: "agent", ID: "p"}, {Kind: "human", ID: "p"}, {Kind: "agent", ID: "q"}, {Kind: "human", ID: "q"}}
	for seed := int64(1100); seed < 1240; seed++ {
		before := auditGraph(seed, 16, seed%2 == 0)
		after := auditGraph(seed+101, 16, seed%3 == 0)
		policy := auditPolicy(seed+99, 16)
		if err := ValidatePolicy(before, policy); err != nil {
			t.Fatalf("pre seed %d: %v", seed, err)
		}
		if err := ValidatePolicy(after, policy); err != nil {
			t.Fatalf("post seed %d: %v", seed, err)
		}
		chosen := []Principal{principals[0], principals[1], principals[0], principals[3]}
		expected := []string{}
		for _, node := range after.Nodes {
			for _, pr := range chosen {
				if !Authorize(before, policy, pr, node.ID, "read").Allowed && Authorize(after, policy, pr, node.ID, "read").Allowed {
					expected = append(expected, node.ID)
					break
				}
			}
		}
		sort.Strings(expected)
		got, err := ProbeVisibilityGains(before, after, policy, chosen)
		if err != nil {
			t.Fatalf("probe seed %d: %v", seed, err)
		}
		if !reflect.DeepEqual(expected, got.NewlyVisible) || got.Admission != "blocked" || got.Identity != "unresolved_cross_import_identity" || got.ReviewCompleteness != "sampled_principals_only_not_exhaustive" {
			t.Fatalf("seed %d expected %v got %+v", seed, expected, got)
		}
	}
}
func TestAudit002WorstRulePrecedenceAndMissingNodes(t *testing.T) {
	before := auditGraph(10, 20, true)
	after := auditGraph(20, 21, true)
	p := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
	for _, node := range before.Nodes {
		if node.ID == "n-0" || node.ID == "n-1" || node.ID == "n-2" {
			p.Rules = append(p.Rules, Rule{PrincipalKind: "agent", PrincipalID: "p", NodeID: node.ID, Action: "read", Effect: "grant", Inherit: true})
			p.Rules = append(p.Rules, Rule{PrincipalKind: "agent", PrincipalID: "p", NodeID: node.ID, Action: "read", Effect: "deny", Inherit: false})
		}
	}
	pr := Principal{Kind: "agent", ID: "p"}
	got, err := ProbeVisibilityGains(before, after, p, []Principal{pr})
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{}
	for _, n := range after.Nodes {
		if n.ID == "n-20" {
			continue
		}
		if !Authorize(before, p, pr, n.ID, "read").Allowed && Authorize(after, p, pr, n.ID, "read").Allowed {
			expected = append(expected, n.ID)
		}
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(got.NewlyVisible, expected) {
		t.Fatalf("%v != %v", got.NewlyVisible, expected)
	}
}
func TestAudit002DuplicatePrincipalAmplification(t *testing.T) {
	before := auditGraph(33, 200, false)
	after := auditGraph(34, 200, false)
	p := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
	for _, count := range []int{1, 40, 400, 2000} {
		principals := make([]Principal, count)
		for i := range principals {
			principals[i] = Principal{Kind: "agent", ID: "p"}
		}
		start := time.Now()
		got, err := ProbeVisibilityGains(before, after, p, principals)
		elapsed := time.Since(start)
		if err != nil || got.Admission != "blocked" || len(got.NewlyVisible) != 0 {
			t.Fatalf("count %d: %+v err %v", count, got, err)
		}
		t.Logf("duplicate principals=%d nodes=200 duration_ms=%.3f", count, float64(elapsed.Microseconds())/1000)
	}
}
func TestAudit002RejectInvalidRulesRegardlessOfAffectedPrincipal(t *testing.T) {
	d := auditGraph(42, 20, true)
	p := auditPolicy(42, 20)
	bad := p
	bad.Rules = append([]Rule(nil), p.Rules...)
	bad.Rules = append(bad.Rules, Rule{PrincipalKind: "agent", PrincipalID: "p", NodeID: "n-0", Action: "DELETE", Effect: "grant", Inherit: true})
	if _, err := ProbeVisibilityGains(d, d, bad, []Principal{{Kind: "agent", ID: "q"}}); err == nil {
		t.Fatal("invalid unrelated rule accepted")
	}
}

func TestAudit002UnrelatedRuleWork(t *testing.T) {
	d := auditGraph(91, 100, true)
	testPrincipal := Principal{Kind: "agent", ID: "target"}
	for _, others := range []int{0, 4000} {
		p := Policy{ProjectID: "proj", Epoch: 1, HumanRootID: "owner", Rules: []Rule{}}
		count := 0
		for k := 0; k < 5 && count < others; k++ {
			for j := 0; j < 100 && count < others; j++ {
				for _, act := range []string{"read", "write"} {
					for _, eff := range []string{"grant", "deny"} {
						for _, inh := range []bool{false, true} {
							p.Rules = append(p.Rules, Rule{PrincipalKind: "human", PrincipalID: fmt.Sprintf("bystander-%d", k), NodeID: fmt.Sprintf("n-%d", j), Action: act, Effect: eff, Inherit: inh})
							count++
							if count >= others {
								break
							}
						}
					}
				}
			}
		}
		start := time.Now()
		r, err := ProbeVisibilityGains(d, d, p, []Principal{testPrincipal})
		ms := float64(time.Since(start).Microseconds()) / 1000
		if err != nil || r.Admission != "blocked" || len(r.NewlyVisible) != 0 {
			t.Fatalf("rules=%d %+v error %v", others, r, err)
		}
		t.Logf("unrelated_rules=%d time_ms=%.3f", len(p.Rules), ms)
	}
}
