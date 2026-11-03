package inspection

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

func graph() phasea.Document {
	return phasea.Document{
		Schema: phasea.Schema, ProjectID: "test", SourceID: "fixture", Nodes: []phasea.Node{{
			ID: "root", ParentID: "", Order: 0, Title: "root", Content: "text", SourceRef: "source", SourceDigestClaim: strings.Repeat("a", 64),
		}}, Edges: []phasea.Edge{},
	}
}
func TestValidXMLAndEscaping(t *testing.T) {
	d := graph()
	d.Nodes[0].Title = `<script>alert("bad")</script>&` + "\x01" + "\u202e"
	svg, err := SVG(d)
	if err != nil {
		t.Fatal(err)
	}
	dec := xml.NewDecoder(bytes.NewReader(svg))
	for {
		_, err := dec.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatal(err)
		}
	}
	if bytes.Contains(svg, []byte("<script")) || !bytes.Contains(svg, []byte("&lt;script&gt;")) {
		t.Fatalf("unsafe SVG: %s", svg)
	}
	html := HTML(svg)
	if bytes.Contains(html, []byte("<script")) || !bytes.Contains(html, []byte("Content-Security-Policy")) {
		t.Fatal("unsafe HTML envelope")
	}
}
func TestDeterministicAndBudget(t *testing.T) {
	d := graph()
	a, err := SVG(d)
	if err != nil {
		t.Fatal(err)
	}
	b, err := SVG(d)
	if err != nil || !bytes.Equal(a, b) {
		t.Fatal("nondeterministic SVG")
	}
	for i := 1; i <= MaxRenderedNodes; i++ {
		d.Nodes = append(d.Nodes, phasea.Node{ID: "n" + strings.Repeat("x", i), ParentID: "root", Order: i - 1, Title: "t", SourceRef: "source", SourceDigestClaim: strings.Repeat("a", 64)})
	}
	if _, err := SVG(d); err == nil {
		t.Fatal("oversized SVG should fail")
	}
}
func TestNoAuthorityClaim(t *testing.T) {
	svg, err := SVG(graph())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(svg, []byte("not authorization")) || !bytes.Contains(svg, []byte("untrusted")) {
		t.Fatal("missing visible caveats")
	}
}
