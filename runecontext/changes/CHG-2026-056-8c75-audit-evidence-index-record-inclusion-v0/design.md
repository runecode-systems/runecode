# Design

## Overview
This feature implements the generic lookup layer beneath RuneCode's verification plane.

Canonical evidence remains authoritative. The audit-evidence index exists only to make verification practical by providing deterministic, rebuildable lookup over canonical events, segments, seals, receipts, and later related evidence families.

The first-class user-facing outcome of that index is `AuditRecordInclusion`: a trusted answer to where a record lives and what sealed checkpoint commits to it.

## Goals
- Make record inclusion lookup fast enough for interactive use.
- Keep the index rebuildable from canonical evidence alone.
- Keep incremental update paths deterministic and cheap.
- Fail closed or force refresh when the index disagrees with canonical evidence.
- Provide a generic audit feature that later proof work could reuse without making proofs the foundation now.

## Non-Goals
- Replacing canonical evidence with an index.
- Building a proof-specific data model.
- Requiring a remote verifier to answer local inclusion questions.
- Storing mutable search materializations as authoritative evidence.

## Architecture

### `AuditEvidenceIndex`
Purpose: fast trusted lookup over canonical audit evidence.

Recommended fields:

- schema version
- last indexed segment id or sequence
- record inclusion map
- segment seal map
- seal-by-chain-index map
- optional latest verification report digest
- optional index-build provenance

Rules:

- store it under trusted local state only
- use owner-only permissions
- keep it canonical JSON
- rebuild it from canonical evidence
- never export it as authoritative evidence without the underlying evidence

### Minimum Lookup Support
The index should support at least:

- record digest to segment id and frame index
- segment id to current seal digest and seal chain index
- seal chain index to seal digest
- latest verification report discovery
- future optional lookups for receipts, approvals, anchors, runtime evidence, and related sidecars

### Update Semantics
Incremental updates are allowed for append and seal operations.

Required rules:

- rebuild from canonical evidence must produce the same result as incremental updates
- mismatch between index and canonical evidence must trigger refresh or fail closed
- design must avoid rewrite-heavy behavior on every small mutation as the ledger grows
- ordering must come from canonical record or seal structure, not from filesystem modification times

### `AuditRecordInclusion`
Purpose: answer the common question "where does this record live and what seal commits to it?"

Recommended fields:

- schema id and version
- record digest
- record envelope digest
- optional canonical record envelope bytes or object
- segment id
- frame index
- segment record count
- segment seal digest
- segment seal payload or digest reference
- previous seal digest when available
- ordered Merkle inclusion material or enough information to build it deterministically

Rules:

- derive it from canonical evidence
- keep it independently checkable
- make it safe to expose through a read-only trusted API
- include enough material to support external checking without forcing a full ledger rescan in the common case
- harden inclusion and checkpoint binding seams needed by downstream publication durability barriers and crash reconcile without implementing remote durability gating in this lane

## Trusted Surfaces

### `internal/auditd/`
This feature should own:

- evidence index implementation
- deterministic rebuild logic
- mismatch detection and refresh behavior
- record inclusion resolution

### `internal/brokerapi/`
This feature should expose:

- a read-only trusted local API for record inclusion
- a read-only trusted local API for index-backed lookup where needed

### `protocol/schemas/`
If `AuditRecordInclusion` is exposed across a reviewed boundary or exported, the object must be defined canonically in the shared protocol schema set rather than as an ad hoc internal JSON shape.

## Failure And Recovery Behavior
- If the index is missing, RuneCode should rebuild it from canonical evidence.
- If the index is stale, RuneCode should refresh it deterministically.
- If the index conflicts with canonical evidence and the conflict cannot be repaired safely, RuneCode should fail closed rather than serving silently wrong inclusion data.
- If a requested record cannot be found, RuneCode should return an explicit not-found result rather than fabricating partial history.

## Performance Requirements
- routine evidence append should remain near constant time with respect to historical ledger size
- record inclusion lookup should be interactive and index-backed
- seal discovery should not require rescanning the full ledger in the common case
- full rebuild must remain possible from canonical evidence alone

## Test Requirements
- deterministic rebuild tests proving rebuild output matches incremental updates
- record-inclusion lookup tests covering single-segment and multi-segment ledgers
- proper previous-seal linkage tests in multi-segment fixtures
- real computed ordered-Merkle roots in fixture seals
- mismatch handling tests proving refresh or fail-closed behavior
- owner-only permission checks for sensitive evidence directories
- API tests proving returned inclusion data is independently checkable

## Experimental Concepts To Keep, Change, And Watch

### Keep
- record digest to segment and frame mapping
- segment to seal lookup
- seal-chain-index to seal lookup
- automatic refresh or rebuild when mismatches are detected

### Change
- rename the concept away from `proof`
- keep the model generic rather than proof-specific
- keep the index derived rather than authoritative

### Watch
- avoid an implementation that rewrites too much state on every append at large scale
- keep rebuild deterministic and cheap enough for recovery workflows
- keep the same trust model while allowing append-friendly or sharded trusted derived storage as ledgers grow
- prefer compact ordered-Merkle inclusion material when it reduces export and lookup cost without weakening independent checkability
