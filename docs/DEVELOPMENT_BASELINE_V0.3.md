# Historical baseline checkpoint (before v0.3 publication)

> This document intentionally preserves the pre-publication starting state. It is not a claim about current GitHub state. See [v0.3 promotion reconciliation](V0.3_PROMOTION_RECONCILIATION.md) for PR #2, merge, CI, and open governance evidence.

# Development baseline — v0.3 graph statistics

- Target repository: `https://github.com/es-3581100/Gen_Type-go`
- Published, verified starting merge commit: `8bac78ffac56ac2f7aa831bab17996b8afb52a68` (`main`).
- Exact published v0.2 Git tree: `f8b9a2780a6bd671c72309491d45d9fa43bd0296`.
- Preserved source archive: `gen-type-go-development-candidate-v0.2.zip`, SHA-256 `16411b69604848ed89aa87ab2e38b666e0e7d3413b97917669e27e9a83e30c92`.
- Verification: the source archive passed ZIP CRC and all 52 indexed SHA-256 entries; clean Go baseline tests passed (five packages).
- Verified hosted CI at merge commit: GitHub Actions run `41593905454`, conclusion `success`.
- Branch created: `feat/graph-stats-v0.3` from `8bac78ffac56ac2f7aa831bab17996b8afb52a68`.
- Implementation workspace: isolated source extraction in this agent's container; **not** the owner's intended future local checkout `~/repos/es-3581100/gen_type-go/`.
- No generated v0.3 implementation commits or pushes authorized at this checkpoint. The remote feature branch is initially the same commit as `main`.

This increment adds read-only graph statistics only. The experimental authority code is out of scope and remains blocked by `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`.
