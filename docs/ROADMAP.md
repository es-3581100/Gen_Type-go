# Roadmap (proposals, not completed milestones)

1. Publish the reviewed read-only CLI preview after MIT source review and explicit publication approval (the full inherited race suite passed locally).
2. Improve deterministic hierarchy import adapters with source preservation and local identity only; separately review any new parser or untrusted-file boundary.
3. Finalize **specification** of issuer-qualified authority, trusted node-lineage issuance/revocation, coherent protected policy epochs and revision admission; define and independently test negative-disposition precedence. Keep historical evaluators separate.
4. Only after ratification and explicit human Phase 1 authorization: propose a new independently audited composite authority implementation. No existing evaluator automatically inherits approval.
5. Later independently gated work: SQLite project isolation, history/search, typed scoped agent APIs, interactive HTML5/SVG frontend, and opt-in retrieval/discovery.

Do not silently promote roadmap entries into implemented or authorized capabilities.

## v0.2 candidate development

The bounded deterministic graph diff CLI is implemented **in an unmerged development candidate**, not a hosted or released feature. Its node-ID comparisons are syntactic only and have no authority. Next suggested safe milestone: add explicit read-only, source-local graph statistics and refine inert SVG layouts against real bounded fixtures. Cross-import identity verification, protected policy and Phase 1 remain blocked and must not be inferred from diff.
