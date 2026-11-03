# Current architecture (only implemented paths)

```text
bounded local JSON / standard input
    -> internal/phasea.DecodeStrict (strict exact wire shape, structural graph checks)
    -> internal/phasea.Canonicalize (deterministic preorder and sorted edges)
    -> canonical JSON + LF / SHA-256 / strict roundtrip
    -> cmd/gen-type-go (read-only CLI)
          -> internal/graphdiff (pure validated comparison; no trusted lineage)
          -> internal/inspection (bounded, XML-escaped static SVG/HTML)
```

`internal/phasea/authz.go` is preserved historical **non-operational pure proof code**, with no authenticated principal issuance, protected policy publication, request handler, or admission. The CLI does not call it. The older authority overlay is a separate incompatible historical implementation, not imported here.

The graph v0 schema requires project/source identifiers, explicit node and edge arrays, unique node IDs in a snapshot, valid parent references, contiguous sibling order, supported link kinds, and finite size budgets. A forest is allowed; source-provided identifiers and digest claims **do not** establish identity across imports. Rendering uses escaped text and rejects graphs exceeding 256 nodes; it is not an authorization projection.

Persistence, multi-user access, history, semantic search, discovery, HTML editing, agent JSON APIs, and revision admission remain planned only. An SVG/HTML *export* is not a live frontend/service. Do not interpret a static graphic, matching digest, or pure diagnostic as a trusted permission decision.

Version 0.2 adds `diff`: two bounded and strictly validated inputs pass through the existing canonicalization contract, and a pure `internal/graphdiff` comparison emits sorted JSON differences. Source IDs and node IDs remain untrusted strings even when equal across inputs. `--summary` renders human-readable counts using the same Report. Nothing in this path calls the historical authorization proof.

## v0.3 local graph statistics (development candidate only)

`cmd/gen-type-go stats` calls the existing bounded `readGraph`/`phasea.DecodeStrict` path, then `internal/graphstats.Analyze`. The analyzer calls `phasea.EncodeCanonical` for structural validation and canonical SHA-256, computes BFS depth and branching measurements, and uses an in-memory disjoint set for weak component sizes (parent-child plus explicitly supplied typed edges, treated undirected solely for connectivity). Sorted histograms and fixed integer ratios produce deterministic JSON without data-dependent map iteration order. Neither the analyzer nor CLI mutates its input, calls `phasea.Authorize`, or persists data. See `GRAPH_STATS_CONTRACT.md` for exact metrics and definitions.
