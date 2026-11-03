# Changelog

## 0.2.0 development candidate — unreleased

- Added bounded, deterministic `gen-type-go diff BEFORE AFTER` JSON comparison of valid local graphs.
- Reports canonical input digests, node additions/removals/field-name changes, reparenting, ordered-child lists, edge differences, and project/source metadata.
- Added optional `diff --summary` text counts for quick terminal inspection.
- Added synthetic changed-graph fixture and adversarial, large-graph, CLI and fuzz tests.
- Kept source-provided node IDs and digest claims explicitly untrusted; no authority, admission, protected API or persistence added.

## 0.1.0 Phase A preview — first public source publication

- Read-only strict graph validation and canonical SHA-256, reconstruction, static SVG/HTML exports.
- Historical authorization proof remains inert and separately gated.
