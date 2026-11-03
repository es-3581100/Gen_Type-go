# Unified Authority Contract status — unratified

Historical source: GTG-PHA-UAC-004 (Draft 004) SHA-256 `1a57209a9ef17c90f929d82e1bd35f1c489ffdd18302e6dba1ac5bb2d3f33ff6`. Draft 004 is preserved outside this public source tree as a referenced, unchanged artifact. The earlier authority-hardening candidate has SHA-256 `8307fdeac24cefb7d385ef9e168644f987fb514d1cd5bd22d2ea48e40093774b`; it is **not** the original canonical-contract overlay. The original overlay is separately identified as `ba74695a75c863f525cd0a179ea0c17565a0d7791d9d99c2ecb5fe4832f503b9` in historical review evidence, not re-hashed from a local source file here.

Current state: `EVIDENCE_INTEGRITY_PASS__REPLAY_REPORT_CONSISTENT` with `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED`.

Draft 004's fixture linter checks structure and conditional-positive consistency. It **does not** compute authoritative authorization dispositions for every negative fixture. An executed seven-case read-only replay establishes accepted/rejected *fixture checker* outcomes. In particular, the unchanged CV-44 fixture passes structural/negative checks with each proposed label `DENY`, `BLOCKED`, and `UNRESOLVED`; that is not proof that all three decisions are correct.

## Negative classification candidates (not ratified)

- **DENY:** a coherent protected view contains an applicable, authenticated, current denial.
- **BLOCKED:** the protected view is incoherent, stale, incomplete, or otherwise cannot authorize a decision at all.
- **UNRESOLVED:** evidence about identity/lineage or policy applicability is not independently verifiable.

Those are **candidate definitions only**. There is no selected precedence rule among faults that overlap. Proposed options for review include `BLOCKED > UNRESOLVED > DENY` (integrity failures dominate), `UNRESOLVED > BLOCKED > DENY` (uncertainty exposed), or an explicit *combined-fault* disposition with no precedence until a protected view can be established. The outcome must be fail-closed in every option; the external disclosure contract also remains undecided.

### Mandatory adversarial questions for a future decision oracle

| Fixture | Competing non-allow classifications | Normative status |
| --- | --- | --- |
| Coherent view, applicable valid inherited denial | DENY | Proposed single-fault reference, not ratified |
| Incoherent protected authority view | BLOCKED | Proposed single-fault reference, not ratified |
| Unverifiable grant or denial lineage | UNRESOLVED | Proposed single-fault reference, not ratified |
| Valid inherited denial plus unverifiable lineage | DENY / UNRESOLVED / BLOCKED | OPEN precedence |
| Valid inherited denial plus stale protected revision | DENY / BLOCKED | OPEN precedence |
| Incoherent view plus inherited denial | BLOCKED / DENY | OPEN precedence |
| Conflicting lineage records, one resolvable | UNRESOLVED / BLOCKED | OPEN classification |
| Current source revision vs protected policy epoch conflict | BLOCKED / UNRESOLVED | OPEN classification |

Issuer and credential validation, project membership and typed roles, mandatory actor/action ceilings, explicit grants, denial semantics, lineage proofs for inherited grants/denials, structural changes, policy publication and authenticated approval events all remain independent unimplemented gates. A node grant may not manufacture project membership or override an actor-type ceiling.

**No policy activation, no unified evaluator, no root/bootstrap implementation, no ratification.** Contract options require independent specification review and explicit owner sign-off before coding or publishing privileged behavior.
