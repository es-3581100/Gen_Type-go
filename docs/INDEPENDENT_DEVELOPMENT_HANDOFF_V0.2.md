# Independent audit handoff — v0.2 deterministic graph diff

**Review gate:** source-level and executable independent adversarial re-audit of this exact candidate. Do not mistake builder test evidence for independent execution. **Starting GitHub commit:** `dcdd9cbd4f73bcb087d72fe270a7374588627faa`. **Unmodified original Git tree:** `a990afc88516d7dfd0f914fb3c4cac8706947b3d`.

## What to attack

1. Compare exact source ZIP members against its internal SHA256SUMS, and reconstruct the original published baseline independently. Verify that no historical `internal/phasea` authority proof files changed.
2. Run `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...`, `go build ./...`, and `gofmt -l cmd internal` on a **fresh extraction**, with Go 1.23+ and module downloads off. The first full race attempt timed out, but a subsequent bounded full-suite rerun passed all five packages, including `internal/phasea` (~140 seconds). Independent exact-byte rerun remains a review gate.
3. Feed malformed JSON, duplicate field aliases, cyclic hierarchies, dangling references, invalid edge kinds, oversized stdin and sibling ordinal collisions to either side of `diff`. Success stdout must remain empty on invalid input; code 2 denotes failure.
4. Attack output determinism: reorder raw `nodes` and `edges` arrays while preserving valid structure, change process locale, compare repeated runs, test sibling reorder, reparenting, root-list reorder, added/removed nodes, content-only change, edge-only change and differing project/source metadata.
5. Attack source-ID reuse: the same untrusted `id` across unrelated imports must not become an authenticated lineage assertion. Confirm JSON's `identity_boundary` warning is always present and no call to `authz` or protected policy pathways is reachable from the CLI.
6. Probe output-size limits, allocation amplification, large but permitted graphs, output writer errors and short writes, one stdin operand, two stdin operands (must reject), and whitespace/escaped JSON formatting.
7. Independently read `docs/GRAPH_DIFF_CONTRACT.md` against real exported struct tags and CLI semantics. A syntactic match is not equivalent to a semantic node match.
8. Confirm that `.github/workflows/ci.yml` retains read-only permissions and the full Phase A race suite; this change adds smoke checks but must not weaken previous CI checks.

## Reproduction

```sh
sha256sum /path/to/gen-type-go-development-candidate-v0.2.zip
unzip -q /path/to/gen-type-go-development-candidate-v0.2.zip -d /tmp/gtg-diff-review
cd /tmp/gtg-diff-review
sha256sum -c SHA256SUMS
GOPROXY=off GOSUMDB=off GOWORK=off GOTOOLCHAIN=local go test -count=1 ./...
GOPROXY=off GOSUMDB=off GOWORK=off GOTOOLCHAIN=local go test -race -count=1 ./...
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/diff_after.json
```

The candidate ZIP SHA-256, changed-file hashes, exact verification logs and baseline-diff patch are recorded **outside the source tree** in the machine checkpoint and evidence package to avoid self-referential hashes. No remote feature branch, PR, commit, or merge has been created.

## Protected exclusions

No authenticated principals, policy-packet publication, root administration, live authorization, database admission, privileged agent API, or trusted imported lineage. Preserve `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`. Any normative decision among negative-disposition outcomes requires a separate RFC ratification and human authorization; this feature neither chooses nor implements precedence.

**Suggested independent dispositions:** `V0.2_FEATURE_INDEPENDENT_PASS` (only after exact-byte execution and adversarial inspection), `V0.2_REPAIR_REQUIRED`, or `V0.2_INCONCLUSIVE`. Do not imply Phase 1 approval from a diff audit.
