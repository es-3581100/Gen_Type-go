// Package inspection renders bounded static, inert human-readable graph views.
// It does not make authorization decisions or create trusted lineage claims.
package inspection

import (
	"bytes"
	"fmt"
	"html"
	"strings"
	"unicode"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

const MaxRenderedNodes = 256

// textLabel drops illegal XML 1.0 control characters, normalizes whitespace,
// limits display width, then XML/HTML-escapes untrusted source text.
func textLabel(raw string, max int) string {
	var b strings.Builder
	used := 0
	spaced := false
	for _, r := range raw {
		if !(r == 9 || r == 10 || r == 13 || r >= 0x20 && r <= 0xD7FF || r >= 0xE000 && r <= 0xFFFD || r >= 0x10000 && r <= 0x10FFFF) {
			r = '�'
		}
		if unicode.IsSpace(r) {
			r = ' '
		}
		if r == ' ' {
			if spaced {
				continue
			}
			spaced = true
		} else {
			spaced = false
		}
		if used == max {
			b.WriteRune('…')
			break
		}
		b.WriteRune(r)
		used++
	}
	return html.EscapeString(b.String())
}

// SVG creates a purely static SVG tree outline. It intentionally rejects
// oversized graphs rather than silently showing an incomplete authority view.
func SVG(doc phasea.Document) ([]byte, error) {
	canonical, err := phasea.Canonicalize(doc)
	if err != nil {
		return nil, err
	}
	if len(canonical.Nodes) > MaxRenderedNodes {
		return nil, fmt.Errorf("SVG node limit exceeded: %d > %d", len(canonical.Nodes), MaxRenderedNodes)
	}
	depth := map[string]int{}
	var buf bytes.Buffer
	height := len(canonical.Nodes)*38 + 76
	fmt.Fprintf(&buf, "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 1040 %d\" role=\"img\" aria-label=\"Read-only graph hierarchy\">\n", height)
	fmt.Fprintln(&buf, "  <title>Gen Type-Go local graph inspection (not authorization)</title>")
	fmt.Fprintln(&buf, "  <rect width=\"1040\" height=\"100%\" fill=\"#fafafa\"/>")
	fmt.Fprintf(&buf, "  <text x=\"24\" y=\"29\" font-family=\"sans-serif\" font-size=\"18\">Project: %s</text>\n", textLabel(canonical.ProjectID, 90))
	fmt.Fprintln(&buf, "  <text x=\"24\" y=\"49\" font-family=\"sans-serif\" font-size=\"11\">Source claims are untrusted. Cross-import identity is unresolved.</text>")
	for i, n := range canonical.Nodes {
		d := 0
		if n.ParentID != "" {
			d = depth[n.ParentID] + 1
		}
		depth[n.ID] = d
		visualDepth := d
		if visualDepth > 15 {
			visualDepth = 15
		}
		x := 24 + visualDepth*24
		y := 72 + i*38
		if i%2 == 0 {
			fmt.Fprintf(&buf, "  <rect x=\"12\" y=\"%d\" width=\"1016\" height=\"34\" rx=\"4\" fill=\"#ebeff6\"/>\n", y-22)
		}
		fmt.Fprintf(&buf, "  <circle cx=\"%d\" cy=\"%d\" r=\"4\" fill=\"#315b86\"/>\n", x, y-4)
		fmt.Fprintf(&buf, "  <text x=\"%d\" y=\"%d\" font-family=\"sans-serif\" font-size=\"14\">%s</text>\n", x+12, y, textLabel(n.ID+" — "+n.Title, 90))
	}
	fmt.Fprint(&buf, "</svg>\n")
	return buf.Bytes(), nil
}

// HTML wraps only the previously escaped, internally constructed SVG markup.
// There is no JavaScript, external asset, fetch, server, or active UI policy.
func HTML(svg []byte) []byte {
	header := `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src 'self' data:; style-src 'unsafe-inline'">
<title>Gen Type-Go — local graph view</title></head><body>
<h1>Gen Type-Go — read-only graph</h1><p>Source claims are untrusted. This display grants no access or authority.</p>
`
	var b bytes.Buffer
	b.WriteString(header)
	b.Write(svg)
	b.WriteString("</body></html>\n")
	return b.Bytes()
}
