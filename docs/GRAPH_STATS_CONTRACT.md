# Graph statistics contract — v0.3 development

**Status:** locally implemented, not independently reviewed, not a published/ratified authority feature.

The `gen-type-go stats [--summary] INPUT.json` command produces deterministic, read-only measurements of **one explicitly supplied, fully validated Phase A graph**. It reuses `phasea.EncodeCanonical`; it does not reimplement a parser, accept partial graphs, infer hidden edges, or assign identity continuity across imports. The schema is `gen-type-go/graph-stats-v0`.

## Input and errors

- Input is a local file or `-` (standard input), bounded by `phasea.MaxInputBytes` (2 MiB). Existing strict `phasea.DecodeStrict` and `phasea.Validate` behavior applies unchanged: duplicate/unknown JSON keys, invalid types, invalid IDs, duplicate node IDs, invalid parent references, invalid sibling ordinals, missing roots, cycles, noncontiguous ordinals, duplicate/invalid typed edges, missing explicit arrays, and record-budget violations all fail closed.
- One argument is required. `--summary` selects text output; otherwise stdout is compact deterministic JSON with a final LF. `--sha256` is not a valid stats flag.
- The CLI returns code `0` on success or code `2` for argument, input-validation, or output failures. Invalid inputs do not produce a partial success report. The CLI never mutates the input.
- JSON output is capped at 4 MiB before writing; a short write or writer error produces exit code `2`.

## Metrics and meanings

All counts are unsigned in meaning but encoded as Go `int` / JSON integer values. The validated schema requires at least one node, so ratios have a nonzero denominator.

| JSON field | Definition |
|---|---|
| `canonical_sha256` | SHA-256 of the existing canonical JSON encoding (with terminating LF), **not** of original file bytes |
| `node_count` | Number of validated nodes |
| `explicit_edge_count` | Number of typed edges in the input `edges` array |
| `hierarchy_edge_count` | Number of parent-child relationships, exactly `node_count - root_count` |
| `root_count` | Nodes whose `parent_id` is empty |
| `leaf_count` | Nodes with zero hierarchical children, including isolated roots |
| `max_depth` | Greatest hierarchical distance from any root (roots are depth 0) |
| `depth_distribution` | Ascending `{value: depth, count: nodes}` buckets, including zero |
| `max_branching_factor` | Maximum number of direct hierarchical children of a node |
| `branching_factor_distribution` | Ascending `{value: child_count, count: nodes}` buckets, including zero |
| `mean_branching_factor` | Exact fraction `{numerator: hierarchy_edge_count, denominator: node_count}` with no floating point rounding |
| `weak_component_count` | Connected component count when parent-child relationships **and** typed edges are treated as undirected links |
| `weak_component_sizes` | Component sizes in descending numeric order (ties indistinguishable); all nodes included |
| `explicit_edge_kinds` | Lexically sorted `{kind, count}` buckets, counting only explicit typed edges |
| `isolated_node_count` | Nodes with no parent, no children, and **no incident explicit edges** |
| `explicit_self_loop_count` | Explicit typed edges whose `from` and `to` are equal, if permitted by the validated schema |
| `identity_boundary` | Constant statement identifying unverified source claims and no trusted lineage/permissions |

**Important distinction:** the hierarchy is a forest, possibly with multiple roots. Weak components are computed from the union of its parent-child links **and** explicit typed edges. Thus a cross-root `depends_on` edge can connect two otherwise separate trees for graph-theoretic statistics. It does not create a trusted parent relationship, merge identities, or grant permissions. `root_count` and `weak_component_count` can differ legitimately. A self-loop counts as an explicit edge but does not increase connectivity.

## Determinism and output shape

- Statistics are computed after full validation and reuse existing canonical hashing. Node and edge array permutations with the same canonical graph produce **byte-identical** statistics JSON.
- Histograms are sorted by numeric value, explicit edge-kind counts by kind string, and component sizes descending. All empty collections emit `[]`, never JSON `null`.
- Counting uses integer arithmetic. No host-specific timestamps, random IDs, external state, network access, locale-dependent formatting, or floating-point computations are used.
- The JSON report contains no node contents, titles, source references, policy labels, principals, or source-digest claims. Only the canonical digest and structural aggregates are emitted. Digest changes when content changes; it is an integrity marker, not an attestation.
- The `--summary` format includes the canonical digest and headline counts; use JSON for distributions.

## Working synthetic example

```bash
GOPROXY=off GOSUMDB=off GOWORK=off go run ./cmd/gen-type-go stats testdata/stats_forest.json
GOPROXY=off GOSUMDB=off GOWORK=off go run ./cmd/gen-type-go stats --summary testdata/stats_forest.json
```

This fixture contains seven nodes in three rooted trees, two typed edges (one cross-root link and one self-loop), four leaves, maximum depth 2, and two weak components of sizes `[6,1]`. It deliberately uses unverified source claims.

## Security boundary

`NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`

This is **local non-authoritative analysis of supplied data**, not access filtering, identity verification, protected structural admission, an authorization oracle, a trustworthy provenance store, a privileged API, or an authoritative project map. Neither cross-import ID matches nor a common digest establishes trusted continuity. No Phase 1 decisions are made.
