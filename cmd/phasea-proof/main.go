// phasea-proof validates and canonicalizes an in-memory JSON graph.
// It is not the production Gen Type-Go server or importer.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/es-3581100/Gen_Type-go/internal/phasea"
)

func run(in io.Reader, out io.Writer, errout io.Writer) int {
	// All untrusted input is bounded before reading any JSON structure.
	raw, err := io.ReadAll(io.LimitReader(in, phasea.MaxInputBytes+1))
	if err != nil {
		fmt.Fprintln(errout, "read:", err)
		return 2
	}
	doc, err := phasea.DecodeStrict(raw)
	if err != nil {
		fmt.Fprintln(errout, "invalid graph:", err)
		return 2
	}
	wire, digest, err := phasea.EncodeCanonical(doc)
	if err != nil {
		fmt.Fprintln(errout, "canonical:", err)
		return 2
	}
	if _, err := phasea.ReconstructCanonical(wire); err != nil {
		fmt.Fprintln(errout, "round trip:", err)
		return 2
	}
	// No disk writes and no network access. JSON output bears no authority.
	message := struct {
		Status        string          `json:"status"`
		Sha256        string          `json:"sha256"`
		CanonicalJSON json.RawMessage `json:"canonical_json"`
		Caveat        string          `json:"caveat"`
	}{"PASS_LOCAL", digest, bytes.TrimSpace(wire), "untrusted source claims; identity continuity and admission blocked"}
	if err := json.NewEncoder(out).Encode(message); err != nil {
		fmt.Fprintln(errout, "write:", err)
		return 2
	}
	return 0
}

func main() { os.Exit(run(os.Stdin, os.Stdout, os.Stderr)) }
