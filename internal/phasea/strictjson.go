package phasea

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// DecodeStrict rejects duplicate keys at EVERY object depth, unknown fields,
// malformed UTF-8, unpaired JSON escaped surrogates, and trailing data.
func DecodeStrict(raw []byte) (Document, error) {
	var doc Document
	if len(raw) > MaxInputBytes {
		return doc, errors.New("input exceeds byte budget")
	}
	if !utf8.Valid(raw) {
		return doc, errors.New("input is not UTF-8")
	}
	if err := validateEscapedUnicode(raw); err != nil {
		return doc, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := scanJSONValue(dec, 0); err != nil {
		return doc, err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return doc, errors.New("trailing JSON value")
		}
		return doc, fmt.Errorf("trailing JSON: %w", err)
	}
	// Check the decoded JSON object keys against the exact wire contract
	// *before* binding Go structs. encoding/json's struct matching is
	// case-insensitive even with DisallowUnknownFields enabled, so only
	// shape validation can reject case-alias attacks reliably.
	if err := validateGraphWireShape(raw); err != nil {
		return Document{}, err
	}
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields() // defense in depth, not exact-case enforcement
	if err := dec.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("decode graph: %w", err)
	}
	if doc.Nodes == nil || doc.Edges == nil {
		return Document{}, errors.New("nodes and edges must be JSON arrays, not omitted or null")
	}
	if err := Validate(doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func scanJSONValue(dec *json.Decoder, depth int) error {
	if depth > 96 {
		return errors.New("JSON nesting exceeds depth budget")
	}
	tok, err := dec.Token()
	if err != nil {
		return fmt.Errorf("JSON token: %w", err)
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("non-string object key")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = true
			if err := scanJSONValue(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("malformed object terminator")
		}
	case '[':
		for dec.More() {
			if err := scanJSONValue(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("malformed array terminator")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}

// encoding/json substitutes the replacement rune for invalid \u surrogate
// sequences. The source dialect rejects these rather than silently repairing.
// Syntax beyond escaped strings is checked by json.Decoder.
func validateEscapedUnicode(raw []byte) error {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		i++
		for ; i < len(raw); i++ {
			if raw[i] == '"' {
				break
			}
			if raw[i] != '\\' {
				continue
			}
			i++
			if i >= len(raw) {
				return errors.New("dangling JSON string escape")
			}
			if raw[i] != 'u' {
				continue
			}
			if i+4 >= len(raw) {
				return errors.New("incomplete unicode escape")
			}
			high, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
			if err != nil {
				return errors.New("malformed unicode escape")
			}
			i += 4
			if high >= 0xDC00 && high <= 0xDFFF {
				return errors.New("unpaired low surrogate")
			}
			if high >= 0xD800 && high <= 0xDBFF {
				if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
					return errors.New("unpaired high surrogate")
				}
				low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
				if err != nil || low < 0xDC00 || low > 0xDFFF {
					return errors.New("invalid surrogate pair")
				}
				i += 6
			}
		}
		if i >= len(raw) {
			return errors.New("unterminated JSON string")
		}
	}
	return nil
}

// Struct decoding accepts JSON null as a zero-value string or integer; that
// would be ambiguous for identity, parent, and ordinal fields. Require each
// wire field to be present and non-null, and require array members to be objects.
func validateGraphWireShape(raw []byte) error {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return err
	}
	if top == nil {
		return errors.New("graph must be an object")
	}
	if err := requireExactFields(top, "graph", "schema", "project_id", "source_id", "nodes", "edges"); err != nil {
		return err
	}
	for _, k := range []string{"schema", "project_id", "source_id", "nodes", "edges"} {
		if err := mustField(top, k); err != nil {
			return err
		}
	}
	for _, k := range []string{"schema", "project_id", "source_id"} {
		if err := mustString(top[k]); err != nil {
			return fmt.Errorf("%s: %w", k, err)
		}
	}
	var nodes []json.RawMessage
	if err := json.Unmarshal(top["nodes"], &nodes); err != nil {
		return fmt.Errorf("nodes: %w", err)
	}
	if nodes == nil {
		return errors.New("nodes must be array")
	}
	for i, item := range nodes {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil {
			return fmt.Errorf("node %d: %w", i, err)
		}
		if obj == nil {
			return fmt.Errorf("node %d must be object", i)
		}
		if err := requireExactFields(obj, fmt.Sprintf("node %d", i),
			"id", "parent_id", "order", "title", "content", "source_ref", "source_digest_claim"); err != nil {
			return err
		}
		for _, k := range []string{"id", "parent_id", "order", "title", "content", "source_ref", "source_digest_claim"} {
			if err := mustField(obj, k); err != nil {
				return fmt.Errorf("node %d: %w", i, err)
			}
			if k == "order" {
				continue
			}
			if err := mustString(obj[k]); err != nil {
				return fmt.Errorf("node %d field %s: %w", i, k, err)
			}
		}
		var num json.Number
		d := json.NewDecoder(bytes.NewReader(obj["order"]))
		d.UseNumber()
		if err := d.Decode(&num); err != nil {
			return fmt.Errorf("node %d order: %w", i, err)
		}
		if _, err := num.Int64(); err != nil {
			return fmt.Errorf("node %d order must be integer: %w", i, err)
		}
	}
	var edges []json.RawMessage
	if err := json.Unmarshal(top["edges"], &edges); err != nil {
		return fmt.Errorf("edges: %w", err)
	}
	if edges == nil {
		return errors.New("edges must be array")
	}
	for i, item := range edges {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil {
			return fmt.Errorf("edge %d: %w", i, err)
		}
		if obj == nil {
			return fmt.Errorf("edge %d must be object", i)
		}
		if err := requireExactFields(obj, fmt.Sprintf("edge %d", i), "from", "to", "kind"); err != nil {
			return err
		}
		for _, k := range []string{"from", "to", "kind"} {
			if err := mustField(obj, k); err != nil {
				return fmt.Errorf("edge %d: %w", i, err)
			}
			if err := mustString(obj[k]); err != nil {
				return fmt.Errorf("edge %d field %s: %w", i, k, err)
			}
		}
	}
	return nil
}

// requireExactFields forbids unexpected field names in EVERY graph object,
// including case aliases (project_id/PROJECT_ID), alternate Unicode escapes,
// and aliases appearing before or after their canonical spelling. Unknown
// names never reach the case-insensitive Go struct decoder.
// Exact duplicate decoded names are rejected earlier by scanJSONValue.
func requireExactFields(obj map[string]json.RawMessage, location string, allowed ...string) error {
	names := make(map[string]bool, len(allowed))
	for _, name := range allowed {
		names[name] = true
	}
	for key := range obj {
		if !names[key] {
			return fmt.Errorf("%s: undeclared or case-aliased JSON field %q", location, key)
		}
	}
	return nil
}

func mustField(obj map[string]json.RawMessage, key string) error {
	value, found := obj[key]
	if !found {
		return fmt.Errorf("missing field %q", key)
	}
	if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return fmt.Errorf("null field %q", key)
	}
	return nil
}
func mustString(raw json.RawMessage) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	return nil
}
