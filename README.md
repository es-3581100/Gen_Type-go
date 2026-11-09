# Gen Type-Go

**Experimental v0.3.0 development preview — merged to `main` on 2026-11-03; read-only local-first graph workbench, not a tagged release or production authority service.**

Gen Type-Go currently runs as a **Go command-line tool** for validating bounded hierarchical JSON, producing deterministic canonical representations and SHA-256 digests, checking round trips, exporting inert SVG/HTML graph outlines, **deterministically diffing two local graphs**, and **computing deterministic statistics for one local graph**. The included graph input and source digests are *untrusted data*, not identity or provenance attestations. No server, account, permission issuer, persistence layer, or automatic importer is active.

**Build (Go 1.23+, standard library only):**

```sh
go build -o gen-type-go ./cmd/gen-type-go
./gen-type-go help
./gen-type-go validate testdata/tiny_graph.json
./gen-type-go sha256 testdata/tiny_graph.json
./gen-type-go canonical testdata/tiny_graph.json > /tmp/gtg-canonical.json
./gen-type-go svg testdata/tiny_graph.json > /tmp/gtg-graph.svg
./gen-type-go html testdata/tiny_graph.json > /tmp/gtg-graph.html
./gen-type-go diff testdata/tiny_graph.json testdata/diff_after.json > /tmp/gtg-diff.json
./gen-type-go diff --summary testdata/tiny_graph.json testdata/diff_after.json
./gen-type-go stats testdata/stats_forest.json > /tmp/gtg-stats.json
./gen-type-go stats --summary testdata/stats_forest.json
```

**Working operations:** `help`, `version`, `validate`, `canonical`, `sha256`, `verify --sha256 HEX`, `roundtrip`, `svg`, `html`, `diff [--summary] BEFORE AFTER`, and `stats [--summary] INPUT`. Pass `-` as a path to read standard input (for `diff`, at most one input may be `-`). The CLI does not write to project files; only output redirection creates a file. SVG/HTML views are offline static documents, not permission-filtered projections.

**Not implemented:** live wiki ingestion, verified cross-import identity, protected revision/policy publication, authenticated users or agents, an API/server, database persistence or history, semantic search, web discovery, full editor, and deployment. The historical pure authorization code under `internal/phasea` is an **inert proof**, not a ratified authorization service and is never called by the public CLI.

**Authority gate:** `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`. RFC Draft 004 is unratified; the independent negative-disposition precedence oracle is missing. **Publication readiness does not constitute authority ratification.** Do not use Gen Type-Go to protect private information, enforce access, admit revisions, or grant permissions.

**License:** [MIT](LICENSE), selected and copyright attribution approved by the project owner for the initial public publication (“2026 Gen Type-Go contributors”). Licensing does not authorize the blocked security/authority features. See [publication review](docs/INITIAL_PUBLICATION_REVIEW.md).

## Compare two local graph files (v0.2 merged)

```sh
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/diff_after.json
go run ./cmd/gen-type-go diff --summary testdata/tiny_graph.json testdata/diff_after.json
```

Diff JSON is deterministic and reports node changes, ordered child lists, edge differences, metadata and canonical hashes; it is **not an authorization or trusted lineage proof**. Matching node IDs across sources remain only syntactic matches. See the [graph diff contract](docs/GRAPH_DIFF_CONTRACT.md) and [changelog](CHANGELOG.md).

## Measure one local graph (v0.3 merged preview)

```sh
go run ./cmd/gen-type-go stats testdata/tiny_graph.json
go run ./cmd/gen-type-go stats --summary testdata/stats_forest.json
```

The deterministic `stats` JSON reports node/edge/root/leaf counts, hierarchical depth and branching distributions, exact rational mean branching, explicit edge-kind counts, and weak component sizes. Weak connectivity treats the supplied parent-child and typed edges as undirected *for statistics only*. A root count is not necessarily a component count. The CLI does **not** infer hidden lineage, permissions, or trusted associations. See [graph statistics contract](docs/GRAPH_STATS_CONTRACT.md).

## v0.4 development candidate — read-only graph inspection

```sh
go run ./cmd/gen-type-go inspect --id c testdata/stats_forest.json
go run ./cmd/gen-type-go diagnose testdata/stats_forest.json
go run ./cmd/gen-type-go diagnose testdata/invalid_cycle.json  # emits a JSON report with "valid":false, exits 2
```

`inspect` reports a node's source-provided fields, parent, ordered children, ancestors and descendants, roots and leaves, and explicit incoming/outgoing relationships. `diagnose` wraps the existing strict graph decoder in stable first-error diagnostics and actionable JSON-path hints. This v0.4 work is **not yet merged or released**. Neither command authenticates source claims, grants access, or evaluates protected authority. See [v0.4 inspection contract](docs/GRAPH_INSPECTION_CONTRACT_V0.4.md).

## Try verification and reconstruction

```sh
HASH=$(./gen-type-go sha256 testdata/tiny_graph.json)
./gen-type-go verify --sha256 "$HASH" testdata/tiny_graph.json
./gen-type-go roundtrip testdata/tiny_graph.json
./gen-type-go validate testdata/invalid_cycle.json  # expected nonzero exit
```

`verify` checks the SHA-256 of **canonical JSON including final newline**, not the byte hash of the input file. Different valid input orderings can have the same canonical digest. A digest is not a signature or a lineage guarantee.

The local graph schema is `gen-type-go/phase-a-graph-v0`. JSON is strictly checked (case-sensitive keys, no duplicates/unknown/null wire fields), with a 2 MiB input budget, at most 10,000 nodes and 30,000 edges. SVG/HTML export is separately limited to 256 nodes; it refuses larger graphs instead of presenting a truncated picture. Multiple roots are currently supported, not a ratified cross-import identity model.

The original legacy `cmd/phasea-proof` CLI remains as a clearly labeled *historical compatibility proof*; it does not start services or authorize changes.

## Contributing and project status

The original v0.1 baseline was published with independent bounded verification. v0.2 was merged to `main` after independent review. v0.3 was merged to `main` on 2026-11-03 after passing feature-branch, PR, and post-merge GitHub Actions checks; this is a merged development preview, not evidence of a tagged release or formal GitHub review. PR #2 reports a bounded independent audit. A later explicit owner instruction (2026-11-03T20:11:07Z) in a separate project conversation authorized the normal merge; its out-of-band provenance is described in the [PR #2 authorization addendum](https://github.com/es-3581100/Gen_Type-go/pull/2#issuecomment-6615318149). No formal GitHub review was submitted. At the time of the v0.3 merge, `main` had no active protection. Subsequently, ruleset `26499653` was activated and independently verified as protecting `main`, and [issue #4](https://github.com/es-3581100/Gen_Type-go/issues/4) was closed as completed on 2026-11-04 UTC. The later ruleset does not retroactively protect PR #2. See the [v0.3 promotion and governance record](docs/V0.3_PROMOTION_RECONCILIATION.md). See [v0.3 development verification](docs/DEVELOPMENT_VERIFICATION_V0.3.md) for executed local checks, and [v0.2 verification](docs/DEVELOPMENT_VERIFICATION_V0.2.md) for the previous milestone. Go source has no third-party dependencies. Source-archive creation additionally uses standard-library Python 3; see [reproducibility](docs/REPRODUCIBILITY.md).

Read [architecture](docs/ARCHITECTURE.md), [build status](docs/BUILD_STATUS.md), [authority contract status](docs/AUTHORITY_CONTRACT_STATUS.md), [roadmap](docs/ROADMAP.md), [security](SECURITY.md), and [independent audit handoff](docs/INDEPENDENT_AUDIT_HANDOFF.md).

v0.3 began at v0.2 merge `8bac78ffac56ac2f7aa831bab17996b8afb52a68`, was published as feature commit `df687011fc39ff8898babe6b7ae29c89971ba0a6` in [PR #2](https://github.com/es-3581100/Gen_Type-go/pull/2), and merged to `main` as `0d00e00b65224253e8e733dce3c3ee9853cb5086`. This promotion does not grant authority permissions; the separate Phase 1 ratification gate is still blocked.
