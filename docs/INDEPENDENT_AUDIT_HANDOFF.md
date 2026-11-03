# Independent audit handoff — exact initial-publication candidate

## Identity and source baseline

- Initial publication source ZIP: verify against **external** `gen-type-go-publication-checkpoint.json` and `SHA256SUMS` delivered alongside it; do not infer its SHA from this file (would be self-referential).
- Historical Go Repair 004: `5d6d3cdf31ea5ba8d2fbafc0bb288c0741407afaa2dc75e0dfcc929ec5586d7b` (96 members, 95 indexed).
- RFC Draft 004: `1a57209a9ef17c90f929d82e1bd35f1c489ffdd18302e6dba1ac5bb2d3f33ff6` (32 members, 31 indexed).
- Exact 7-case supplemental: `037d45f2159095f1e1f220fb02edc668be5d9e5a6b937ce3e68aac7fa2441877` (4 members, 3 indexed).
- Recorded execution evidence: `7b9d67d57274ebd8efbed7b2531eb513c46ff3fd26fecdb12a86e321ff3ccc18`.
- Authority-hardening comparison: `8307fdeac24cefb7d385ef9e168644f987fb514d1cd5bd22d2ea48e40093774b`, not the original canonical-contract overlay.

The public source ZIP preserves the Repair 004 Go Phase A graph implementation and pure proof tests but does not bundle large historical evidence archives. Reviewers obtain those original inputs separately by verified SHA-256. The separate historical authority implementations are deliberately not combined.

## Run independently

```sh
unzip -q gen-type-go-initial-publication-candidate.zip -d review
cd review
sha256sum -c SHA256SUMS
export GOWORK=off GOPROXY=off GOSUMDB=off
go mod verify
go test -count=1 ./...
go vet ./...
go test -race -count=1 ./...
go build ./...
go run ./cmd/gen-type-go validate testdata/tiny_graph.json
```

Read-only validator replay when both historical ZIP inputs are supplied:

```sh
python3 verify_exact_artifacts.py DRAFT004.zip SUPPLEMENTAL.zip
```

The replay verifier is shipped separately in the **external evidence bundle**, not the public module. It checks exact hashes, unpacks in a temporary directory, invokes the unchanged Draft 004 linter, reruns seven cases, and checks CV-44's three accepted labels. This is **fixture validation**, not a runtime authorization oracle.

## Attack surfaces and unresolved decisions

1. Attempt case-aliased and escaped JSON fields, duplicate IDs, cycles, invalid Unicode, over-budget inputs, and canonical hash collisions due to semantic reordering. Ensure appropriate rejection and no input mutation.
2. Challenge static SVG/HTML output using hostile node titles, XML control characters and deep hierarchy; ensure escaping and explicit renderer budgets. There is no network or browser service to exploit.
3. Confirm no public CLI command calls `Authorize`, `ProbeVisibilityGains`, or performs policy activation. Tests of historical pure functions are not authorization grants.
4. Re-run the immutable CV-44 same-input `DENY`/`BLOCKED`/`UNRESOLVED` label mutation and demonstrate the **missing negative-disposition oracle**. Do not mark this as a runtime escalation.
5. For future specification review, test valid denial plus unverifiable lineage; denial plus stale revision; incoherence plus inherited denial; conflicting lineage evidence; source revision versus policy epoch. **No ratified precedence is present.**
6. Verify the owner-selected MIT LICENSE and copyright attribution, clean checkout, GitHub CI permissions, full inherited race-suite PASS independently reproduced where possible, and actual commit state before approving public publication.

No independent sign-off of this **new publication tree** is claimed. No production authority, authenticated publisher, root recovery, or Phase 1 integration may be inferred.
