---
schema_version: 1
id: security/audit-evidence-index-and-record-inclusion
title: Audit Evidence Index And Record Inclusion
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Audit Evidence Index And Record Inclusion

When trusted RuneCode services build or use derived audit-evidence lookup state:

- Keep the append-only ledger, segment seals, signed receipts, and other canonical evidence as the only authoritative audit history; any audit-evidence index is an acceleration layer, not a second history model
- Keep derived audit-evidence indexes rebuildable from canonical evidence alone; do not require mutable convenience tables, ad-hoc recovery notes, or ambient UI projections to reconstruct inclusion state
- Use derived index state to answer practical lookup questions such as record digest to segment or seal, seal-chain position, receipt discovery, and latest verification report selection without making those lookup tables authoritative
- Fail closed or force deterministic rebuild when derived index state disagrees with canonical evidence digests, seal linkage, or persisted verification foundations
- Keep `AuditRecordInclusion` as a derived, verifier-facing object that explains where a record lives and which seal commits to it; it must not become the canonical persistence format for the record itself
- Expose record-inclusion lookup through trusted read-only surfaces rather than direct untrusted access to ledger internals or filesystem layout
- Preserve proof-neutral vocabulary and contracts in `v0`: record inclusion may carry the material needed to support later proof systems, but the primary contract is trustworthy evidence lookup rather than proof-family-specific machinery
- Avoid machine-local convenience identifiers such as run IDs, UI cursors, or storage paths as substitutes for canonical record digest and seal identity
- Treat inclusion material that is omitted from durable storage as derivable only if the derivation remains deterministic from canonical evidence; do not backfill heuristically from unrelated receipts or sidecar filenames
- Tests should cover incremental index update, deterministic rebuild, seal rotation, disagreement detection, restart-time reload, and record-inclusion answers for both nominal and fail-closed cases
