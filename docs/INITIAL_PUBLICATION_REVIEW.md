# Initial publication review — experimental Phase A preview

**Candidate**: `v0.1.0-phase-a-preview` (proposed pre-release identifier, not a GitHub release). **Target**: `es-3581100/Gen_Type-go`. **Remote state at preparation**: empty repository, no branches. **Local user checkout**: inaccessible to builder; publication tree prepared in an isolated workspace, no original repo modified.

| Gate | Assessment | Evidence / blocker |
| --- | --- | --- |
| Build | PASS (local) | Go 1.23 standard-library module, fresh clean extraction build; full `go test -race -count=1 ./...` later passed all four packages |
| Functionality | PASS (bounded scope) | Strict graph input, canonical output/digest, roundtrip, static SVG/HTML CLI, actual examples |
| Evidence | PASS (local verification) | Repair 004 and Draft 004 archive hashes verified; original validator and seven-case replay executed |
| Documentation | PASS (technical) | README, scope, security boundary, roadmap, reproducibility, independent handoff |
| Security boundary | PASS (inactive) / BLOCKED (authority) | Public CLI has no service/authority integration; ratification, lineage and negative decision oracle unresolved |
| GitHub publication | BLOCKED | Owner selected MIT and LICENSE is bundled; independent audit of the exact MIT source bytes and explicit commit/push approval remain outstanding; full inherited race suite passed locally |
| Target-platform behavior | NOT ESTABLISHED | Native Linux/amd64 tests performed; cross-compilation does not prove macOS/Windows runtime |

**Release decision:** Prepare a GitHub-ready source candidate and request owner review. MIT has been selected and added; public push must wait for independent review of the exact MIT tree, owner approval. Protected Phase 1 stays blocked.

**Not claimed:** existing production application, authenticated identity, security certification, policy enforcement, independent review pass of these new source bytes, source provenance attestation, or deployed services.

**MIT selection checkpoint (2026-11-02):** The project owner requested MIT, now bundled as `LICENSE`. The attribution line uses “Gen Type-Go contributors”; verify it accurately reflects the copyright holder before public release. The earlier unlicensed archive and its SHA-256 remain historical artifacts and must not be cited as verification of this new archive.
