package main

import (
	"bytes"
	"os"
	"testing"
)

func TestCLIRealFixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/tiny_graph.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, errs bytes.Buffer
	code := run(bytes.NewReader(data), &out, &errs)
	if code != 0 {
		t.Fatalf("code %d: %s", code, errs.String())
	}
	if !bytes.Contains(out.Bytes(), []byte(`"status":"PASS_LOCAL"`)) {
		t.Fatalf("bad output %s", out.String())
	}
	if !bytes.Contains(out.Bytes(), []byte(`"caveat":"untrusted source claims; identity continuity and admission blocked"`)) {
		t.Fatalf("missing caveat %s", out.String())
	}
}

func TestCLIInvalidInput(t *testing.T) {
	var out, errs bytes.Buffer
	if code := run(bytes.NewReader([]byte(`{"schema":"fake"}`)), &out, &errs); code == 0 {
		t.Fatal("invalid input allowed")
	}
	if out.Len() != 0 {
		t.Fatal("partial success output")
	}
}
