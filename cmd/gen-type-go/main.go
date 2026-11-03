// gen-type-go is a read-only local graph inspection workbench. It has no
// authority service, policy publisher, persistent storage, or network listener.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/es-3581100/Gen_Type-go/internal/inspection"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

const version = "0.1.0-phase-a-preview"
const usage = `Gen Type-Go — read-only Phase A graph workbench (no authority services)

Usage:
  gen-type-go help
  gen-type-go version
  gen-type-go validate INPUT.json
  gen-type-go canonical INPUT.json
  gen-type-go sha256 INPUT.json
  gen-type-go verify --sha256 HEX INPUT.json
  gen-type-go roundtrip INPUT.json
  gen-type-go svg INPUT.json
  gen-type-go html INPUT.json

INPUT.json may be '-' for standard input. Input is limited to 2 MiB.
All SHA-256 values refer to canonical JSON bytes (including final LF),
not to the original input file. No command mutates its input or grants authority.
The svg/html commands emit inert static views; redirect stdout to save them.
`

var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func run(args []string, in io.Reader, out, errout io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(out, usage)
		return 0
	}
	if args[0] == "version" {
		if len(args) != 1 {
			fmt.Fprintln(errout, "version accepts no arguments")
			return 2
		}
		fmt.Fprintln(out, "gen-type-go", version)
		return 0
	}
	command := args[0]
	switch command {
	case "validate", "canonical", "sha256", "verify", "roundtrip", "svg", "html":
	default:
		fmt.Fprintln(errout, "unknown command:", command)
		fmt.Fprint(errout, usage)
		return 2
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errout)
	expected := fs.String("sha256", "", "lowercase SHA-256 of canonical JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errout, "expected exactly one input path (use - for stdin)")
		return 2
	}
	if command == "verify" {
		if !digestPattern.MatchString(*expected) {
			fmt.Fprintln(errout, "verify requires --sha256 with 64 lowercase hex characters")
			return 2
		}
	} else if *expected != "" {
		fmt.Fprintln(errout, "--sha256 is only valid for verify")
		return 2
	}
	path := fs.Arg(0)
	reader := in
	if path != "-" {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(errout, "open:", err)
			return 2
		}
		defer f.Close()
		reader = f
	}
	raw, err := io.ReadAll(io.LimitReader(reader, phasea.MaxInputBytes+1))
	if err != nil {
		fmt.Fprintln(errout, "read:", err)
		return 2
	}
	doc, err := phasea.DecodeStrict(raw)
	if err != nil {
		fmt.Fprintln(errout, "invalid graph:", err)
		return 2
	}
	canonical, digest, err := phasea.EncodeCanonical(doc)
	if err != nil {
		fmt.Fprintln(errout, "canonical:", err)
		return 2
	}
	switch command {
	case "validate":
		_, err = fmt.Fprintf(out, "VALID graph: %d nodes, %d edges (untrusted provenance)\n", len(doc.Nodes), len(doc.Edges))
	case "canonical":
		_, err = out.Write(canonical)
	case "sha256":
		_, err = fmt.Fprintln(out, digest)
	case "verify":
		if *expected != digest {
			fmt.Fprintf(errout, "canonical SHA-256 mismatch: got %s\n", digest)
			return 1
		}
		_, err = fmt.Fprintln(out, "MATCH", digest)
	case "roundtrip":
		var rebuilt phasea.Document
		rebuilt, err = phasea.ReconstructCanonical(canonical)
		if err == nil {
			var round []byte
			round, _, err = phasea.EncodeCanonical(rebuilt)
			if err == nil && !bytes.Equal(round, canonical) {
				err = errors.New("reconstructed encoding drift")
			}
		}
		if err == nil {
			_, err = fmt.Fprintln(out, "ROUNDTRIP PASS", digest)
		}
	case "svg", "html":
		var svg []byte
		svg, err = inspection.SVG(doc)
		if err == nil {
			if command == "html" {
				svg = inspection.HTML(svg)
			}
			_, err = out.Write(svg)
		}
	}
	if err != nil {
		fmt.Fprintln(errout, "output:", err)
		return 2
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
