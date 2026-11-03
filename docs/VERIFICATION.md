# Executed local verification — initial-publication candidate

This is the *builder's local execution evidence*. It is not a signed external audit, official GitHub CI run, authority ratification, or production deployment.

| Command / check | Status | Scope / caveat |
| --- | --- | --- |
| `go version` | PASS | Go 1.23.2, Linux/amd64 |
| `go mod verify` | PASS | Standard-library module; no third-party Go deps |
| `gofmt -l cmd internal` | PASS | One preexisting Repair 004 test file needed formatting-only normalization |
| `go test -count=1 ./...` | PASS | Four packages, inherited proof tests and new CLI/render tests |
| `go vet ./...` | PASS | No findings |
| `go build ./...` | PASS | Local build |
| `go test -race -count=1 ./...` | PASS (new MIT candidate rerun) | Four packages passed, including inherited `internal/phasea` (40.179s); previous bounded execution timeout is preserved in historical evidence, not promoted to a pass |
| Bounded race suites | PASS | Public CLI, legacy proof CLI, static renderer, selected Phase A error-boundary tests |
| `go test -fuzz=FuzzStrictGraphNoPanic ... -fuzztime 2s` | PASS | 4 seed cases + 6 recorded fuzz executions; small smoke only |
| CLI `validate`, `canonical`, `sha256`, `verify`, `roundtrip`, `svg`, `html` | PASS | Synthetic tiny graph; canonical SHA `d74c08bdf9b5b2a0264ee24238bc5e318adbe2da143fc3c99ed257438e24d21b` |
| Malformed cyclic graph rejection | PASS | Nonzero exit, no data written to stdout |
| Linux/386, macOS/arm64, Windows/amd64 | COMPILE PASS | No execution on other platforms |
| Exact Draft 004 read-only validator + seven-case replay | PASS | 31/31 ZIP member hashes; replay 7/7 matches records; fixture checks do not establish authorization dispositions |
| CV-44 with DENY/BLOCKED/UNRESOLVED | GAP REPRODUCED | All three **fixture labels** accepted, confirming missing independent negative-classification oracle |
| Source archive fresh extraction and rebuild | See external checkpoint | Verification must be run on the final frozen ZIP bytes; look up exact hash in external checkpoint |

The original Draft 004 validator source has SHA-256 `085f62feede5de67f04ebfda142f26f6a0c9802288eb0b58ccfdd28b9eaad922`. Python runtime is 3.13.5; optional JSON Schema validator is `jsonschema` 4.26.0. The exact replay command executed was `python3 verify_exact_artifacts.py gen-type-go-unified-authority-rfc-draft-004-2026-11-02.zip gen-type-go-draft004-targeted-lineage-evidence-2026-11-02.zip` using a temporary extraction and unchanged source. These evidence scripts and byte-level original source archives are external support assets, not present in the public source tree.

**Full inherited race suite completed successfully in the MIT publication gate run** (exit 0, all four packages). Earlier timeout remains a separate historical result; neither outcome supplies an independent third-party audit. Reviewers should rerun the full test suite against the exact final source ZIP.
