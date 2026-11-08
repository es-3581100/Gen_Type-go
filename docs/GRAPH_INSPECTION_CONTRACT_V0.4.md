# Gen Type-Go v0.4 — read-only inspection and diagnostics

**Development candidate only:** `feat/graph-inspection-v0.4`, originally based on v0.3 `main` commit `da9df31ca00fc9c9eb7c16838ce7f5cdf3d6872b` and subsequently reconciled against `main` merge commit `7b071c7c0ed5037b107f4ae9a263f3c06853c46f`. No PR #6 merge/release/authority promotion.

## Commands

`gen-type-go inspect --id NODE_ID INPUT.json`

`gen-type-go diagnose INPUT.json`

`INPUT.json` may be `-` for standard input. Existing 2 MiB strict local graph validation and record-budget controls apply, without changing graph encoding or hashing.

- **Inspect:** returns one node (including untrusted input content and source claims), optional parent, sibling-ordered children, ancestors root-to-parent, preorder descendants, all roots/leaves, and explicitly supplied incoming/outgoing typed edges. Self-loops are present in both edge directions. JSON includes `schema`, canonical digest and a clear no-authority identity boundary. Arrays are never null.
- **Diagnose:** uses `phasea.DecodeStrict` as its **only** acceptance oracle. Valid input returns `valid:true`, a canonical digest, an empty `diagnostics` array and exit 0. Rejected input returns `valid:false`, a **single** stable diagnostic with `code`, best-effort JSON-style `path`, `message`, `action`, and exit 2. No canonical digest is emitted on failure.
- Paths are hints, e.g. `$.nodes[1].parent_id`. When no exact location is justified, the path is `$`. No line-number or byte-offset claim is made; no partial graph is accepted.
- File I/O failure, malformed flags, unknown inspected IDs and output failures report to stderr, return exit 2, and do not produce success JSON. Malformed **readable** input with `diagnose` produces a deterministic rejection JSON even while returning exit 2.
- `inspect` reports are capped at 8 MiB and `diagnose` reports at 4 MiB, both before any stdout write. The `inspect` cap is derived from the worst valid 2 MiB input after JSON escaping, so every input the strict decoder accepts can be inspected; `diagnose` reports are bounded by construction (key text echoed in a path is truncated and escaped). Successful stdout ends with LF. Output write errors and short writes return exit 2.
- `diagnose` always terminates and, for any readable input the strict decoder rejects, emits a rejection report; excessive JSON nesting is reported as `NESTING_TOO_DEEP` and unpaired surrogate escapes as `INVALID_ESCAPE`. An object key that is not a plain identifier appears in a path as a bracket-quoted ASCII string, e.g. `$.nodes[1]["x.title[0]"]`.
- Every source ID, title, content field and digest claim remains **unverified local data**, not proof of origin, lineage, visibility, access, or permission. Do not expose private source material to unauthorized readers.

## Verification gate

Require `gofmt`, `go mod verify`, `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` and `go build ./...`, plus deterministic permutation, malformed-input, output writer and fuzz-seed regression tests. Hosted Go checks and independent review are distinct gates.

No ingestion, persistence, editing, interactive UI, authority API, or permission evaluation is in scope. Historical Phase A evaluator remains unused by the public CLI.

**Blocked authority state remains** `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`.
