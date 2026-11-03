# Deterministic graph diff — development contract v0.2

**Status:** implemented, non-authoritative, experimental. **Dialect:** `gen-type-go/graph-diff-v0`. **Go input dialect:** `gen-type-go/phase-a-graph-v0`.

## Command

```sh
go run ./cmd/gen-type-go diff BEFORE.json AFTER.json
go run ./cmd/gen-type-go diff --summary BEFORE.json AFTER.json
```

A path of `-` means standard input; at most **one** input may be `-`. Both paths are read separately, bounded to **2 MiB each**, and decoded by the existing strict case-sensitive graph parser. No output is produced until **both** graphs are structurally valid and comparison encoding succeeds. All errors, including missing files, invalid syntax, duplicate IDs, dangling parents/edges, cycles, bad ordinal sequences and over-budget inputs, exit **2** and write a diagnostic to stderr without a success JSON body. Successful comparison exits **0**, including when the graphs differ. A write error also exits **2** (a failing output stream can have consumed partial bytes; do not mistake that for a completed report).

`--summary` outputs deterministic plain-text counts and hashes instead of JSON, solely as a convenience to humans. Existing v0.1 commands are unchanged except the development version string and shared input-reader refactor. The command does not create files, network connections, signatures, access decisions, or policy changes. Redirect stdout to save a successful result.

## JSON contract

A single compact UTF-8 JSON object plus one LF, with keys in fixed struct order:

- `schema`: `gen-type-go/graph-diff-v0`.
- `before_sha256`, `after_sha256`: lowercase SHA-256 hex of each input's **canonical JSON plus final LF**, not of its original source file.
- `identical`: true iff canonical encodings are byte-identical.
- `identity_boundary`: fixed warning `node_id_equality_is_syntactic_only;trusted_cross_import_lineage_unverified`.
- `metadata_changes`: sorted by declared key order (`project_id`, then `source_id`); each `{field,before,after}`. The `schema` is constant for valid inputs and cannot differ.
- `added_nodes`, `removed_nodes`: sorted node ID strings.
- `modified_nodes`: sorted `{id,fields}` for syntactically matching IDs whose `title`, `content`, `source_ref`, or `source_digest_claim` differ. `fields` are in this documented order. **Values are intentionally omitted** to prevent duplication of large contents, not to claim confidentiality. Parent/order differences are reported separately.
- `reparented_nodes`: sorted `{id,before_parent_id,after_parent_id}` for shared syntactic IDs with different parent IDs.
- `child_order_changes`: sorted by `parent_id`; each `{parent_id,before,after}` contains the complete ordered child ID lists. The empty parent ID `""` designates the roots. This catches sibling reorder and membership changes from added/removed/reparented nodes.
- `added_edges`, `removed_edges`: graph-edge objects `{from,to,kind}` sorted by `(from,kind,to)`.

All eight list fields are explicit arrays (`[]`, never `null`). No properties are inferred. In particular, a node reordered under the same parent is captured in `child_order_changes`, and changes to graph links are captured only in `added_edges` and `removed_edges` (not necessarily equivalent to a parent change).

## Identity and provenance boundary

IDs are **snapshot-local untrusted identifiers**. Matching `node.id` across two inputs is a *syntactic join only*. An imported node with a reused ID may show as `modified_nodes` rather than an add/remove pair; **this is not evidence of true node continuity**. No project or source ID, source digest claim, content similarity, or canonical checksum authenticates an issuer, lineage, protected policy revision, or permission. The diff cannot approve migrations, admission, access deltas, or authority decisions. For different projects or source IDs, this warning is especially consequential; it also applies when these IDs are equal.

## Resource and determinism contract

The existing phasea graph constraints apply to **each** input: maximum 10,000 nodes, 30,000 edges, 2 MiB of JSON input, strict identifiers and bounded text fields, acyclic complete hierarchy, contiguous sibling order. The diff output is marshaled wholly in memory and must not exceed **16 MiB**. No untrusted map iteration order reaches serialized output. Order independence of the wire node and edge arrays is inherited from phasea canonicalization. Only canonical before/after graph bytes determine the result, not process state, wall time, locale, filesystem path, or random seed. Both input structs remain unchanged. The output can be recreated from the same explicit inputs without a running service.

## Example

```sh
go run ./cmd/gen-type-go diff testdata/tiny_graph.json testdata/diff_after.json
go run ./cmd/gen-type-go diff --summary testdata/tiny_graph.json testdata/diff_after.json
```

The example reports one added node, two modified text field names, a changed ordered-child list, two added edges and one removed edge. `testdata/ambiguous_reused_node_id.json` exercises an **untrusted** reused identifier and source-ID mismatch. The historical negative-disposition and lineage oracle are not part of this feature.
