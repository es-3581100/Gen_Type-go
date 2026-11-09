# Gen Type-Go v0.4 — formal bound on inspection JSON output

**Scope:** PR #6, source baseline `67204b5fe76f14212b4e6c1d41d16fdec2ba9030` (November 8–9, 2026). This is a source-conditional mathematical argument, not permission or authority evidence. The underlying derivation and byte-exact model were prepared in the November 8 Claude audit capsules; the reasoning below is stated in terms of the repository's actual `phasea` input contract and `graphinspect` report structure.

## Claim

**Theorem (strict-decoded input).** If `raw` is accepted by `phasea.DecodeStrict(raw)`, so `len(raw) <= phasea.MaxInputBytes = 2,097,152`, and `id` names a node in the resulting document, then the JSON encoding produced by `graphinspect.Encode(graphinspect.Inspect(doc, id))`, **including its final LF**, has length at most **5,326,298 bytes**. In particular, its length is less than `graphinspect.MaxOutputBytes = 8,388,608` (8 MiB).

The real-valued inequality derived below is `O <= 5,326,298 + 8/49` bytes; because `O` is an integer, `O <= 5,326,298`. A regression uses the more conservative **ceiling, 5,326,299**, when deciding whether the configured cap covers the proof. The margin from the integer bound to the cap is 3,062,310 bytes.

**Important domain restriction:** this statement is about graphs accepted from a **2 MiB or smaller wire input**. `Inspect` also accepts a Go `phasea.Document` directly; a caller-created in-memory document may satisfy `phasea.Validate` while having a serialized size larger than 2 MiB. The theorem does not promise every such document will encode below 8 MiB. The cap and output-error path remain necessary.

## Source contracts and model

The proof uses these implementation contracts; the companion Go test `TestInspectionOutputBoundSourceContracts` checks the contracts by executable fixtures, parser/encoder measurements, and a source-level check of the supported edge-kind set.

| Contract | Value | Source |
|---|---|---|
| Maximum accepted input bytes | `2 << 20` | `internal/phasea/types.go`; enforced in `DecodeStrict` |
| Node/parent IDs | ASCII identity pattern; 1–128 bytes (except empty root parent) | `internal/phasea/graph.go` |
| Digest claim | exactly 64 lowercase hexadecimal characters | `internal/phasea/graph.go` |
| `title`, `content`, `source_ref` | at most 4,096; 262,144; 4,096 decoded bytes, respectively | `phasea.Validate` |
| `source_ref` | nonempty | `phasea.Validate` |
| Ordinal | nonnegative, at most 10,000, contiguous within siblings | `phasea.Validate` |
| Edge kinds | `links_to`, `related_to`, `depends_on`; duplicate typed triples rejected | `phasea.Validate` |
| Node wire fields | seven required exact keys, even if some values are empty | `phasea.DecodeStrict` / `validateGraphWireShape` |
| Output | fixed `Report` layout, Go `encoding/json` HTML escaping, final `\n` | `internal/graphinspect/inspect.go` |

`Inspect` puts the full selected node **N** and its full parent **P** (or `null`) in the report. Other nodes appear only as ASCII IDs in the five lists (`children`, `ancestors`, `descendants`, `roots`, `leaves`). Explicit edges appear in the two lists (`incoming_edges`, `outgoing_edges`); a self-loop on N appears in both. This model is valid for a rooted, acyclic, sibling-ordered graph because the strict decoder calls `Validate`.

Write **W** for input wire bytes, **O** for inspection output bytes (including LF), and **D** for the combined decoded byte length of the three free-text fields (`title`, `content`, `source_ref`) of N and P. From the field caps,

```
D <= 2 * (4096 + 262144 + 4096) = 540672.
```

### Lemma 1 — escaping and the two full payloads

A text byte on the valid wire costs at least one byte; Go JSON marshalling uses at most six output bytes per decoded UTF-8 byte. The amplification factor six is attained by `<`, `>`, and `&`, each represented as `\u003c`, `\u003e`, or `\u0026`. Quotes, backslashes, and JSON controls cannot increase that ratio; multibyte Unicode characters also do not exceed it. Identifiers, hexadecimal digest claims, and the permitted edge kinds are ASCII and have no HTML-escapable characters.

Only N and P have full `Node` objects in the inspection report, so the maximum potentially amplified **payload** is D. Every other node contributes only its ID, already bounded by its wire cost.

### Lemma 2 — how often IDs can appear

A node other than N or P occurs in at most **three** of the five lists. The maximum is attained by a *leaf child of N* (`children`, `descendants`, and `leaves`). Deeper descendant leaves appear in at most two; ancestors, other roots, and unrelated leaves appear in at most two. N and P each appear in at most two ID lists, separately budgeted below.

An ID of length **L <= 128** costs at most `L + 3` output bytes per list entry, allowing quotes and an array comma. A non-root node with such an ID costs **at least `166 + L` input bytes** (seven-key JSON object, a one-byte parent ID, a one-byte source reference, a 64-byte digest, plus one array delimiter). A root has the slightly smaller floor `165 + L` but occurs in at most two lists.

Thus the maximal output-list-byte / input-node-byte ratio for **non-hot** nodes is bounded by

```
rho = max_{1 <= L <= 128} 3 * (L + 3) / (166 + L)
    = 393 / 294 = 131 / 98 (~1.336735).
```

The expression increases in L. Nodes that appear fewer times, including roots, have smaller ratios. The two hot nodes N and P are charged separately for their payload and possible repeated ID memberships.

### Lemma 3 — edges

Each edge has equal wire and output representation except for its array delimiters. It can appear in at most one output edge list unless it is a self-loop on N. The unique `(from, kind, to)` constraint and the **three** permitted kinds allow at most **three** N→N self-loops. One maximal edge object (128-byte endpoints, longest 10-byte kind) is 295 bytes; with its delimiter, 296. Therefore a safe fixed allowance for *extra* self-loop copies is `3 * 296 = 888` bytes. Other edges are already covered by `rho > 1` times their wire cost.

## Bound calculation

The fixed JSON report skeleton with a zero-value `Node` (99 bytes), a `null` parent, all non-null empty arrays, a 64-character digest, and the current schema/boundary strings is **469 bytes** without LF. Its portion outside the `Node` object is **370 bytes**, so the output skeleton plus LF contributes 371 bytes before adding the full node objects.

A minimal strict document envelope (the schema, one-byte project/source IDs, and empty arrays) has **96 bytes**. To conservatively handle the last element of each of the two input arrays, deduct **2** delimiter bytes: the envelope contribution is bounded below by **94**.

In addition to the skeleton, assign these conservative fixed allowances:

| Allowance | Bytes |
|---|---:|
| Skeleton outside N plus trailing LF | 371 |
| Extra digits for two hot-node ordinals (4 each beyond digit `0`) | 8 |
| N and P, each in up to two ID lists: `4 * (128 + 3)` | 524 |
| Extra N self-loop edge copies: `3 * 296` | 888 |
| **K** | **1,791** |

These allowances deliberately overcount some cases (for instance, P might be null and the root and leaf positions cannot all coincide). After the fixed K, the combined non-payload wire cost has output/input ratio no greater than `rho`. The remaining text payload D has expansion at most six. Therefore

```
O <= K + rho * (W - 94) + (6 - rho) * D
  <= 1791 + (131/98) * (2097152 - 94)
           + (6 - 131/98) * 540672
   = 5,326,298 + 8/49 bytes.
```

Both coefficients are positive; the maximum is obtained by taking the wire and D budgets to their upper limits. Since O is integer, **O <= 5,326,298 < 8,388,608**. This proves the claimed sufficient bound under the enumerated source contracts, **not** that a maximizer with precisely that length exists.

### Conservative alternative if the strict wire-key contract changes

The source does currently require all seven keys. The original derivation also considered an intentionally weaker case where only `id`, `parent_id`, `source_ref`, and `source_digest_claim` are required. Under that relaxation, non-root overhead is smaller, yielding `rho' = 393/260` and `K' = 1889`. Repeating the calculation gives **5,598,458.3 bytes**, still below 8 MiB. This secondary figure does *not* remove the need to review any future source change: modified report fields, allowed ID alphabet, free-text budgets, or edge kinds could invalidate both arguments.

## Empirical cross-checks (not used as proof premises)

The November 8 Claude audit reconstructed the encoder and compared it with reported Go measurements. Its highest constructed example produced **5,305,345 bytes**, about 0.4% below the strict bound. The PR #6 regression's valid-input star fixture produced **5,228,543 bytes**; another combination of full title/content/source-ref fields produced **5,304,837 bytes**. The attached audit also reported a randomized search and comparisons against those Go outputs. These observations increase confidence but cannot replace the bound's assumptions and reasoning.

## Maintenance, verification, and trust boundary

`internal/graphinspect/output_bound_contract_test.go` recalculates a conservative ceiling from `phasea.MaxInputBytes`, pins the current report skeleton and minimal wire costs, checks the required keys and exact edge-kind set, and asserts field/ID boundary acceptance/rejection. If a bound-relevant source contract changes, update and re-review this document and its tests together. The test is a drift detector, **not a formal proof checker**; reviewers must still examine changes to the encoding and inspection logic.

Verification evidence (checksums, tests, CI, synthetic counterexamples, and this proof) is not authority evidence. No source claim is trusted, no access decision is made by inspection, and no production authority gate is changed. The separate governance state remains `NEGATIVE_DISPOSITION_ORACLE_OPEN__RATIFICATION_PENDING__PHASE1_BLOCKED` until authorized review.
