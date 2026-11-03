package graphdiagnostics

import (
	"bytes"
	"encoding/json"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
	"strings"
	"testing"
)

const claim = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func valid() []byte {
	d := phasea.Document{Schema: phasea.Schema, ProjectID: "p", SourceID: "s", Nodes: []phasea.Node{
		{ID: "root", ParentID: "", Order: 0, Title: "Root", SourceRef: "local", SourceDigestClaim: claim},
		{ID: "child", ParentID: "root", Order: 0, Title: "Child", SourceRef: "local", SourceDigestClaim: claim},
	}, Edges: []phasea.Edge{}}
	b, _, _ := phasea.EncodeCanonical(d)
	return b
}
func TestValidDiagnostics(t *testing.T) {
	b := valid()
	r := Diagnose(b)
	if !r.Valid || r.CanonicalSHA256 == "" || len(r.Diagnostics) != 0 {
		t.Fatalf("valid %+v", r)
	}
	x, err := Encode(r)
	if err != nil || !json.Valid(x) || !bytes.HasSuffix(x, []byte("\n")) {
		t.Fatal("invalid JSON")
	}
	for i := 0; i < 8; i++ {
		y, _ := Encode(Diagnose(b))
		if !bytes.Equal(x, y) {
			t.Fatal("unstable output")
		}
	}
}
func TestInvalidDiagnostics(t *testing.T) {
	b := valid()
	cases := []struct {
		name       string
		raw        []byte
		code, path string
	}{
		{"duplicate key", bytes.Replace(b, []byte(`"schema":`), []byte(`"schema":"extra","schema":`), 1), "DUPLICATE_JSON_KEY", "$"},
		{"alias", bytes.Replace(b, []byte(`"title":"Child"`), []byte(`"TITLE":"Child"`), 1), "UNKNOWN_FIELD", "$.nodes[1].TITLE"},
		{"unknown top", bytes.Replace(b, []byte(`"schema":`), []byte(`"bonus":true,"schema":`), 1), "UNKNOWN_FIELD", "$.bonus"},
		{"missing", bytes.Replace(b, []byte(`"title":"Child",`), nil, 1), "MISSING_FIELD", "$.nodes[1].title"},
		{"null", bytes.Replace(b, []byte(`"title":"Child"`), []byte(`"title":null`), 1), "NULL_FIELD", "$.nodes[1].title"},
		{"schema", bytes.Replace(b, []byte(`"schema":"gen-type-go/phase-a-graph-v0"`), []byte(`"schema":"wrong"`), 1), "BAD_SCHEMA", "$.schema"},
		{"parent", bytes.Replace(b, []byte(`"parent_id":"root"`), []byte(`"parent_id":"lost"`), 1), "INVALID_PARENT", "$.nodes[1].parent_id"},
		{"edge kind", bytes.Replace(b, []byte(`"edges":[]`), []byte(`"edges":[{"from":"root","to":"child","kind":"grant"}]`), 1), "INVALID_EDGE_KIND", "$.edges[0].kind"},
		{"utf8", append(append([]byte{}, b...), 0xff), "INVALID_UTF8", "$"},
		{"budget", bytes.Repeat([]byte("a"), phasea.MaxInputBytes+1), "INPUT_TOO_LARGE", "$"},
		{"malformed", []byte(`{"x":`), "MALFORMED_JSON", "$"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Diagnose(tc.raw)
			if r.Valid || r.CanonicalSHA256 != "" || len(r.Diagnostics) != 1 {
				t.Fatalf("unexpected: %+v", r)
			}
			d := r.Diagnostics[0]
			if d.Code != tc.code || d.Path != tc.path || d.Action == "" {
				t.Fatalf("got %+v want code %s path %s", d, tc.code, tc.path)
			}
			out, err := Encode(r)
			if err != nil || !json.Valid(out) {
				t.Fatal(err)
			}
		})
	}
	dup := bytes.Replace(b, []byte(`"id":"child"`), []byte(`"id":"root"`), 1)
	r := Diagnose(dup)
	if r.Valid || r.Diagnostics[0].Code != "DUPLICATE_NODE" || r.Diagnostics[0].Path != "$.nodes[1].id" {
		t.Fatalf("duplicate: %+v", r)
	}
	if !strings.Contains(r.IdentityBoundary, "no_trusted") {
		t.Fatal("authority boundary lost")
	}
}
func FuzzDiagnoseDeterminism(f *testing.F) {
	f.Add(valid())
	f.Add([]byte(`{}`))
	f.Add([]byte{0xff})
	f.Add([]byte(`{"bad":`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > phasea.MaxInputBytes+1 {
			t.Skip()
		}
		a, b := Diagnose(raw), Diagnose(raw)
		x, err := Encode(a)
		if err != nil {
			t.Fatal(err)
		}
		y, err := Encode(b)
		if err != nil || !bytes.Equal(x, y) {
			t.Fatal("unstable diagnostics")
		}
		if a.Valid && len(a.Diagnostics) != 0 {
			t.Fatal("valid with error")
		}
	})
}

func TestMultipleIndependentErrorsAreDeterministic(t *testing.T) {
 raw:=bytes.Replace(valid(),[]byte(`"title":"Child"`),[]byte(`"TITLE":"Child","ZZZ":"other"`),1)
 first,_:=Encode(Diagnose(raw))
 for i:=0;i<250;i++ {
  got,_:=Encode(Diagnose(raw))
  if !bytes.Equal(first,got) {t.Fatalf("multiple malformed fields produced unstable diagnosis")}
 }
 var report Report
 if err:=json.Unmarshal(first,&report);err!=nil||report.Valid||len(report.Diagnostics)!=1||report.Diagnostics[0].Code!="UNKNOWN_FIELD" {t.Fatalf("unexpected rejection: %+v %v",report,err)}
}
