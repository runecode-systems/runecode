## Summary
RuneCode adds a trusted cross-machine evidence replication and restore foundation that keeps canonical evidence authoritative on each node, replicates immutable evidence objects and signed replication checkpoints to configured remote durability targets, allows thin-local history retention on storage-constrained developer machines, and restores or repairs missing evidence without introducing a second truth surface.

## Problem
`CHG-2026-057-d5c1-verification-plane-foundation-v0` deliberately stopped at local canonical persistence, portable evidence bundles, and offline verification. That is the right MVP boundary, but it leaves one important gap for real multi-machine use:

- developers, remote workers, and shared servers need to verify related work across machines
- developer laptops may clear historical evidence to free local storage
- any machine may lose power, run out of disk, or become unrecoverable at any moment
- publication-sensitive actions need stronger durability guarantees than best-effort local persistence

Without a reviewed follow-on change, teams would be pushed toward unsafe shortcuts:

- treating exported bundles as the primary shared replication primitive
- treating object-storage listings or mutable remote state as authority
- creating local or runner-owned replication heuristics outside the trusted verification model
- allowing publication under degraded durability with no durable recovery path

Those shortcuts would conflict with the current verification-plane foundation, which explicitly requires canonical evidence to remain under trusted control, derived surfaces to stay rebuildable, and future cross-machine workflows to avoid creating a second truth surface.

## Proposed Change
- Add one trusted cross-machine evidence replication and restore lane owned by broker and auditd rather than by runner or an untrusted sidecar.
- Replicate immutable canonical evidence objects and signed replication checkpoint manifests as the primary shared durability substrate.
- Replicate verifier-friendly evidence bundles and signed export manifests when they are explicitly created, without making bundles the replication authority model.
- Add typed S3-compatible remote replica target descriptors with tenant and project scoped namespace layout while keeping storage paths non-authoritative.
- Add a thin-local retention model that allows historical local evidence GC only after required remote durability is confirmed.
- Keep enough small local checkpoint and index skeleton state so a node can keep functioning for new work without re-downloading full historical evidence.
- Add fetch-on-miss, restore, and anti-entropy repair flows driven by signed replication checkpoints and verified immutable object identities.
- Freeze a durability barrier for publication-sensitive actions: required pre-action evidence must be sealed or checkpointed and durably replicated to the healthy replica set before the action executes.
- Reuse durable prepared and execute plus reconcile semantics for publication-sensitive actions so crash recovery remains trustworthy even if a machine fails immediately after remote state mutation.
- Forbid a permanent lower-assurance publication path for degraded-state changes. If degraded-state work survives outside a healthy evidentiary run, RuneCode should capture it only as a recovery seed and re-create it through a fresh healthy audited run before publication.
- Keep one topology-neutral architecture across constrained local devices and scaled deployments by varying only queue depth, cache size, and target count rather than logical trust semantics.

## Why Now
This work should be tracked now as a dedicated downstream change so the current verification-plane bundle and coverage work can remain future-safe without being overloaded into a federation redesign.

Freezing the replication, GC, and publication-durability model now avoids later pressure to retrofit:

- bundle manifests into replication checkpoints
- object-store path layout into semantic identity
- local storage pressure exceptions into permanent lower-assurance publication lanes
- workflow-local publication shortcuts that bypass durable recovery and exact-action approval rules

## Assumptions
- `CHG-2026-057-d5c1-verification-plane-foundation-v0` remains the authority for canonical evidence versus derived surfaces.
- `CHG-2026-055-546a-verification-evidence-preservation-bundle-export-v0` continues to own bundle export and offline verification semantics; this change must extend that work without making bundles the replication trust root.
- `CHG-2026-058-04e9-verification-coverage-expansion-v0` remains the lane for meta-audit evidence such as export, import, restore, retention, and verification-surface changes.
- `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` remains the authority for repo-scoped product instance identity and local trusted lifecycle posture.
- Developer machines may intentionally delete historical local evidence after reviewed durability conditions are met.
- Some operators will use one remote durability target for convenience, but the healthy self-healing posture should require two independent remote targets.

## Out of Scope
- Replacing local canonical evidence with remote object storage as the authoritative source of truth.
- Making bundles the primary replication primitive.
- Allowing runner-owned, workflow-local, or client-local evidence federation authority.
- Defining peer-to-peer replication as a required first implementation slice.
- Allowing permanent lower-assurance publication of degraded-state changes.

## Impact
This change creates one reviewed future path for multi-machine evidence durability:

- local canonical evidence remains authoritative per node
- remote S3-compatible stores become verified durability substrates rather than authority surfaces
- publication-sensitive actions gain a pre-action durability barrier and durable recovery semantics
- developer machines can shed historical local evidence safely after confirmed remote durability
- teams can fetch and verify missing evidence on demand without changing the trust model

It also keeps cross-machine evidence work aligned with existing hard-floor remote-mutation semantics: exact-action approval, typed request binding, durable prepare and execute lifecycles, and fail-closed drift handling remain shared foundations rather than feature-local exceptions.
