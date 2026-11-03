# Security and support scope

**Experimental read-only graph preview. Not a security boundary.**

The public CLI accepts local synthetic/untrusted JSON, validates bounded graph structure, produces deterministic bytes/hashes, and generates inert SVG/HTML. It does not authenticate users, check permission to read a graph file, defend a server endpoint, encrypt storage, or issue trusted identity. Any readable local file passed by a user is treated as user-provided data. Generated HTML/SVG is not an access-control-filtered view. Avoid using confidential production data in this preview.

The `internal/phasea` policy evaluator and sampled diagnostic are historical isolated proof code only. They are **not** an active authorization interface, and a successful pure decision does not grant permission to publish or admit a revision. Cross-import IDs and source digests are not authenticated; identity continuity is unresolved.

**Open high-priority gates:** RFC Draft 004 ratification; issuer-qualified principal authentication; protected memberships and active policy publication; trusted node/grant/deny lineage; coherent policy epochs and snapshot revisions; exhaustive access-relaxation approval; and a normative decision oracle distinguishing DENY/BLOCKED/UNRESOLVED with overlap precedence. Phase 1 is blocked.

For a suspected security issue, contact the repository owner privately through an owner-provided channel, or use GitHub private vulnerability reporting **if the owner enables it**. Do not publish exploit payloads for third-party deployments in a public issue. No response-time SLA or support/security certification is claimed.

CI checks are bounded evidence; they do not establish authenticated provenance, trusted policy, deployment safety, or independent specialist sign-off.
