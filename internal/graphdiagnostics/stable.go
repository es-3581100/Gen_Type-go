package graphdiagnostics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
	"regexp"
	"sort"
	"strconv"
	"unicode/utf8"
)

var localID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@/-]{0,127}$`)
var digestClaim = regexp.MustCompile(`^[0-9a-f]{64}$`)

func problem(code, path, message, action string) Diagnostic {
	return Diagnostic{Code: code, Path: path, Message: message, Action: action}
}

// stableDiagnostic is advisory *only*. It runs after DecodeStrict has already
// rejected the graph. It never accepts an input or supplies authorization.
// Every inspection is in wire order or over explicitly sorted keys.
func stableDiagnostic(raw []byte) Diagnostic {
	if len(raw) > phasea.MaxInputBytes {
		return problem("INPUT_TOO_LARGE", "$", "input exceeds byte budget", "reduce input to 2 MiB or less")
	}
	if !utf8.Valid(raw) {
		return problem("INVALID_UTF8", "$", "input is not UTF-8", "supply valid UTF-8")
	}
	if !json.Valid(raw) {
		return problem("MALFORMED_JSON", "$", "input is not one valid JSON value", "provide one complete JSON graph object")
	}
	// DecodeStrict checks escaped surrogates before its depth and duplicate-key
	// pass, so the advisory classifier keeps that precedence.
	if hasUnpairedSurrogateEscape(raw) {
		return problem("INVALID_ESCAPE", "$", "string contains an unpaired UTF-16 surrogate escape", "use a complete surrogate pair or the literal character")
	}
	switch duplicate, tooDeep := scanJSONKeys(raw); {
	case duplicate:
		return problem("DUPLICATE_JSON_KEY", "$", "JSON object has a duplicate decoded key", "remove duplicated object keys")
	case tooDeep:
		return problem("NESTING_TOO_DEEP", "$", "JSON nesting exceeds the depth budget", "reduce JSON nesting to 96 levels or fewer")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil || top == nil {
		return problem("FIELD_TYPE", "$", "graph must be a JSON object", "provide one graph object")
	}
	if d, ok := checkObject(top, "$", []string{"schema", "project_id", "source_id", "nodes", "edges"}); ok {
		return d
	}
	for _, name := range []string{"schema", "project_id", "source_id"} {
		if !stringField(top[name]) {
			return problem("FIELD_TYPE", "$."+name, "field must be a string", "use the exact JSON field type")
		}
	}
	var nodes, edges []json.RawMessage
	if err := json.Unmarshal(top["nodes"], &nodes); err != nil || nodes == nil {
		return problem("FIELD_TYPE", "$.nodes", "nodes must be a JSON array", "provide an explicit nodes array")
	}
	if err := json.Unmarshal(top["edges"], &edges); err != nil || edges == nil {
		return problem("FIELD_TYPE", "$.edges", "edges must be a JSON array", "provide an explicit edges array")
	}
	for i, item := range nodes {
		base := fmt.Sprintf("$.nodes[%d]", i)
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil || obj == nil {
			return problem("FIELD_TYPE", base, "node must be an object", "provide a node object")
		}
		if d, ok := checkObject(obj, base, []string{"id", "parent_id", "order", "title", "content", "source_ref", "source_digest_claim"}); ok {
			return d
		}
		for _, k := range []string{"id", "parent_id", "title", "content", "source_ref", "source_digest_claim"} {
			if !stringField(obj[k]) {
				return problem("FIELD_TYPE", base+"."+k, "node field must be a string", "supply a string")
			}
		}
		var order int
		if err := json.Unmarshal(obj["order"], &order); err != nil {
			return problem("FIELD_TYPE", base+".order", "order must be a JSON integer", "use an integer ordinal")
		}
	}
	for i, item := range edges {
		base := fmt.Sprintf("$.edges[%d]", i)
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil || obj == nil {
			return problem("FIELD_TYPE", base, "edge must be an object", "provide an edge object")
		}
		if d, ok := checkObject(obj, base, []string{"from", "to", "kind"}); ok {
			return d
		}
		for _, k := range []string{"from", "to", "kind"} {
			if !stringField(obj[k]) {
				return problem("FIELD_TYPE", base+"."+k, "edge field must be a string", "supply a string")
			}
		}
	}
	var doc phasea.Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		return problem("FIELD_TYPE", "$", "graph contains invalid field types", "correct the graph wire types")
	}
	if doc.Schema != phasea.Schema {
		return problem("BAD_SCHEMA", "$.schema", "unsupported graph schema", "use gen-type-go/phase-a-graph-v0")
	}
	if !localID.MatchString(doc.ProjectID) || !localID.MatchString(doc.SourceID) {
		return problem("INVALID_ID", "$", "project/source ID is invalid", "use valid local identifier strings")
	}
	if len(doc.Nodes) == 0 {
		return problem("INVALID_HIERARCHY", "$.nodes", "graph has no root nodes", "supply a rooted acyclic hierarchy")
	}
	if len(doc.Nodes) > phasea.MaxNodes || len(doc.Edges) > phasea.MaxEdges {
		return problem("INPUT_TOO_LARGE", "$", "graph exceeds record budget", "reduce nodes or edges")
	}
	byID := make(map[string]int, len(doc.Nodes))
	kids := make(map[string][]int, len(doc.Nodes))
	orders := make(map[string]map[int]bool)
	for i, n := range doc.Nodes {
		base := fmt.Sprintf("$.nodes[%d]", i)
		if !localID.MatchString(n.ID) {
			return problem("INVALID_ID", base+".id", "node ID is invalid", "use a valid local node ID")
		}
		if n.ParentID != "" && !localID.MatchString(n.ParentID) {
			return problem("INVALID_ID", base+".parent_id", "parent node ID is invalid", "use a valid local parent ID or an empty root parent")
		}
		if _, ok := byID[n.ID]; ok {
			return problem("DUPLICATE_NODE", base+".id", "node ID is repeated", "use distinct node IDs")
		}
		byID[n.ID] = i
		if n.Order < 0 || n.Order > phasea.MaxNodes {
			return problem("INVALID_ORDER", base+".order", "node ordinal is outside allowed range", "use contiguous sibling orders from zero")
		}
		if len(n.Title) > 4096 || len(n.Content) > 262144 || len(n.SourceRef) > 4096 {
			return problem("INVALID_GRAPH", base, "node exceeds field budget", "shorten oversized node fields")
		}
		if n.SourceRef == "" || !digestClaim.MatchString(n.SourceDigestClaim) {
			return problem("INVALID_SOURCE_CLAIM", base, "source reference or digest claim is invalid", "provide a source reference and 64 lowercase hex claim (unverified)")
		}
		kids[n.ParentID] = append(kids[n.ParentID], i)
		if orders[n.ParentID] == nil {
			orders[n.ParentID] = make(map[int]bool)
		}
		if orders[n.ParentID][n.Order] {
			return problem("INVALID_ORDER", base+".order", "sibling order is repeated", "use unique contiguous sibling ordinals")
		}
		orders[n.ParentID][n.Order] = true
	}
	// Specific parent faults come before the aggregate no-root fault, as in
	// phasea.Validate; otherwise a lone dangling parent reads as "no root".
	for i, n := range doc.Nodes {
		if n.ParentID == n.ID {
			return problem("INVALID_PARENT", fmt.Sprintf("$.nodes[%d].parent_id", i), "node cannot parent itself", "use a different existing parent or empty root parent")
		}
		if n.ParentID != "" {
			if _, ok := byID[n.ParentID]; !ok {
				return problem("INVALID_PARENT", fmt.Sprintf("$.nodes[%d].parent_id", i), "parent node ID is missing", "reference an existing node ID")
			}
		}
	}
	roots := kids[""]
	if len(roots) == 0 {
		return problem("INVALID_HIERARCHY", "$.nodes", "graph has no root", "supply a rooted hierarchy")
	}
	parentNames := make([]string, 0, len(orders))
	for name := range orders {
		parentNames = append(parentNames, name)
	}
	sort.Strings(parentNames)
	for _, parent := range parentNames {
		seen := orders[parent]
		for i := 0; i < len(seen); i++ {
			if !seen[i] {
				return problem("INVALID_ORDER", "$.nodes", "sibling orders contain a gap", "use ordinals 0 through child count minus one")
			}
		}
	}
	// Iterative reachability avoids recursion; it is only an explanatory hint.
	visited := make(map[int]bool, len(doc.Nodes))
	queue := append([]int{}, roots...)
	for pos := 0; pos < len(queue); pos++ {
		i := queue[pos]
		if visited[i] {
			return problem("INVALID_HIERARCHY", "$.nodes", "hierarchy contains repeated traversal", "repair cycles")
		}
		visited[i] = true
		for _, child := range kids[doc.Nodes[i].ID] {
			queue = append(queue, child)
		}
	}
	if len(visited) != len(doc.Nodes) {
		return problem("INVALID_HIERARCHY", "$.nodes", "hierarchy contains unreachable or cyclic nodes", "repair the parent hierarchy")
	}
	edgeKinds := map[string]bool{"links_to": true, "related_to": true, "depends_on": true}
	seenEdges := make(map[string]bool, len(doc.Edges))
	for i, e := range doc.Edges {
		base := fmt.Sprintf("$.edges[%d]", i)
		if !edgeKinds[e.Kind] {
			return problem("INVALID_EDGE_KIND", base+".kind", "unsupported explicit edge kind", "use links_to, depends_on, or related_to")
		}
		if _, ok := byID[e.From]; !ok {
			return problem("INVALID_EDGE_ENDPOINT", base+".from", "explicit edge source is missing", "reference an existing node ID")
		}
		if _, ok := byID[e.To]; !ok {
			return problem("INVALID_EDGE_ENDPOINT", base+".to", "explicit edge target is missing", "reference an existing node ID")
		}
		key := e.From + "\x00" + e.Kind + "\x00" + e.To
		if seenEdges[key] {
			return problem("DUPLICATE_EDGE", base, "explicit edge is repeated", "remove duplicate edges")
		}
		seenEdges[key] = true
	}
	return problem("INVALID_GRAPH", "$", "strict graph decoder rejected this input", "correct the graph against the strict Phase A schema; no partial input is admitted")
}
func stringField(raw json.RawMessage) bool {
	var s string
	return json.Unmarshal(raw, &s) == nil
}
func checkObject(obj map[string]json.RawMessage, base string, required []string) (Diagnostic, bool) {
	allowed := make(map[string]bool, len(required))
	for _, k := range required {
		allowed[k] = true
	}
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !allowed[k] {
			return problem("UNKNOWN_FIELD", base+pathKey(k), "undeclared or case-aliased JSON field", "use only exact schema field names"), true
		}
	}
	for _, k := range required {
		val, found := obj[k]
		if !found {
			return problem("MISSING_FIELD", base+"."+k, "required field is missing", "supply the required field"), true
		}
		if bytes.Equal(bytes.TrimSpace(val), []byte("null")) {
			return problem("NULL_FIELD", base+"."+k, "required field is null", "supply a non-null field of the proper type"), true
		}
	}
	return Diagnostic{}, false
}

// maxJSONDepth mirrors the strict decoder's nesting budget (phasea.scanJSONValue).
const maxJSONDepth = 96

// scanJSONKeys scans decoded keys in wire order and reports the first duplicate
// key, or that nesting passes maxJSONDepth, whichever the wire reaches first. It
// never accepts JSON or overrides the strict parser's independent rejection.
//
// Every finding or decode error stops the whole scan. Returning "no finding"
// from a nested call without consuming its token would leave the enclosing loop
// looking at the same token forever, so a 98-level array used to hang diagnose.
func scanJSONKeys(raw []byte) (duplicate, tooDeep bool) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	var walk func(int) (stop bool)
	walk = func(depth int) bool {
		if depth > maxJSONDepth {
			tooDeep = true
			return true
		}
		tok, err := dec.Token()
		if err != nil {
			return true
		}
		delim, ok := tok.(json.Delim)
		if !ok {
			return false
		}
		switch delim {
		case '{':
			seen := make(map[string]bool)
			for dec.More() {
				key, err := dec.Token()
				name, isString := key.(string)
				if err != nil || !isString {
					return true
				}
				if seen[name] {
					duplicate = true
					return true
				}
				seen[name] = true
				if walk(depth + 1) {
					return true
				}
			}
			_, _ = dec.Token()
		case '[':
			for dec.More() {
				if walk(depth + 1) {
					return true
				}
			}
			_, _ = dec.Token()
		}
		return false
	}
	walk(0)
	return duplicate, tooDeep
}

// hasUnpairedSurrogateEscape mirrors the strict decoder's rejection of \uD800-\uDFFF
// escapes that are not a high/low pair. It runs only after json.Valid, so every
// escape is well formed and every string is terminated.
func hasUnpairedSurrogateEscape(raw []byte) bool {
	inString := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if !inString {
			inString = c == '"'
			continue
		}
		switch c {
		case '"':
			inString = false
		case '\\':
			i++
			if i >= len(raw) || raw[i] != 'u' {
				continue
			}
			unit, ok := hex4(raw, i+1)
			if !ok {
				return false
			}
			i += 4
			switch {
			case unit >= 0xDC00 && unit <= 0xDFFF:
				return true
			case unit >= 0xD800 && unit <= 0xDBFF:
				if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
					return true
				}
				low, ok := hex4(raw, i+3)
				if !ok || low < 0xDC00 || low > 0xDFFF {
					return true
				}
				i += 6
			}
		}
	}
	return false
}

func hex4(raw []byte, at int) (rune, bool) {
	if at+4 > len(raw) {
		return 0, false
	}
	v, err := strconv.ParseUint(string(raw[at:at+4]), 16, 16)
	return rune(v), err == nil
}

// maxPathKeyBytes bounds how much attacker-controlled key text a path echoes.
const maxPathKeyBytes = 64

var simpleKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)

// pathKey renders an object key as a path segment. Plain identifiers use dot
// notation; anything else is bracket-quoted, ASCII-escaped and truncated, so a
// hostile key cannot forge extra path segments, carry terminal control bytes,
// or inflate the report past its output budget.
func pathKey(key string) string {
	if simpleKey.MatchString(key) {
		return "." + key
	}
	truncated := false
	if len(key) > maxPathKeyBytes {
		cut := maxPathKeyBytes
		for cut > 0 && !utf8.RuneStart(key[cut]) {
			cut--
		}
		key, truncated = key[:cut], true
	}
	quoted := strconv.QuoteToASCII(key)
	if truncated {
		quoted = quoted[:len(quoted)-1] + `...(truncated)"`
	}
	return "[" + quoted + "]"
}
