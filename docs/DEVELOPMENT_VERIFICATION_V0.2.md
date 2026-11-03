# v0.2 development candidate — verification and reproduction

**Scope:** offline bounded deterministic graph diff; no API, accounts, storage, authorized decisions, policy publication, or trusted lineage. **Branch:** not pushed. **Baseline:** GitHub `main` at `dcdd9cbd4f73bcb087d72fe270a7374588627faa` (source tree `a990afc88516d7dfd0f914fb3c4cac8706947b3d`).

## Commands and local results

Environment: Go `go1.23.2 linux/amd64`; `GOPROXY=off GOSUMDB=off GOWORK=off GOTOOLCHAIN=local`. Checks were run from an isolated extraction of the published baseline plus the listed new feature files.

| Executed command | Result |
|---|---|
| `go mod verify` | **PASS** — all modules verified |
| `go test -count=1 ./...` | **PASS** — all five packages including the inherited Phase A proof |
| `go vet ./...` | **PASS** |
| `go build ./...` | **PASS** |
| `test -z "$(gofmt -l cmd internal)"` | **PASS** |
| `go test -race -count=1 ./cmd/gen-type-go ./internal/graphdiff ./internal/inspection ./cmd/phasea-proof` | **PASS** — four packages, covering the new diff engine and CLI |
| `go test -race -count=1 -timeout=70s -run 'TestRepair\|TestStrict\|TestCanonical\|TestBad\|TestMalformed\|TestAuthz\|TestCross\|TestZero\|TestRoundTrip\|TestInvalid' ./internal/phasea` | **PASS** — selected inherited tests; ~70 s |
| `go test -race -count=1 -timeout=535s ./...` | **PASS** on bounded rerun: all five packages; `internal/phasea` 139.755 s, exit 0 |
| `go test ./internal/graphdiff -run '^$' -fuzz '^FuzzGeneratedValidGraphDiff$' -fuzztime=4s -parallel=1` | **PASS** — 1,450 executions / 4 seconds in local run |
| `go test ./internal/graphdiff -run '^$' -fuzz '^FuzzCompare$' -fuzztime=4s -parallel=1` | **PASS** — 4 executions (malformed raw inputs rarely stay valid); **not broad fuzz coverage** |

Fuzz execution counts depend on hardware and runtime; they are not invariant requirements. Initial `go test -race -count=1 ./...` exceeded the sandbox execution limit while `internal/phasea` was running; it was **not** treated as a pass. The same complete suite was rerun with an explicit 535-second test timeout (540-second external timeout), and **all five packages passed with exit 0**, without excluding inherited cases. This was an actual local rerun, not a transplanted historical pass.

## Demonstrated CLI behavior

```sh
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/diff_after.json
go run ./cmd/gen-type-go diff --summary testdata/tiny_graph.json testdata/diff_after.json
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/tiny_graph.json
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/ambiguous_reused_node_id.json
# Must fail before writing JSON success:
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/invalid_cycle.json
```

Verified JSON output SHA-256 for the example: `6df7d50a9e240a5c4b7c8bd11437bb6ab840a1ff86f3b7a78420f5514952bc97`. Repeating the command produced byte-identical JSON. Expected canonical input hashes:

- `testdata/tiny_graph.json`: `d74c08bdf9b5b2a0264ee24238bc5e318adbe2da143fc3c99ed257438e24d21b`
- `testdata/diff_after.json`: `38f762581680a0398cc5f6501be30c4e550c91c77be99026867b6e5d810f916b`

Example behavior: one added node; one content-changed syntactically matched node; one changed ordered-child list; two added edges, one removed edge. The invalid-cycle command returned an error and an empty stdout file. The new code's tests also exercise malformed/duplicate/dangling/cyclic graphs, reparenting, sibling reordering, output-size bounds, 4,001-node valid graphs, input immutability, short output writes, and cross-import ID collisions.

## Reproducing from exact candidate ZIP

```sh
mkdir -p /tmp/gtg-v02-check
cd /tmp/gtg-v02-check
unzip /path/to/gen-type-go-development-candidate-v0.2.zip
sha256sum -c SHA256SUMS
export GOPROXY=off GOSUMDB=off GOWORK=off GOTOOLCHAIN=local
go version
go mod verify
go test -count=1 ./...
go vet ./...
go build ./...
go test -race -count=1 ./...
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/diff_after.json
```

The full race suite passed locally on a bounded rerun; the earlier timeout is preserved in the external evidence logs for auditability. The proposed CI retains the full inherited race gate; it has not been weakened. Independent exact-byte re-execution remains required before merging.

Cross-build **compile-only** checks of the new CLI succeeded for `linux/arm64`, `darwin/arm64`, and `windows/amd64`. These do **not** prove runtime correctness on those platforms.

## Unresolved security gate

No compiled feature here evaluates, issues, signs or enforces authorization. The historical negative-disposition oracle and Draft 004 remain unratified. `DENY`, `BLOCKED`, and `UNRESOLVED` precedence is **not** selected or implemented. Matching two `node.id` strings is always a syntactic comparison and never proof of node lineage or access rights. No GitHub mutation was authorized by this v0.2 development mission.
