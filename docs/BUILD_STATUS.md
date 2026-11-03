# Build status — initial public Phase A preview

Implemented and tested: a real standard-library Go module, bounded exact-case JSON decoder, graph validation, deterministic canonical encoding and SHA-256, strict reconstruct/canonical roundtrip, safe read-only CLI commands, and bounded static SVG/HTML exports. Existing Phase A proof tests were carried forward from independently reviewed Repair 004; new CLI and inspection tests were added.

Historical experimental only: in-memory pure `Authorize` and sampled visibility diagnostics. They retain their source and test coverage but have **no live issuer, service, publisher, or admission consumer**.

Deferred: robust wiki importers, trusted cross-import lineage, graph editing, persistence/recovery, authenticated user/agent APIs, policy activation, root recovery, search/memory, web discovery, multi-project isolation, and deployable UI.

Quality update: full inherited `go test -race -count=1 ./...` PASS (four packages) for the MIT candidate; the earlier timed-out run remains separately recorded.

Known gates: MIT license selected and included; exact MIT publication candidate independent re-audit pending; RFC Draft 004 unratified; negative-disposition oracle absent; protected Phase 1 authorization withheld. This repository is a *new proposed source tree*, not an existing local checkout or GitHub commit.
