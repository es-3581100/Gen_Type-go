package phasea

import (
	"fmt"
	"testing"
	"time"
)

func TestIndependentVisibilityScaling(t *testing.T) {
	for _, count := range []int{100, 200, 400, 800} {
		nodes := make([]Node, 0, count+2)
		source := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		nodes = append(nodes, Node{ID: "root", ParentID: "", Order: 0, Title: "root", SourceRef: "fixture", SourceDigestClaim: source})
		nodes = append(nodes, Node{ID: "restricted", ParentID: "root", Order: 0, Title: "restricted", SourceRef: "fixture", SourceDigestClaim: source})
		for i := 0; i < count; i++ {
			nodes = append(nodes, Node{ID: fmt.Sprintf("leaf-%d", i), ParentID: "restricted", Order: i, Title: "leaf", SourceRef: "fixture", SourceDigestClaim: source})
		}
		before := Document{Schema: Schema, ProjectID: "project", SourceID: "source", Nodes: nodes, Edges: []Edge{}}
		after := before
		after.Nodes = append([]Node(nil), before.Nodes...)
		for i := 2; i < len(after.Nodes); i++ {
			after.Nodes[i].ParentID = "root"
			after.Nodes[i].Order = i - 1
		}
		pol := Policy{ProjectID: "project", Epoch: 1, HumanRootID: "owner", Rules: []Rule{
			{PrincipalKind: "agent", PrincipalID: "agent", NodeID: "root", Action: "read", Effect: "grant", Inherit: true},
			{PrincipalKind: "agent", PrincipalID: "agent", NodeID: "restricted", Action: "read", Effect: "deny", Inherit: true},
		}}
		start := time.Now()
		change, err := ProbeVisibilityGains(before, after, pol, []Principal{{Kind: "agent", ID: "agent"}})
		dur := time.Since(start)
		if err != nil {
			t.Fatal(err)
		}
		if change.Admission != "blocked" || len(change.NewlyVisible) != count {
			t.Fatalf("n=%d result=%+v", count, change)
		}
		t.Logf("sampled_principals=1 leaves=%d duration=%s admission=%s", count, dur, change.Admission)
	}
}
