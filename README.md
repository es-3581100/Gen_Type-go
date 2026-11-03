# Gen Type-Go

**Experimental v0.3.0 development candidate — read-only local-first graph workbench.**

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

## Measure one local graph (v0.3 development)

```sh
go run ./cmd/gen-type-go stats testdata/tiny_graph.json
go run ./cmd/gen-type-go stats --summary testdata/stats_forest.json
```

The deterministic `stats` JSON reports node/edge/root/leaf counts, hierarchical depth and branching distributions, exact rational mean branching, explicit edge-kind counts, and weak component sizes. Weak connectivity treats the supplied parent-child and typed edges as undirected *for statistics only*. A root count is not necessarily a component count. The CLI does **not** infer hidden lineage, permissions, or trusted associations. See [graph statistics contract](docs/GRAPH_STATS_CONTRACT.md).

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

The original v0.1 baseline was published with independent bounded verification. v0.2 was merged to `main` after independent review. This v0.3 increment is **not released** and requires its own exact-byte independent review. See [v0.3 development verification](docs/DEVELOPMENT_VERIFICATION_V0.3.md) for executed local checks, and [v0.2 verification](docs/DEVELOPMENT_VERIFICATION_V0.2.md) for the previous milestone. Go source has no third-party dependencies. Source-archive creation additionally uses standard-library Python 3; see [reproducibility](docs/REPRODUCIBILITY.md).

Read [architecture](docs/ARCHITECTURE.md), [build status](docs/BUILD_STATUS.md), [authority contract status](docs/AUTHORITY_CONTRACT_STATUS.md), [roadmap](docs/ROADMAP.md), [security](SECURITY.md), and [independent audit handoff](docs/INDEPENDENT_AUDIT_HANDOFF.md).

The verified v0.3 starting baseline is the v0.2 merge commit `8bac78ffac56ac2f7aa831bab17996b8afb52a68` on [es-3581100/Gen_Type-go](https://github.com/es-3581100/Gen_Type-go). The new branch `feat/graph-stats-v0.3` initially points to that commit; **v0.3 source changes are not pushed or merged**. Public development changes and protected authority decisions remain under owner control.
