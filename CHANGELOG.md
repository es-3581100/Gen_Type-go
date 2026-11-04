# Changelog

## 0.3.0 development preview — merged to `main` on 2026-11-03 (not a tagged release)

- Added read-only `gen-type-go stats [--summary] INPUT.json` using the existing strict Phase A graph decoder and canonical digest.
- Reports node/explicit-edge/hierarchy-edge/root/leaf counts, depth and branching histograms, exact rational mean branching, weak component sizes, explicit edge-kind counts, isolated nodes, and explicit self-loops.
- Defines weak connectivity precisely as ignoring direction across **both** hierarchical parent-child relationships and explicit typed edges, without treating this as source or identity continuity.
- Added independent-reference randomized valid-forest tests, full CLI error-path tests, a bounded valid-forest fuzz target, and a multi-root synthetic example.
- Keeps the historical authority proof inert. No network, persistence, policy publication, or privileged action is added.
- PR #2 at feature commit `df687011fc39` passed hosted pre-merge CI; merge commit `0d00e00b6522` passed post-merge CI. No formal GitHub review was recorded. An explicit out-of-band owner authorization at 2026-11-03T20:11:07Z was later located and recorded in the [PR #2 authorization addendum](https://github.com/es-3581100/Gen_Type-go/pull/2#issuecomment-6615318149); at the time of v0.3 promotion, `main` had no active branch protection. Subsequently, ruleset `26499653` was activated and independently verified, and [issue #4](https://github.com/es-3581100/Gen_Type-go/issues/4) was closed as completed on 2026-11-04 UTC. That later protection did not retroactively govern PR #2. See [promotion reconciliation](docs/V0.3_PROMOTION_RECONCILIATION.md).

## 0.2.0 — merged to `main`, independently reviewed

- Added bounded, deterministic `gen-type-go diff BEFORE AFTER` JSON comparison of valid local graphs.
- Reports canonical input digests, node additions/removals/field-name changes, reparenting, ordered-child lists, edge differences, and project/source metadata.
- Added optional `diff --summary` text counts for quick terminal inspection.
- Added synthetic changed-graph fixture and adversarial, large-graph, CLI and fuzz tests.
- Kept source-provided node IDs and digest claims explicitly untrusted; no authority, admission, protected API or persistence added.

## 0.1.0 Phase A preview — first public source publication

- Read-only strict graph validation and canonical SHA-256, reconstruction, static SVG/HTML exports.
- Historical authorization proof remains inert and separately gated.
