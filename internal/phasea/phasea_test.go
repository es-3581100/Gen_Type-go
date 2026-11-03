package phasea

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

const shaClaim = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func fixture() Document {
	return Document{Schema: Schema, ProjectID: "project-1", SourceID: "source-1", Nodes: []Node{
		{ID: "child", ParentID: "root", Order: 0, Title: "Child", Content: "<svg>こんにちは</svg>", SourceRef: "docs/child.md", SourceDigestClaim: shaClaim},
		{ID: "root", ParentID: "", Order: 0, Title: "Root", Content: "root", SourceRef: "docs/root.md", SourceDigestClaim: shaClaim},
		{ID: "peer", ParentID: "root", Order: 1, Title: "Peer", Content: "peer", SourceRef: "docs/peer.md", SourceDigestClaim: shaClaim},
	}, Edges: []Edge{{From: "peer", To: "child", Kind: "related_to"}, {From: "root", To: "child", Kind: "links_to"}}}
}

func TestCanonicalBytesStableAndRoundTrip(t *testing.T) {
	d := fixture()
	original, _ := json.Marshal(d)
	wire1, hash1, err := EncodeCanonical(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(wire1, []byte("\n")) {
		t.Fatal("missing LF")
	}
	d.Nodes[0], d.Nodes[2] = d.Nodes[2], d.Nodes[0]
	d.Edges[0], d.Edges[1] = d.Edges[1], d.Edges[0]
	wire2, hash2, err := EncodeCanonical(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(wire1, wire2) || hash1 != hash2 {
		t.Fatalf("permutation changed canonical result")
	}
	canonical, err := ReconstructCanonical(wire1)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := EqualGraphs(canonical, fixture())
	if err != nil || !ok {
		t.Fatalf("round trip mismatch: %v", err)
	}
	copyD := fixture()
	after, _ := json.Marshal(copyD)
	if !bytes.Equal(original, after) {
		t.Fatal("unexpected fixture mutation")
	}
	sha := sha256.Sum256(wire1)
	if fmt.Sprintf("%x", sha[:]) != hash1 {
		t.Fatal("digest wrong")
	}
}

func TestDecodeStrictRoundTrip(t *testing.T) {
	wire, _, _ := EncodeCanonical(fixture())
	d, err := DecodeStrict(wire)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Nodes) != 3 {
		t.Fatal(len(d.Nodes))
	}
}

func TestStrictRejectsMalformedInputs(t *testing.T) {
	valid, _, _ := EncodeCanonical(fixture())
	tests := map[string][]byte{
		"duplicate top key":       bytes.Replace(valid, []byte(`"project_id":"project-1",`), []byte(`"project_id":"p","project_id":"project-1",`), 1),
		"duplicate nested key":    bytes.Replace(valid, []byte(`"id":"root",`), []byte(`"id":"root","id":"root",`), 1),
		"duplicate escaped key":   bytes.Replace(valid, []byte(`"project_id":"project-1",`), []byte(`"project_id":"p","project_\u0069d":"project-1",`), 1),
		"unknown top field":       bytes.Replace(valid, []byte(`"schema":`), []byte(`"bonus":true,"schema":`), 1),
		"unknown nested field":    bytes.Replace(valid, []byte(`"title":"Root",`), []byte(`"hidden_role":"root","title":"Root",`), 1),
		"unpaired high surrogate": bytes.Replace(valid, []byte(`"title":"Root"`), []byte(`"title":"\uD800"`), 1),
		"unpaired low surrogate":  bytes.Replace(valid, []byte(`"title":"Root"`), []byte(`"title":"\uDE00"`), 1),
		"bad unicode pair":        bytes.Replace(valid, []byte(`"title":"Root"`), []byte(`"title":"\uD800\u0041"`), 1),
		"trailing document":       append(append([]byte{}, valid...), []byte(" {}")...),
		"null edges":              bytes.Replace(valid, []byte(`"edges":[`), []byte(`"edges":null,"extra":[`), 1),
		"null title":              bytes.Replace(valid, []byte(`"title":"Root"`), []byte(`"title":null`), 1),
		"null parent":             bytes.Replace(valid, []byte(`"parent_id":""`), []byte(`"parent_id":null`), 1),
		"null order":              bytes.Replace(valid, []byte(`"order":0`), []byte(`"order":null`), 1),
		"missing content":         bytes.Replace(valid, []byte(`,"content":"root"`), []byte(``), 1),
		"float ordinal":           bytes.Replace(valid, []byte(`"order":0`), []byte(`"order":0.5`), 1),
		"invalid UTF-8":           append(append([]byte{}, valid...), 0xff),
		"top level array":         []byte(`[]`),
		"exceeds byte limit":      bytes.Repeat([]byte(" "), MaxInputBytes+1),
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeStrict(raw); err == nil {
				t.Fatalf("accepted invalid JSON %q", raw[:min(len(raw), 120)])
			}
		})
	}
}

func TestStrictAcceptsValidSurrogatePair(t *testing.T) {
	wire, _, _ := EncodeCanonical(fixture())
	raw := bytes.Replace(wire, []byte(`"title":"Root"`), []byte(`"title":"\uD83D\uDE80"`), 1)
	if _, err := DecodeStrict(raw); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedGraphFailsClosed(t *testing.T) {
	tests := map[string]func(*Document){
		"duplicate id":              func(d *Document) { d.Nodes[1].ID = "child" },
		"dangling parent":           func(d *Document) { d.Nodes[0].ParentID = "unknown" },
		"self cycle":                func(d *Document) { d.Nodes[0].ParentID = "child" },
		"two node cycle":            func(d *Document) { d.Nodes[0].ParentID = "peer"; d.Nodes[2].ParentID = "child" },
		"noncontiguous order":       func(d *Document) { d.Nodes[2].Order = 8 },
		"sibling ordinal collision": func(d *Document) { d.Nodes[2].Order = 0 },
		"edge target missing":       func(d *Document) { d.Edges[0].To = "lost" },
		"duplicate edge":            func(d *Document) { d.Edges[0] = d.Edges[1] },
		"unknown edge kind":         func(d *Document) { d.Edges[0].Kind = "authority" },
		"bad source claim":          func(d *Document) { d.Nodes[0].SourceDigestClaim = "verified" },
		"invalid project":           func(d *Document) { d.ProjectID = "../evil project" },
		"nil edges":                 func(d *Document) { d.Edges = nil },
		"missing root":              func(d *Document) { d.Nodes[1].ParentID = "child" },
		"huge node count":           func(d *Document) { d.Nodes = make([]Node, MaxNodes+1) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			d := fixture()
			mutate(&d)
			if err := Validate(d); err == nil {
				t.Fatal("invalid graph accepted")
			}
		})
	}
}

func TestCanonicalWireRejectsNoncanonicalForms(t *testing.T) {
	wire, _, _ := EncodeCanonical(fixture())
	if _, err := ReconstructCanonical(bytes.TrimSuffix(wire, []byte("\n"))); err == nil {
		t.Fatal("missing LF accepted")
	}
	if _, err := ReconstructCanonical([]byte(" " + string(wire))); err == nil {
		t.Fatal("whitespace accepted")
	}
	d := fixture()
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = ReconstructCanonical(append(b, '\n')); err == nil {
		t.Fatal("unordered representation accepted")
	}
}

func TestCrossImportCannotSelfCertify(t *testing.T) {
	a, b := fixture(), fixture()
	b.SourceID = "source-2"
	b.Nodes[0].Title = "renamed"
	b.Nodes[0].SourceDigestClaim = a.Nodes[0].SourceDigestClaim
	r, err := AssessCrossImport(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if r.Admission != "blocked" || r.Identity != "unresolved_cross_import_identity" {
		t.Fatalf("unexpected approval: %+v", r)
	}
	if r.BeforeSHA256 == r.AfterSHA256 {
		t.Fatal("did not detect change")
	}
}

func TestRandomPermutationInvariance(t *testing.T) {
	baseline := fixture()
	wire, digest, _ := EncodeCanonical(baseline)
	for i := 0; i < 200; i++ {
		r := rand.New(rand.NewSource(int64(i)))
		d := fixture()
		r.Shuffle(len(d.Nodes), func(i, j int) { d.Nodes[i], d.Nodes[j] = d.Nodes[j], d.Nodes[i] })
		r.Shuffle(len(d.Edges), func(i, j int) { d.Edges[i], d.Edges[j] = d.Edges[j], d.Edges[i] })
		got, h, err := EncodeCanonical(d)
		if err != nil || h != digest || !bytes.Equal(wire, got) {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
}

func TestBadSourceClaimNeverPromoted(t *testing.T) {
	d := fixture()
	d.Nodes[0].SourceDigestClaim = strings.Repeat("b", 64)
	if _, _, err := EncodeCanonical(d); err != nil {
		t.Fatal(err)
	}
	// This demonstrates ONLY that claims are preserved; no method here asserts
	// authorship, verifies original source bytes, or admits a revision.
}

func FuzzStrictGraphNoPanic(f *testing.F) {
	wire, _, _ := EncodeCanonical(fixture())
	f.Add(wire)
	f.Add([]byte("{}"))
	f.Add([]byte(`{"x":"\uD800"}`))
	f.Add([]byte{0xff})
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > MaxInputBytes {
			t.Skip()
		}
		d, err := DecodeStrict(b)
		if err == nil {
			out, _, err := EncodeCanonical(d)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ReconstructCanonical(out); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestRoundTripPreservesSemanticFields(t *testing.T) {
	d := fixture()
	wire, _, _ := EncodeCanonical(d)
	got, err := ReconstructCanonical(wire)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := Canonicalize(d)
	if !reflect.DeepEqual(c, got) {
		t.Fatalf("expected %#v got %#v", c, got)
	}
}
