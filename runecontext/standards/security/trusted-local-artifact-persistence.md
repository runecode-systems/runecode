---
schema_version: 1
id: security/trusted-local-artifact-persistence
title: Trusted Local Artifact Persistence
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Trusted Local Artifact Persistence

When trusted Go services persist artifact-store state, audit logs, backup material, or other authoritative local state:

- Treat `state.json`, `audit.log`, backup manifests, backup signatures, and artifact blobs as sensitive local data
- Treat authoritative audit sidecars such as audit receipts, segment seals, and verification reports as audit-owned truth; artifact-store export copies remain derivatives and must not replace or overwrite those authoritative sidecars
- Treat `secretsd` secret metadata, protected secret-material files, lease state, revocation state, and linkage metadata as trusted local state with the same fail-closed expectations as artifact and audit persistence
- Create trusted local persistence paths with private-by-default permissions on platforms that honor POSIX mode bits
- Normalize trusted local directory permissions on startup/open so pre-existing installs do not retain broader legacy modes after an upgrade
- Do not weaken Unix permission expectations for these files or directories without an explicit reviewed exception
- Keep Windows portability explicit: permission-bit assertions may differ there, but the implementation should still avoid broader-than-necessary defaults
- Replace sensitive state files atomically with a same-directory temp file plus rename path, using backup-and-restore when portability needs it, so partial failures do not silently become the new authoritative state
- Verify that full buffers are written before syncing or promoting authoritative state files into place; short writes must fail closed and must not become the new durable truth
- Persist audit-sequence state so restarted processes do not reuse audit sequence numbers after partial failures
- If persisted audit data can get ahead of persisted state, startup must reconcile to the highest durable audit sequence before new events are emitted
- Treat durable approval records, policy decisions, revocation state, and their linkage metadata as trusted local state with the same fail-closed expectations as artifact and audit persistence
- Treat persisted runtime facts, immutable runtime evidence snapshots, runtime lifecycle state, and runtime audit-emission dedupe markers as trusted local state with the same fail-closed expectations as artifact and audit persistence
- Reconstruct authoritative runtime read models from durable persisted runtime evidence/lifecycle state after restart; do not rely on transient in-memory launcher or broker caches as the source of truth
- Backup and restore must preserve integrity/authenticity checks and must not bypass artifact digest validation, approval binding checks, policy-decision identity, revocation durability, or runtime-evidence integrity checks
- When backups are intended to be portable or operator-moved, export them as self-contained bundles that carry the manifest, authenticity material, and referenced blobs together rather than relying on ambient local blob paths
- Backup export must fail closed on corrupted source blobs and must not leave partial or invalid bundled blob payloads behind after a failed export attempt
- If bundle export writes multiple blobs and a later blob fails verification or copy, clean up earlier successfully written bundled blobs so one failed export does not leave a misleading partial blob set behind
- Backup and restore must preserve runtime evidence digests, lifecycle projections, and audit dedupe state so restarted services do not silently re-emit or orphan launcher runtime events
- Backup and restore must preserve durable runtime attestation support, verification, and projection inputs strongly enough that restored broker read models do not silently degrade to launcher-local or client-local inference
- Restore of bundled blobs must verify digest and size against the manifest before making restored blob content authoritative, and must roll back newly restored bundled blobs on failure
- If backup authenticity is keyed to a machine- or user-local persistent secret, treat that verification key as sensitive trusted local state and keep cross-store verification semantics explicit rather than ambient
- After restart or restore, fail closed unless persisted revocation and policy-decision state is reconstructed consistently enough to preserve prior deny outcomes and approval linkage
- Tests should cover both nominal persistence and fail-closed recovery paths for audit/state divergence, runtime evidence/lifecycle replay, restart-time authoritative-state reconstruction, and backup bundle corruption or partial-write cleanup
