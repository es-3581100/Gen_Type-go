# v0.2 development baseline — published source identity

- Target repository: `https://github.com/es-3581100/Gen_Type-go`.
- GitHub `main` HEAD independently queried through the connected read-only repository integration: `dcdd9cbd4f73bcb087d72fe270a7374588627faa`.
- Original public source ZIP SHA-256: `139d9ee6a3e0fb178eb4a816b2380aaea8975070362139a3cd2ad49374930a97`.
- ZIP CRC: PASS; 44 members including SHA256SUMS; no integrity failure.
- **Locally reconstructed Git tree hash from exact source archive:** `a990afc88516d7dfd0f914fb3c4cac8706947b3d`, matching the published head's previously verified tree. `git init`, `git add -A`, `git write-tree` were performed only in an isolated archive-verification directory; no commit was synthesized.
- Go toolchain: Go 1.23.2 linux/amd64. Module: `github.com/es-3581100/Gen_Type-go`, Go 1.23, standard-library only.
- Baseline `go mod verify` and `go test -count=1 ./...`: PASS (four packages).
- Preferred owner's future local checkout: `~/repos/es-3581100/gen_type-go/`; it was **not accessed**, cloned, or modified here.
- The sandbox cannot resolve `github.com` for `git clone`, so the source was recovered from the **exact previously published canonical source ZIP** rather than inventing a local Git history. The development snapshot is an isolated working tree with **no local Git commit, branch, or push**.
- **Original Phase A authority proof remains unchanged.** New code under `internal/graphdiff` is read-only, non-authoritative, and not permitted to infer identity continuity.

Protected gate: `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`.
