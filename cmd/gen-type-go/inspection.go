package main

import (
	"flag"
	"fmt"
	"github.com/es-3581100/Gen_Type-go/internal/graphdiagnostics"
	"github.com/es-3581100/Gen_Type-go/internal/graphinspect"
	"github.com/es-3581100/Gen_Type-go/internal/phasea"
	"io"
	"os"
)

func runInspectCLI(args []string, in io.Reader, out, errout io.Writer) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(errout)
	id := fs.String("id", "", "local node ID")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *id == "" || fs.NArg() != 1 {
		fmt.Fprintln(errout, "inspect requires --id NODE_ID and exactly one input path")
		return 2
	}
	doc, err := readGraph(fs.Arg(0), in)
	if err != nil {
		fmt.Fprintln(errout, "invalid graph:", err)
		return 2
	}
	report, err := graphinspect.Inspect(doc, *id)
	if err != nil {
		fmt.Fprintln(errout, "inspect:", err)
		return 2
	}
	encoded, err := graphinspect.Encode(report)
	if err != nil {
		fmt.Fprintln(errout, "inspect output:", err)
		return 2
	}
	return writeInspection(encoded, out, errout)
}
func runDiagnoseCLI(args []string, in io.Reader, out, errout io.Writer) int {
	fs := flag.NewFlagSet("diagnose", flag.ContinueOnError)
	fs.SetOutput(errout)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(errout, "diagnose requires exactly one input path")
		return 2
	}
	reader := in
	if fs.Arg(0) != "-" {
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			fmt.Fprintln(errout, "open graph:", err)
			return 2
		}
		defer f.Close()
		reader = f
	}
	raw, err := io.ReadAll(io.LimitReader(reader, phasea.MaxInputBytes+1))
	if err != nil {
		fmt.Fprintln(errout, "read graph:", err)
		return 2
	}
	report := graphdiagnostics.Diagnose(raw)
	encoded, err := graphdiagnostics.Encode(report)
	if err != nil {
		fmt.Fprintln(errout, "diagnose output:", err)
		return 2
	}
	if code := writeInspection(encoded, out, errout); code != 0 {
		return code
	}
	if !report.Valid {
		return 2
	}
	return 0
}
func writeInspection(b []byte, out, errout io.Writer) int {
	n, err := out.Write(b)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	if err != nil {
		fmt.Fprintln(errout, "output:", err)
		return 2
	}
	return 0
}
