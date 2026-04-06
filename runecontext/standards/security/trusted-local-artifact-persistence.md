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

When trusted Go services persist artifact-store state, audit logs, or backup material locally:

- Treat `state.json`, `audit.log`, backup manifests, backup signatures, and artifact blobs as sensitive local data
- Create trusted local persistence paths with private-by-default permissions on platforms that honor POSIX mode bits
- Normalize trusted local directory permissions on startup/open so pre-existing installs do not retain broader legacy modes after an upgrade
- Do not weaken Unix permission expectations for these files or directories without an explicit reviewed exception
- Keep Windows portability explicit: permission-bit assertions may differ there, but the implementation should still avoid broader-than-necessary defaults
- Persist audit-sequence state so restarted processes do not reuse audit sequence numbers after partial failures
- If persisted audit data can get ahead of persisted state, startup must reconcile to the highest durable audit sequence before new events are emitted
- Backup and restore must preserve integrity/authenticity checks and must not bypass artifact digest validation
- Tests should cover both nominal persistence and fail-closed recovery paths for audit/state divergence
