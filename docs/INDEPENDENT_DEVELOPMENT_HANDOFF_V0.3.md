# Gen Type-Go v0.3 — independent development review handoff

**Candidate status:** local graph-statistics increment, independent audit pending. **Not** a security/authority review or Phase 1 authorization.

## Provenance and source identity

- Repository: `https://github.com/es-3581100/Gen_Type-go`
- Base merge commit: `8bac78ffac56ac2f7aa831bab17996b8afb52a68`.
- Base verified Git tree: `f8b9a2780a6bd671c72309491d45d9fa43bd0296`.
- Source baseline ZIP SHA-256: `16411b69604848ed89aa87ab2e38b666e0e7d3413b97917669e27e9a83e30c92`.
- Remote branch `feat/graph-stats-v0.3` initially created from the base commit; the v0.3 code is not pushed.
- The separately delivered SHA-backed development checkpoint identifies the exact **candidate** source ZIP and tree hash. Obtain the original candidate archive, not only this Markdown file, for exact-byte independent replay.

## Scope to attack

New code:

- `internal/graphstats/stats.go` — bounded, non-mutating graph aggregation.
- `internal/graphstats/stats_test.go` — handcrafted vectors, independent-reference randomized forests, opt-in valid-graph fuzz target.
- `cmd/gen-type-go/main.go` — `stats` CLI plumbing, no new privileged execution path.
- `cmd/gen-type-go/stats_test.go` — stdout, stdin, invalid input, writer failures, injection non-disclosure.
- `testdata/stats_forest.json` — synthetic example with three hierarchy roots, two explicit typed edges (one cross-root link and one self-loop), two weak components.
- `docs/GRAPH_STATS_CONTRACT.md` — precise semantics.
- `.github/workflows/ci.yml` — offline-safe stats smoke tests in existing read-only workflow.

Source SHA-256 for new and changed Go paths (verify against candidate before judging):

```
63eb3061689575e93b464634681e82c95c92f43988d5d76423b8c94e9ac5cb54  internal/graphstats/stats.go
7e005c53f07fca0430bdb95a8a44071fae1e8d045c5b8ebd3f1ccb1d562b88a2  internal/graphstats/stats_test.go
08b690da94ac170eaf8e9eb2eafe2ffea17f10ee26e010cfa34af40d896edafb  cmd/gen-type-go/main.go
4f4ebc08b23ac62484ec6d9f836ecbe3fa749de3ea1a15fb00d2c9182cdea3f4  cmd/gen-type-go/stats_test.go
```

## Mandatory independent checks

1. Verify exact candidate ZIP SHA-256, CRC, all indexed members, path safety, and byte-identical repackaging.
2. Compare unchanged protected authority/security/instruction files against the published v0.2 baseline's exact bytes and/or GitHub blob hashes at the merge commit. A clean local diff is not authority approval.
3. Fresh-extract and run `GOPROXY=off GOSUMDB=off GOWORK=off GOTOOLCHAIN=local go mod verify`, `go test -count=1 ./...`, `go vet ./...`, `go test -race -count=1 ./...`, `go build ./...` and gofmt.
4. Run the stats CLI twice on the same graph and compare output bytes. Shuffle the node and explicit edge array orders while retaining the same valid forest and require identical output and canonical SHA.
5. Independently calculate: total nodes, explicit edges, number of parent links, roots, leaves, depth histogram, branching histogram, weak component sizes using a separate traversal implementation, and edge-kind counts. Verify both tree-only and cross-root explicit-edge examples.
6. Test invalid/over-budget inputs: duplicate node IDs, invalid schema, wrong types/duplicate JSON keys, gaps or collisions in sibling ordinals, missing parents, cycles, unsupported and duplicate edges, missing arrays, maximum-node/edge limits, >2 MiB stdin, and malformed UTF-8.
7. Test deeply nested legal hierarchies, multiple roots, isolated singleton roots, self-loops, and max-size permitted inputs. Check non-recursive stats traversal does not hide validation-layer stack or performance limits.
8. Inject malicious HTML/JSON-looking values into untrusted titles, content and source refs; no raw user string should appear in stats JSON, except through the canonical digest. Check output writer failure/short writes and no successful partial report after invalid input.
9. Exercise `FuzzAnalyzeValidForest`; develop a separate raw-input fuzz campaign and do not inflate its small corpus into a high-coverage claim.
10. Confirm no route from `stats` reaches `phasea.Authorize`, policy, identity issuance, storage, network or privileged APIs. Verify the semantics of `weak_component_count` and the exact average-branching fraction.

## Expected synthetic fixture results

`testdata/stats_forest.json` has 7 nodes, 2 explicit typed edges, 4 hierarchy edges, 3 roots, 4 leaves, depth buckets `[{0:3},{1:3},{2:1}]`, branching buckets `[{0:4},{1:2},{2:1}]`, max depth 2, max branching factor 2, mean branching factor `4/7`, component sizes `[6,1]`, one isolated singleton root and one explicit self-loop. Its canonical digest is `aa41cbb8300de1eb0f705e0726903e5bbe7a87aa67d76e9b30fea859e04a2261`.

## Security decision boundary

`NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`

This work reports only features of **supplied** graphs. It does not infer authenticated lineage, permission changes, root authority, policy effects, trusted identities, or structural admission. The negative-disposition precedence contract is not ratified. Do not merge or push the source candidate solely because this handoff exists; owner authorization remains separate.
