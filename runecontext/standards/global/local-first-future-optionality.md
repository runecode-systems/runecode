---
schema_version: 1
id: global/local-first-future-optionality
title: Local-First Future Optionality
status: active
aliases:
    - agent-os/standards/global/local-first-future-optionality
---

# Local-First Future Optionality

Keep MVP deployment choices local-first without baking them into boundary-visible logical contracts.

- Treat protocol objects, policy objects, artifact references, audit events, approvals, and lock/lease records as topology-neutral logical contracts.
- Keep transport, storage, and deployment choices separable from object semantics; local IPC, same-UID auth, SQLite, and single-machine operation are implementation postures, not protocol identity.
- Model deployment-specific posture explicitly in metadata or audit when it changes assurance, enforcement, or operator expectations.
- Use globally unique identifiers for boundary-visible objects; do not rely on host-local counters, filesystem paths, socket names, or process-local identity.
- Keep hashed and signed payload semantics transport-neutral and encoding-stable; transport migration must not change logical meaning.
- Reject unknown or ambiguous topology, coordination, or ownership posture fail-closed rather than silently assuming host-local behavior.
- Do not require local usernames, local filesystem paths, or host-specific handles in boundary-visible schemas except as non-authoritative diagnostics.
- Make coordination concepts explicit where later multi-host support may need them (principal identity, lock owner, lease expiry, versioning, conflict state) instead of implying single-process ownership.
- Do not add MVP-only schema shortcuts that would block later multi-user, remote-control-plane, or distributed-coordination work.
- This standard preserves future optionality; it does not require distributed deployment, remote APIs, or multi-user support in MVP.
