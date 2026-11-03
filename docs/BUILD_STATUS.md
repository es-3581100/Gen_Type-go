# Gen Type-Go — build status

**v0.3 merged development preview** — bounded, non-authoritative graph statistics in `main` since 2026-11-03. Hosted PR and post-merge CI passed; this is not a tagged release or authority ratification. See [promotion reconciliation](V0.3_PROMOTION_RECONCILIATION.md).

## Published baseline

- `v0.1`: experimental public Phase A preview of a standard-library Go module, bounded strict JSON parsing, canonical graph validation and hashing, round-trip reconstruction, inert SVG/HTML export.
- `v0.2`: deterministic, non-authoritative graph diff; independently reviewed and merged to `main` in merge commit `8bac78ffac56ac2f7aa831bab17996b8afb52a68`, Git tree `f8b9a2780a6bd671c72309491d45d9fa43bd0296`, hosted CI passed.

## v0.3 implemented feature (merged)

`stats [--summary] INPUT.json` reuses the same strict graph parser and canonical encoding and provides structural counts, histograms, exact branching ratios, and weak connected-component sizes. See `GRAPH_STATS_CONTRACT.md` for exact semantics. New tests exercise valid and invalid graphs, reference forest statistics, fuzzing, deterministic output and writer errors. The historical isolated source candidate and local verification record predate GitHub publication. The feature was committed as `df687011fc39ff8898babe6b7ae29c89971ba0a6`, reviewed in the scope described by PR #2, and merged as `0d00e00b65224253e8e733dce3c3ee9853cb5086`. Hosted feature-branch, PR, and post-merge workflow runs all passed. No formal GitHub review submission or PR-thread merge authorization was recorded; approval provenance needs owner reconciliation.

## Historical experimental code

`internal/phasea/authz.go` remains historical in-memory proof code; the public CLI does not call it. There is no authenticated issuer, trusted cross-import lineage, production authorization service, approval of structural revisions, policy publisher, or privileged agent API.

## Not implemented

Live wiki/project ingestion adapters, verified cross-import identities, editors, persistent project isolation/history, authenticated APIs, trusted policy activation, root recovery, semantic search, web discovery, and production UI.

**Protected authority gate (unchanged):** `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`. Published or tested source is not an authorization grant.
