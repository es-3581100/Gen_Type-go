# Gen Type-Go v0.3 — development verification

**Historical local execution checkpoint:** the local test matrix passed at development time. Subsequently, v0.3 was published, PR #2 was merged, and hosted pre-merge/post-merge CI passed. This document preserves original local evidence rather than presenting it as newly rerun. See [promotion reconciliation](V0.3_PROMOTION_RECONCILIATION.md) for dated hosted results, review provenance limits, and governance follow-up.

## Source and baseline

- Baseline: v0.2 merge commit `8bac78ffac56ac2f7aa831bab17996b8afb52a68`, tree `f8b9a2780a6bd671c72309491d45d9fa43bd0296`.
- Source: exact v0.2 candidate ZIP `16411b69604848ed89aa87ab2e38b666e0e7d3413b97917669e27e9a83e30c92`; 53 entries, 52/52 verified source-member hashes.
- Original checkpoint output: isolated Go source tree before the v0.3 GitHub commit and PR. Later published as feature commit `df687011fc39ff8898babe6b7ae29c89971ba0a6` and merged as `0d00e00b65224253e8e733dce3c3ee9853cb5086`.
- Target branch (created from baseline): `feat/graph-stats-v0.3`.

## Commands and evidence

From the v0.3 extracted source directory on Linux/amd64, Go 1.23.2:

```sh
export GOPROXY=off GOSUMDB=off GOWORK=off GOTOOLCHAIN=local
test -z "$(gofmt -l cmd internal)"
go mod verify
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
go build ./...
GOMAXPROCS=2 go test -run='^$' -fuzz '^FuzzAnalyzeValidForest$' -fuzztime=5s -parallel=2 ./internal/graphstats
go run ./cmd/gen-type-go stats testdata/stats_forest.json
go run ./cmd/gen-type-go stats --summary testdata/stats_forest.json
```

### Executed results (Go 1.23.2, Linux/amd64)

| Check | Result |
|---|---|
| `gofmt -l cmd internal` | PASS; no unformatted Go files |
| `go mod verify` | PASS; standard-library-only module |
| `go test -count=1 ./...` | PASS, all six packages |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `go test -race -count=1 ./...` | PASS, all six packages; inherited `internal/phasea` 40.594 s |
| Valid-forest fuzz, 5 s, two workers | PASS; 17,775 executions, 170 total interesting inputs reported |
| CLI stats repeated output | PASS, byte-identical |
| CLI stdin and shuffled valid graph | PASS, exact output identity |
| Four malformed/invalid CLI fixtures | All rejected (exit 2, no stdout) |
| CLI summary and exact example | PASS |

The background quality script recorded exit code `0` after every Go check and final execution status `0`. Full command stdout, exit statuses, and the executed smoke report are included in the separately delivered evidence package. These are **local executed results**, not independent re-audit or hosted v0.3 CI.

**Synthetic example output** is 808 UTF-8 bytes, SHA-256 `68f4b707ba945dd5c273939f950c68db0b4c792a381f2a01bbbf12d45d88b783`. The canonical input digest is independent from the raw input bytes and is listed below.

## Verifiable working behavior

- The source fixture `testdata/stats_forest.json` canonical SHA-256 is `aa41cbb8300de1eb0f705e0726903e5bbe7a87aa67d76e9b30fea859e04a2261`.
- Expected stats: 7 nodes; 2 explicit edges; 4 hierarchy edges; 3 roots; 4 leaves; 2 weak components `[6,1]`; depth and branching buckets documented in `GRAPH_STATS_CONTRACT.md`.
- The new focused CLI and `internal/graphstats` test suites passed, including under `-race`.
- 120 independently generated valid forests are tested against independent BFS component-size and direct parent-chain depth/reference branching computations; iteration counts are synthetic deterministic test cases, **not** external review.
- Historical code and authority contract documents are not modified by the new stats implementation.

## Limitations and next steps

- At the time of the original local checkpoint, hosted CI and independent re-audit had not yet been performed. The later PR reported bounded independent review, and exact pre-merge and post-merge hosted CI succeeded; formal GitHub review submissions remain absent. This record does not independently replay the external audit's archive, fuzz or Python-oracle claims.
- No authenticated lineage, authorization, revision admission, trusted imported sources, or persistence.
- Full correctness is not established by tests or fuzzing. Particularly stress deeply nested inputs, resource budgets, and graph connectivity under cross-root edges.

`NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`
