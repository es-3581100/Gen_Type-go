# Build status — initial public Phase A preview

Implemented and tested: a real standard-library Go module, bounded exact-case JSON decoder, graph validation, deterministic canonical encoding and SHA-256, strict reconstruct/canonical roundtrip, safe read-only CLI commands, and bounded static SVG/HTML exports. Existing Phase A proof tests were carried forward from independently reviewed Repair 004; new CLI and inspection tests were added.

Historical experimental only: in-memory pure `Authorize` and sampled visibility diagnostics. They retain their source and test coverage but have **no live issuer, service, publisher, or admission consumer**.

Deferred: robust wiki importers, trusted cross-import lineage, graph editing, persistence/recovery, authenticated user/agent APIs, policy activation, root recovery, search/memory, web discovery, multi-project isolation, and deployable UI.

Quality update: full inherited `go test -race -count=1 ./...` PASS (four packages) for the MIT candidate; the earlier timed-out run remains separately recorded.

Known gates: MIT license selected and included; exact MIT publication candidate independent re-audit pending; RFC Draft 004 unratified; negative-disposition oracle absent; protected Phase 1 authorization withheld. This repository is a *new proposed source tree*, not an existing local checkout or GitHub commit.

## v0.2 development candidate (not merged)

Implemented `diff` with deterministic JSON comparison and `--summary` text counts; input validation reuses the existing Go phasea parser. Target scope: local read-only data, no credentials, policies, admission, web server, or trusted cross-import identity. Starting public branch HEAD: `dcdd9cbd4f73bcb087d72fe270a7374588627faa`. Local archive baseline Git tree: `a990afc88516d7dfd0f914fb3c4cac8706947b3d`. New working code is isolated from GitHub and needs independent review. See `DEVELOPMENT_VERIFICATION_V0.2.md` for commands, initial timeout and successful complete race rerun; nothing about this feature changes `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`.
