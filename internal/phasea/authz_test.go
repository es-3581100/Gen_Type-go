package phasea

import (
	"slices"
	"testing"
)

func authFixture() (Document, Policy, Principal) {
	d := fixture()
	policy := Policy{ProjectID: d.ProjectID, Epoch: 1, HumanRootID: "owner", Rules: []Rule{
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "root", Action: "read", Effect: "grant", Inherit: true},
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "child", Action: "read", Effect: "deny", Inherit: true},
	}}
	return d, policy, Principal{Kind: "agent", ID: "worker"}
}

func TestAuthzDenyOverridesInheritedGrant(t *testing.T) {
	d, p, principal := authFixture()
	for _, v := range []struct {
		node, action string
		allowed      bool
	}{
		{"root", "read", true}, {"peer", "read", true}, {"child", "read", false},
		{"root", "write", false}, {"lost", "read", false}, {"root", "root", false},
		{"root", "publish_policy", false}, {"root", "bogus", false},
	} {
		dec := Authorize(d, p, principal, v.node, v.action)
		if dec.Allowed != v.allowed {
			t.Fatalf("%s/%s: %+v", v.node, v.action, dec)
		}
	}
}

func TestAgentCannotBecomeRootThroughPolicy(t *testing.T) {
	d, p, principal := authFixture()
	p.HumanRootID = "worker" // same name still NOT an authenticated human root
	if Authorize(d, p, principal, "root", "root").Allowed {
		t.Fatal("agent assumed root")
	}
	principal.Kind = "human"
	if Authorize(d, p, principal, "root", "root").Allowed {
		t.Fatal("self-declared human assumed root")
	}
}

func TestCrossProjectAndDifferentNamespaceDeny(t *testing.T) {
	d, p, principal := authFixture()
	principal.Kind = "human"
	if Authorize(d, p, principal, "root", "read").Allowed {
		t.Fatal("kind conflation")
	}
	principal.Kind = "agent"
	p.ProjectID = "other"
	if Authorize(d, p, principal, "root", "read").Allowed {
		t.Fatal("cross-project policy")
	}
}

func TestMalformedPoliciesFailClosed(t *testing.T) {
	tests := map[string]func(*Policy){
		"missing epoch":                func(p *Policy) { p.Epoch = 0 },
		"missing protected human root": func(p *Policy) { p.HumanRootID = "" },
		"null rules":                   func(p *Policy) { p.Rules = nil },
		"unknown kind":                 func(p *Policy) { p.Rules[0].PrincipalKind = "bot" },
		"unknown action":               func(p *Policy) { p.Rules[0].Action = "publish_policy" },
		"unknown effect":               func(p *Policy) { p.Rules[0].Effect = "root" },
		"unknown target":               func(p *Policy) { p.Rules[0].NodeID = "lost" },
		"duplicated rule":              func(p *Policy) { p.Rules = append(p.Rules, p.Rules[0]) },
		"foreign project":              func(p *Policy) { p.ProjectID = "elsewhere" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			d, p, principal := authFixture()
			mutate(&p)
			if Authorize(d, p, principal, "root", "read").Allowed {
				t.Fatal("policy failed open")
			}
		})
	}
}

func TestVisibilityGainOnReparent(t *testing.T) {
	d, p, principal := authFixture()
	// Agent has a broad root grant, but child's explicit deny means child
	// remains hidden. Give the peer branch an inherited deny; then moving
	// a node from that branch to root creates an observed read gain.
	d.Nodes = append(d.Nodes, Node{ID: "leaf", ParentID: "peer", Order: 0, Title: "Leaf", Content: "secret", SourceRef: "f.md", SourceDigestClaim: shaClaim})
	p.Rules = []Rule{
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "root", Action: "read", Effect: "grant", Inherit: true},
		{PrincipalKind: "agent", PrincipalID: "worker", NodeID: "peer", Action: "read", Effect: "deny", Inherit: true},
	}
	after := d
	after.Nodes = append([]Node{}, d.Nodes...)
	for i := range after.Nodes {
		if after.Nodes[i].ID == "leaf" {
			after.Nodes[i].ParentID = "root"
			after.Nodes[i].Order = 2
		}
	}
	if Authorize(d, p, principal, "leaf", "read").Allowed {
		t.Fatal("baseline should deny")
	}
	if !Authorize(after, p, principal, "leaf", "read").Allowed {
		t.Fatal("after move should allow")
	}
	report, err := ProbeVisibilityGains(d, after, p, []Principal{principal})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(report.NewlyVisible, "leaf") {
		t.Fatalf("gain not detected: %+v", report)
	}
	if report.Admission != "blocked" || report.ReviewCompleteness != "sampled_principals_only_not_exhaustive" {
		t.Fatalf("unsafe claim: %+v", report)
	}
}

func TestZeroVisibilityGainDoesNotAuthorize(t *testing.T) {
	d, p, principal := authFixture()
	r, err := ProbeVisibilityGains(d, d, p, []Principal{principal})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.NewlyVisible) != 0 || r.Admission != "blocked" {
		t.Fatal("non-exhaustive probe allowed a revision")
	}
}

func TestInvalidPrincipalAlwaysDenied(t *testing.T) {
	d, p, _ := authFixture()
	for _, x := range []Principal{{Kind: "human", ID: ""}, {Kind: "root", ID: "worker"}, {Kind: "agent", ID: "../../etc"}} {
		if Authorize(d, p, x, "root", "read").Allowed {
			t.Fatalf("allowed %+v", x)
		}
	}
}
