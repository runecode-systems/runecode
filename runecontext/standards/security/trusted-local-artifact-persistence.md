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
- Treat `secretsd` secret metadata, protected secret-material files, lease state, revocation state, and linkage metadata as trusted local state with the same fail-closed expectations as artifact and audit persistence
- Create trusted local persistence paths with private-by-default permissions on platforms that honor POSIX mode bits
- Normalize trusted local directory permissions on startup/open so pre-existing installs do not retain broader legacy modes after an upgrade
- Do not weaken Unix permission expectations for these files or directories without an explicit reviewed exception
- Keep Windows portability explicit: permission-bit assertions may differ there, but the implementation should still avoid broader-than-necessary defaults
- Replace sensitive state files atomically with a same-directory temp file plus rename path, using backup-and-restore when portability needs it, so partial failures do not silently become the new authoritative state
- Persist audit-sequence state so restarted processes do not reuse audit sequence numbers after partial failures
- If persisted audit data can get ahead of persisted state, startup must reconcile to the highest durable audit sequence before new events are emitted
- Treat durable approval records, policy decisions, revocation state, and their linkage metadata as trusted local state with the same fail-closed expectations as artifact and audit persistence
- Treat persisted runtime facts, immutable runtime evidence snapshots, runtime lifecycle state, and runtime audit-emission dedupe markers as trusted local state with the same fail-closed expectations as artifact and audit persistence
- Reconstruct authoritative runtime read models from durable persisted runtime evidence/lifecycle state after restart; do not rely on transient in-memory launcher or broker caches as the source of truth
- Backup and restore must preserve integrity/authenticity checks and must not bypass artifact digest validation, approval binding checks, policy-decision identity, or revocation durability
- Backup and restore must preserve runtime evidence digests, lifecycle projections, and audit dedupe state so restarted services do not silently re-emit or orphan launcher runtime events
- After restart or restore, fail closed unless persisted revocation and policy-decision state is reconstructed consistently enough to preserve prior deny outcomes and approval linkage
- Tests should cover both nominal persistence and fail-closed recovery paths for audit/state divergence, runtime evidence/lifecycle replay, and restart-time authoritative-state reconstruction
