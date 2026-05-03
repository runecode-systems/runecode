# Tasks

## Index Foundation

- [ ] Define the generic `AuditEvidenceIndex` object shape and storage rules.
- [ ] Implement record digest to segment and frame lookup.
- [ ] Implement segment to seal and seal-chain-index lookup.
- [ ] Implement latest verification report discovery lookup.
- [ ] Support incremental updates during append and seal operations.
- [ ] Support deterministic rebuild from canonical evidence.

## Mismatch Handling

- [ ] Detect mismatch between index state and canonical evidence.
- [ ] Refresh deterministically when safe.
- [ ] Fail closed when mismatch cannot be repaired safely.

## Record Inclusion

- [ ] Define the `AuditRecordInclusion` object family.
- [ ] Return segment id, frame index, segment record count, segment seal digest, and previous-seal linkage where available.
- [ ] Include ordered-Merkle inclusion material directly or enough information to derive it deterministically.
- [ ] Keep the result independently checkable against canonical evidence.
- [ ] Harden inclusion and sealing-checkpoint seams so downstream publication durability and crash-reconcile flows can bind exact action intent to prior evidence checkpoints.

## Trusted API

- [ ] Add a read-only trusted local API for record inclusion.
- [ ] Keep the API safe for operator and incident-response use without exposing trusted evidence internals to untrusted code.

## Hardening And Performance

- [ ] Keep the index stored under trusted local state with owner-only permissions.
- [ ] Ensure append and seal updates remain near constant time with respect to historical ledger size.
- [ ] Ensure common-case inclusion lookup and seal discovery are index-backed.
- [ ] Keep full rebuild possible from canonical evidence alone.

## Verification

- [ ] Add rebuild tests proving deterministic parity between incremental and rebuilt index state.
- [ ] Add inclusion lookup tests for single-segment and multi-segment ledgers.
- [ ] Add previous-seal linkage tests in multi-segment fixtures.
- [ ] Use real computed ordered-Merkle roots in fixture seals.
- [ ] Add permission and durability tests for sensitive evidence directories.
- [ ] Add mismatch-handling tests proving refresh or fail-closed behavior.

## Acceptance Criteria

- [ ] Record inclusion lookup no longer requires rescanning the full ledger in the common case.
- [ ] Seal discovery and previous-seal lookup are index-backed.
- [ ] Latest verification report discovery is index-backed in the common case.
- [ ] Rebuild from canonical evidence produces the same result as incremental updates.
- [ ] `AuditRecordInclusion` output is independently checkable against canonical evidence.
- [ ] The index remains derived, rebuildable, and non-authoritative.
