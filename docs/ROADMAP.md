# Roadmap (planned work is not implemented authorization)

## Shipped public development baseline

- **v0.1 Phase A preview:** bounded strict local graph validation, canonical JSON/SHA-256, reconstruction, inert SVG/HTML export.
- **v0.2 merged on main:** deterministic, read-only comparison of two strictly validated local graphs; syntactic node-ID equality only.

## Bounded increment implemented: v0.3 (development preview, merged to `main`)

- Deterministic, read-only statistics for **one** validated local graph: nodes and edges; forest roots/leaves, hierarchy depth and branching distributions; weak components considering hierarchy and typed edges; explicit edge kinds; exact average branching fraction. No inferred hidden edges or authority.
- Verified via local reference-generated forests, CLI negatives, fuzz tests and deterministic packaging per PR #2's review report; hosted pre-merge and post-merge Go CI succeeded. This is not authority work. See [promotion reconciliation](V0.3_PROMOTION_RECONCILIATION.md) for the evidence limits and outstanding governance follow-up.

## Later independent development proposals

1. More useful read-only graph inspection, focused error diagnostics, and source-local ingestion adapters with non-destructive source preservation. Each new untrusted-file boundary requires its own review.
2. Optional, bounded local persistence and historical query only after explicit storage/provenance contracts; storage does not grant authority.
3. Independently ratify the **specification** of issuer-qualified principal identity, protected policy publication epochs, trusted lineage issuance/revocation, and the unresolved negative-disposition precedence oracle.
4. Only after separate human authorization and independent audit: consider a production composite authorization evaluator. Existing historical proofs do not inherit permission.
5. Separately gated work: typed scoped agent APIs, interactive HTML5/SVG UI, search/memory, discovery and multi-project isolation.

**Invariant:** `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`. No roadmap item silently promotes a synthetic fixture or computed statistic into trusted identity.
