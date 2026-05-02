## Summary
Deliver a trusted generic audit-evidence index and a first-class `AuditRecordInclusion` capability so RuneCode can answer the common question: "Where does this record live, and which seal commits to it?"

This feature turns raw canonical audit evidence into fast, rebuildable lookup surfaces without making those derived surfaces authoritative.

## Problem
Canonical evidence must remain the source of truth, but a verifier or operator cannot rescan the full ledger every time they need to resolve one record, find the current seal for a segment, or answer an incident-response question under pressure.

RuneCode needs a generic evidence index because:

- record inclusion lookup should be interactive, not full-ledger reconstruction by default
- seal discovery and previous-seal lookup should be routine
- bundle export and incident-response flows need stable ways to resolve a digest back to canonical evidence
- future public verification and inclusion-proof work benefit from a generic inclusion model even if proof systems are not the `v0` foundation

At the same time, the index must not become a second truth surface. If it is authoritative, mutable, or unrebuildable, it weakens the entire evidence model.

## Proposed Change
- Add a generic `AuditEvidenceIndex` stored only under trusted local state.
- Keep the index derived and rebuildable from canonical evidence rather than authoritative.
- Support at least these lookups:
  - record digest to segment id and frame index
  - segment id to current seal digest and seal chain index
  - seal chain index to seal digest
  - latest verification report discovery
  - later optional lookups for approvals, receipts, anchors, and runtime evidence
- Allow incremental updates during append and seal operations, but require deterministic rebuild from canonical evidence.
- Fail closed or refresh when the index and canonical evidence disagree.
- Add a new derived object family, `AuditRecordInclusion`, that maps a record digest to its segment, seal, and inclusion material.
- Provide a read-only trusted local API and local helper for record inclusion.
- Keep the design generic and audit-oriented rather than proof-specific.

## Why Now
This is the first implementation workstream because it removes a major practical blocker for the rest of the verification plane.

Without a generic evidence index and inclusion model:

- bundle export must rediscover evidence expensively
- incident response cannot answer inclusion questions quickly
- seal-chain inspection remains awkward
- future verification UX either rescans too much state or invents unsafe shortcuts

This feature also preserves the most useful salvage from earlier proof-oriented exploration while renaming and generalizing it into the correct verification-plane vocabulary.

## Assumptions
- Canonical audit events, segments, and seals remain the authoritative history.
- Trusted local storage is available for owner-only derived metadata.
- A read-only trusted API is an acceptable way to expose record-inclusion results without granting untrusted code direct evidence-store access.
- Inclusion material may be optional when it would be expensive to persist directly, as long as RuneCode can derive it deterministically from canonical evidence.

## Out of Scope
- Making the index the only history source.
- Introducing proof-generation or proof-verification CLI surfaces.
- Renaming the verification plane around proofs rather than around evidence.
- Adding remote database requirements just to answer local inclusion questions.
- Treating machine-local convenience keys such as `run_id` as sufficient historical evidence.

## Impact
This feature gives RuneCode one generic, practical, trustworthy lookup substrate for verification work.

If completed, RuneCode will be able to:

- resolve a record digest to its segment and seal quickly
- discover seal-chain state without scanning the whole ledger in the common case
- rebuild the index from canonical evidence when corruption or drift is suspected
- support incident response, export, and future independent verification workflows without inventing a second history model
