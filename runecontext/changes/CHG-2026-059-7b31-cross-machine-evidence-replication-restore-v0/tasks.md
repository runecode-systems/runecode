# Tasks

## Replication Checkpoint Object Model

- [ ] Define a signed replication checkpoint object family distinct from `AuditEvidenceBundleManifest`.
- [ ] Bind checkpoints to project identity, node or product-instance identity, previous-checkpoint linkage, covered evidence boundary, root digests, and included object digests.
- [ ] Keep checkpoints append-only and suitable for anti-entropy, restore planning, and local thinning eligibility.
- [ ] Keep checkpoint semantics independent from bucket listing order, mutable markers, or storage path naming.

## Remote Replica Target Model

- [ ] Define a provider-neutral typed remote replica target descriptor for S3-compatible object stores.
- [ ] Bind tenant identity and project identity into the remote target model without making storage paths authoritative.
- [ ] Keep remote auth material in secretsd and lease-bound through trusted code rather than in runner, workflow, or environment-variable shortcuts.
- [ ] Preserve one remote-target contract across constrained local machines and larger shared deployments.

## Primary Replication Units

- [ ] Replicate immutable canonical evidence objects including sealed segments, seals, sidecars, runtime evidence, attestation evidence, verification reports, and external anchor evidence.
- [ ] Replicate signed replication checkpoints as the primary coordination skeleton.
- [ ] Replicate signed bundle manifests and exported verifier-friendly bundles when explicitly created, without making them the replication authority path.
- [ ] Keep object identity digest-based and content-addressed rather than path-shaped.

## Thin-Local Retention And GC

- [ ] Define the local storage split between hot active evidence, compact checkpoint skeleton state, and hydrated historical objects.
- [ ] Allow full local GC of historical bulk evidence only after trusted code confirms required remote durability.
- [ ] Keep enough local checkpoint and sparse index state so nodes can do new ordinary work without re-downloading full historical evidence.
- [ ] Fail closed when GC eligibility is ambiguous, replication confirmation is incomplete, or active prepare or reconcile state still depends on local copies.
- [ ] Emit canonical meta-audit evidence for local historical GC operations.

## Fetch-On-Miss, Restore, And Repair

- [ ] Define fetch-on-miss flows driven by signed checkpoints and immutable object digests.
- [ ] Define restore admission rules that verify remote objects before local admission and deterministic derived-index rebuild.
- [ ] Define checkpoint-driven anti-entropy and repair rather than object-store listing heuristics as authority.
- [ ] Keep restore and repair aligned with explicit import and restore meta-audit evidence from `CHG-2026-058-04e9-verification-coverage-expansion-v0`.

## Durability Posture

- [ ] Define at least `healthy`, `remote_durability_degraded`, and `local_capture_unhealthy` postures.
- [ ] Keep ordinary development allowed in `remote_durability_degraded` when local evidence capture remains healthy.
- [ ] Block publication-sensitive actions in `remote_durability_degraded`.
- [ ] Block mutation-bearing RuneCode execution in `local_capture_unhealthy` and route operators to diagnostics and remediation posture.
- [ ] Define healthy self-healing durability as requiring two independent remote targets, with one-target operation surfaced as explicit degraded posture.

## Publication-Sensitive Durability Barrier

- [ ] Define the hard-floor publication-sensitive actions that must pass the durability barrier before execution.
- [ ] Require sealing or checkpointing, signed checkpoint creation, and successful replication of required evidence to the healthy replica set before publication execute.
- [ ] Bind publication prepare records to exact repository identity, target refs, referenced patch or input digests, expected result tree hash, canonical action request hash, and evidence checkpoint digest.
- [ ] Reuse durable prepared and execute plus reconcile semantics so crash recovery remains trustworthy if a machine fails immediately after remote state mutation.
- [ ] Require post-action outcome evidence and post-action checkpoint replication before the action is considered fully complete.

## Degraded-State Recovery Seeds

- [ ] Forbid a permanent lower-assurance publication path for degraded-state changes.
- [ ] Define non-authoritative recovery-seed capture for surviving degraded-state edits, including diff and file-snapshot oriented recovery inputs when available.
- [ ] Define a fresh healthy reimplementation workflow that re-creates intended degraded-state changes under normal evidence, approval, and publication rules.
- [ ] Keep recovery seeds explicitly non-authoritative and non-publishable by themselves.

## Optional Trusted Helper

- [ ] If a helper is added, keep it in the trusted domain and subordinate to broker and auditd authority.
- [ ] Restrict helper responsibilities to queued transfer, bounded concurrency, retry, backoff, and anti-entropy execution.
- [ ] Forbid the helper from becoming a second public authority surface, second readiness truth surface, or restore-admission authority.

## Verification

- [ ] Add tests for checkpoint integrity, previous-link verification, and fail-closed restore admission.
- [ ] Add tests for GC eligibility rules and thin-local operation without historical bulk evidence.
- [ ] Add tests for fetch-on-miss verification and deterministic local index rebuild after restore.
- [ ] Add tests proving publication-sensitive actions are blocked until the pre-action durability barrier is satisfied.
- [ ] Add crash-recovery tests for prepare, execute, and reconcile around publication-sensitive actions.
- [ ] Add tests proving degraded-state recovery seeds are non-authoritative and cannot be published directly.
- [ ] Add tests proving one remote target is degraded and two independent targets are required for healthy self-healing posture.

## Acceptance Criteria

- [ ] RuneCode can replicate immutable canonical evidence and signed checkpoints across machines without creating a second truth surface.
- [ ] Developer machines can fully GC historical local bulk evidence after trusted remote durability confirmation while keeping enough local skeleton state for new ordinary work.
- [ ] Missing evidence can be restored and verified from remote durability targets through trusted fetch-on-miss and restore flows.
- [ ] Publication-sensitive actions are blocked until pre-action evidence durability is healthy and exact-action bindings include the evidence checkpoint digest.
- [ ] Crash recovery after remote publication-sensitive actions remains trustworthy through durable prepare, execute, and reconcile semantics.
- [ ] Degraded-state changes cannot be published directly and instead must be re-created in a healthy audited run.
- [ ] Healthy self-healing posture requires two independent remote targets; weaker durability remains explicit degraded posture.
