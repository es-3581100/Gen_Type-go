# Current architecture (only implemented paths)

```text
bounded local JSON / standard input
    -> internal/phasea.DecodeStrict (strict exact wire shape, structural graph checks)
    -> internal/phasea.Canonicalize (deterministic preorder and sorted edges)
    -> canonical JSON + LF / SHA-256 / strict roundtrip
    -> cmd/gen-type-go (read-only CLI)
          -> internal/inspection (bounded, XML-escaped static SVG/HTML)
```

`internal/phasea/authz.go` is preserved historical **non-operational pure proof code**, with no authenticated principal issuance, protected policy publication, request handler, or admission. The CLI does not call it. The older authority overlay is a separate incompatible historical implementation, not imported here.

The graph v0 schema requires project/source identifiers, explicit node and edge arrays, unique node IDs in a snapshot, valid parent references, contiguous sibling order, supported link kinds, and finite size budgets. A forest is allowed; source-provided identifiers and digest claims **do not** establish identity across imports. Rendering uses escaped text and rejects graphs exceeding 256 nodes; it is not an authorization projection.

Persistence, multi-user access, history, semantic search, discovery, HTML editing, agent JSON APIs, and revision admission remain planned only. An SVG/HTML *export* is not a live frontend/service. Do not interpret a static graphic, matching digest, or pure diagnostic as a trusted permission decision.
