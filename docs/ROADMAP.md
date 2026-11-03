# Roadmap (planned work is not implemented authorization)

## Shipped public development baseline

- **v0.1 Phase A preview:** bounded strict local graph validation, canonical JSON/SHA-256, reconstruction, inert SVG/HTML export.
- **v0.2 merged on main:** deterministic, read-only comparison of two strictly validated local graphs; syntactic node-ID equality only.

## Current bounded increment: v0.3 (development candidate, not published)

- Deterministic, read-only statistics for **one** validated local graph: nodes and edges; forest roots/leaves, hierarchy depth and branching distributions; weak components considering hierarchy and typed edges; explicit edge kinds; exact average branching fraction. No inferred hidden edges or authority.
- Verify with independent-reference generated forests, CLI negatives, fuzz tests and deterministic packaging. Keep this distinct from any future authority work.

## Later independent development proposals

1. More useful read-only graph inspection, focused error diagnostics, and source-local ingestion adapters with non-destructive source preservation. Each new untrusted-file boundary requires its own review.
2. Optional, bounded local persistence and historical query only after explicit storage/provenance contracts; storage does not grant authority.
3. Independently ratify the **specification** of issuer-qualified principal identity, protected policy publication epochs, trusted lineage issuance/revocation, and the unresolved negative-disposition precedence oracle.
4. Only after separate human authorization and independent audit: consider a production composite authorization evaluator. Existing historical proofs do not inherit permission.
5. Separately gated work: typed scoped agent APIs, interactive HTML5/SVG UI, search/memory, discovery and multi-project isolation.

**Invariant:** `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`. No roadmap item silently promotes a synthetic fixture or computed statistic into trusted identity.
