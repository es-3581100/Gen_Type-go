package phasea

import (
	"errors"
	"fmt"
	"sort"
)

var actions = map[string]bool{"read": true, "write": true}
var effects = map[string]bool{"grant": true, "deny": true}

func validatePrincipal(p Principal) error {
	if p.Kind != "human" && p.Kind != "agent" {
		return errors.New("unknown principal kind")
	}
	if !identityPattern.MatchString(p.ID) {
		return errors.New("invalid principal id")
	}
	return nil
}

func ValidatePolicy(d Document, p Policy) error {
	if err := Validate(d); err != nil {
		return err
	}
	if p.ProjectID != d.ProjectID || p.Epoch == 0 {
		return errors.New("policy project/epoch invalid")
	}
	if !identityPattern.MatchString(p.HumanRootID) || p.Rules == nil {
		return errors.New("missing protected policy inputs")
	}
	byID := make(map[string]bool, len(d.Nodes))
	for _, n := range d.Nodes {
		byID[n.ID] = true
	}
	seen := make(map[string]bool, len(p.Rules))
	for _, r := range p.Rules {
		if err := validatePrincipal(Principal{Kind: r.PrincipalKind, ID: r.PrincipalID}); err != nil {
			return err
		}
		if !byID[r.NodeID] || !actions[r.Action] || !effects[r.Effect] {
			return fmt.Errorf("invalid or out-of-scope rule for %q", r.NodeID)
		}
		key := fmt.Sprintf("%s\x00%s\x00%s\x00%s\x00%s\x00%t", r.PrincipalKind, r.PrincipalID, r.NodeID, r.Action, r.Effect, r.Inherit)
		if seen[key] {
			return errors.New("duplicate policy rule")
		}
		seen[key] = true
	}
	return nil
}

// Authorize is a PURE evaluator. Caller must separately prove that Policy is
// the protected active policy and principal came from an authenticated issuer.
// The proof has neither of those components; this result MUST NOT gate I/O.
func Authorize(d Document, policy Policy, principal Principal, nodeID, action string) Decision {
	if action == "root" || action == "publish_policy" {
		return Decision{Reason: "protected operation unavailable in Phase A proof"}
	}
	if !actions[action] {
		return Decision{Reason: "unsupported action"}
	}
	if err := validatePrincipal(principal); err != nil {
		return Decision{Reason: "principal invalid"}
	}
	if err := ValidatePolicy(d, policy); err != nil {
		return Decision{Reason: "policy or graph invalid"}
	}
	byID := make(map[string]Node, len(d.Nodes))
	for _, n := range d.Nodes {
		byID[n.ID] = n
	}
	n, ok := byID[nodeID]
	if !ok {
		return Decision{Reason: "node not in project"}
	}
	path := map[string]bool{n.ID: true}
	ancestor := n
	for ancestor.ParentID != "" {
		ancestor = byID[ancestor.ParentID]
		path[ancestor.ID] = true
	}
	grant, deny := false, false
	for _, r := range policy.Rules {
		if r.PrincipalKind != principal.Kind || r.PrincipalID != principal.ID || r.Action != action {
			continue
		}
		if !(r.NodeID == nodeID || r.Inherit && path[r.NodeID]) {
			continue
		}
		if r.Effect == "deny" {
			deny = true
		} else {
			grant = true
		}
	}
	if deny {
		return Decision{Reason: "explicit deny"}
	}
	if grant {
		return Decision{Allowed: true, Reason: "matched scoped grant"}
	}
	return Decision{Reason: "default deny"}
}

// Probe limits keep the pure diagnostic helper bounded even if a future
// caller accidentally forwards an untrusted slice. They are NOT authority
// limits or a substitute for a request admission / authentication boundary.
const (
	MaxProbePrincipalInputs  = 4096
	MaxProbeUniquePrincipals = 64
	MaxProbePolicyRules      = 20000
)

type projectionRuleEffect struct {
	directGrant, directDeny       bool
	inheritedGrant, inheritedDeny bool
}

type projectionGraph struct {
	children  map[string][]string
	roots     []string
	nodeCount int
}

// All callers in this file must validate the graph first. No indexing of
// invalid topology is allowed to escape a test-only helper into authority.
func makeProjectionGraph(d Document) projectionGraph {
	graph := projectionGraph{children: make(map[string][]string, len(d.Nodes)), nodeCount: len(d.Nodes)}
	for _, n := range d.Nodes {
		graph.children[n.ParentID] = append(graph.children[n.ParentID], n.ID)
	}
	graph.roots = graph.children[""]
	return graph
}

// Build the rule index ONCE for the whole sampled probe. Non-read rules are
// skipped. Principal.Kind is part of the key, so human and agent namespaces
// cannot alias, even with identical ID strings.
func indexReadRules(policy Policy) map[Principal]map[string]projectionRuleEffect {
	result := make(map[Principal]map[string]projectionRuleEffect)
	for _, r := range policy.Rules {
		if r.Action != "read" {
			continue
		}
		p := Principal{Kind: r.PrincipalKind, ID: r.PrincipalID}
		nodeRules := result[p]
		if nodeRules == nil {
			nodeRules = make(map[string]projectionRuleEffect)
			result[p] = nodeRules
		}
		effect := nodeRules[r.NodeID]
		if r.Effect == "deny" {
			effect.directDeny = true
			if r.Inherit {
				effect.inheritedDeny = true
			}
		} else {
			effect.directGrant = true
			if r.Inherit {
				effect.inheritedGrant = true
			}
		}
		nodeRules[r.NodeID] = effect
	}
	return result
}

// projectReadWithIndex projects an ALREADY VALIDATED graph using an ALREADY
// VALIDATED policy's read-rule index. Its cost is O(nodes + relevant rules)
// only when called with precomputed inputs; building the shared index costs
// O(all policy rules) once per ProbeVisibilityGains call.
func projectReadWithIndex(graph projectionGraph, rules map[string]projectionRuleEffect) map[string]bool {
	visible := make(map[string]bool, graph.nodeCount)
	type pendingNode struct {
		id          string
		grant, deny bool
	}
	stack := make([]pendingNode, 0, graph.nodeCount)
	for _, root := range graph.roots {
		stack = append(stack, pendingNode{id: root})
	}
	for len(stack) > 0 {
		last := len(stack) - 1
		item := stack[last]
		stack = stack[:last]
		effect := rules[item.id]
		visible[item.id] = (item.grant || effect.directGrant) && !(item.deny || effect.directDeny)
		grant := item.grant || effect.inheritedGrant
		deny := item.deny || effect.inheritedDeny
		for _, child := range graph.children[item.id] {
			stack = append(stack, pendingNode{id: child, grant: grant, deny: deny})
		}
	}
	return visible
}

// projectedReadVisibility remains a compatibility helper for unit tests.
// Unlike projectReadWithIndex, its single-principal call necessarily includes
// O(all rules + nodes) setup. It never authorizes real data access.
func projectedReadVisibility(d Document, policy Policy, principal Principal) map[string]bool {
	return projectReadWithIndex(makeProjectionGraph(d), indexReadRules(policy)[principal])
}

// blockedProbeOnError is the fail-closed *diagnostic result contract* for
// failures before AssessCrossImport can produce a validated assessment. This
// marker is not an independently authenticated revision-admission decision.
func blockedProbeOnError() ChangeAssessment {
	return ChangeAssessment{
		Identity:           "unresolved_cross_import_identity",
		NewlyVisible:       []string{},
		ReviewCompleteness: "not_exhaustive",
		Admission:          "blocked",
		Reason:             "diagnostic visibility probe rejected; no trusted revision admission authority",
	}
}

// ProbeVisibilityGains compares a BOUNDED, CALLER-SELECTED principal sample.
// No sampled result is exhaustive authorization or revision admission. Cheap
// sample/rule budgets and untrusted principal *syntax* are checked before any
// graph-dependent hashing to avoid needless work on already rejected calls.
// Complete graph and policy validation remains mandatory before projection.
// Valid-input cost includes canonicalization and policy validation, plus
// O(all rules + P*(before nodes + after nodes)) for the indexed projection,
// where P <= MaxProbeUniquePrincipals.
func ProbeVisibilityGains(before, after Document, policy Policy, principals []Principal) (ChangeAssessment, error) {
	blocked := blockedProbeOnError()
	if len(principals) > MaxProbePrincipalInputs {
		return blocked, errors.New("sampled principal input budget exceeded")
	}
	if len(policy.Rules) > MaxProbePolicyRules {
		return blocked, errors.New("sampled policy rule budget exceeded")
	}

	// Validate EVERY supplied value even if another principal with the same
	// (Kind,ID) was seen earlier. Namespaces remain explicitly distinct.
	unique := make([]Principal, 0, min(len(principals), MaxProbeUniquePrincipals))
	seen := make(map[Principal]bool)
	for _, p := range principals {
		if err := validatePrincipal(p); err != nil {
			return blocked, err
		}
		if seen[p] {
			continue
		}
		if len(unique) >= MaxProbeUniquePrincipals {
			return blocked, errors.New("unique sampled principal budget exceeded")
		}
		seen[p] = true
		unique = append(unique, p)
	}

	assessment, err := AssessCrossImport(before, after)
	if err != nil {
		// AssessCrossImport intentionally returns no hashes on invalid
		// source graphs, but its zero-value assessment is not a suitable
		// caller-facing admission disposition. Never forward that zero.
		return blocked, err
	}
	if before.ProjectID != after.ProjectID {
		return assessment, errors.New("project changes cannot be compared")
	}
	if err := ValidatePolicy(before, policy); err != nil {
		return assessment, err
	}
	if err := ValidatePolicy(after, policy); err != nil {
		return assessment, err
	}

	beforeNodes := make(map[string]bool, len(before.Nodes))
	for _, n := range before.Nodes {
		beforeNodes[n.ID] = true
	}
	gain := make(map[string]bool)
	beforeGraph := makeProjectionGraph(before)
	afterGraph := makeProjectionGraph(after)
	readRules := indexReadRules(policy)
	for _, p := range unique {
		oldVisibility := projectReadWithIndex(beforeGraph, readRules[p])
		newVisibility := projectReadWithIndex(afterGraph, readRules[p])
		for _, n := range after.Nodes {
			if !beforeNodes[n.ID] {
				continue
			} // New nodes need distinct trusted review.
			if !oldVisibility[n.ID] && newVisibility[n.ID] {
				gain[n.ID] = true
			}
		}
	}
	for id := range gain {
		assessment.NewlyVisible = append(assessment.NewlyVisible, id)
	}
	sort.Strings(assessment.NewlyVisible)
	assessment.ReviewCompleteness = "sampled_principals_only_not_exhaustive"
	if len(assessment.NewlyVisible) > 0 {
		assessment.Reason = "access-relaxing structural change detected; exact-scope protected approval not available"
	}
	return assessment, nil
}
