# Design

## Overview
Define explicit shared-workspace concurrency modes, deterministic locking, and auditable conflict handling beyond the one-run-per-workspace default.

## Key Decisions
- One active run per workspace remains the default fail-closed posture.
- Shared-workspace concurrency requires an explicit model, not opportunistic scheduling.
- Concurrency state must be visible to the runner, broker, policy layer, and TUI.
- Approval and artifact bindings remain run-specific even when runs execute concurrently.
- Concurrency posture must surface through the shared broker run-detail/read-model contract rather than through a second UI-only status vocabulary.
- Concurrency should reuse the shared workflow identity model: locks, conflicts, and scoped waits should key off stable logical scope identities while retries/reruns use separate attempt identities.
- Partial blocking or lock contention must surface through shared coordination/read-detail contracts rather than a new public run lifecycle enum.
- Approval, gate, and evidence bindings remain run-specific and must not become workspace-global under concurrency.
- Runtime-instance backend posture is distinct from `run_id` and other workflow identities; future concurrency or scaling work must not treat instance-level backend selection as run-local truth.
- Shared-workspace concurrency must not let one run consume, satisfy, or inherit another run's reduced-assurance backend-selection approval.
- Concurrency must not silently merge, reinterpret, or ignore validated project-substrate snapshot identity when concurrent runs depend on project context.
- Project-substrate drift under shared-workspace concurrency must fail closed or surface explicit coordination/remediation posture rather than continuing on stale mixed assumptions.
- Shared-workspace coordination remains broker-owned truth inside one repo-scoped product instance for the authoritative repository root; client tabs, workbench state, transport bindings, or local attach mechanics must not become concurrency ownership truth.

## Shared Contract Alignment

### Identity + Scope
- Shared-workspace coordination should key on stable logical identities such as `run_id`, `stage_id`, `step_id`, and `role_instance_id`.
- Retries and reruns should continue to use separate attempt identities so concurrency logic does not overload logical scope identity.
- If a future scheduler or scaling layer introduces explicit runtime-instance identity, it should remain separate from workflow identity and preserve the reviewed instance-scoped backend posture model.

### Lifecycle + Coordination
- Public run lifecycle should remain on the shared broker lifecycle vocabulary.
- Lock waits, conflict waits, and partially blocked progress should be represented through `RunCoordinationSummary`, stage summaries, and role summaries instead of a new concurrency-specific lifecycle state.
- Concurrency ownership and coordination should build on the canonical repo-scoped product lifecycle established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` rather than introducing transport-local or client-local product ownership semantics.

### Approval + Gate Binding
- Exact-action approvals, stage sign-off, gate attempts, gate evidence, and gate overrides must stay bound to the correct run even when workspaces are shared.
- Shared-workspace concurrency must not allow one run to consume or satisfy another run's approval or gate result.
- Reduced-assurance backend posture approvals remain exact-action and instance-posture-bound; concurrency must not reinterpret them as workspace-global capability grants.

### Project-Context Binding Under Concurrency
- If concurrent runs bind different validated project-substrate snapshots, the broker and runner must surface that difference explicitly rather than assuming one ambient project-context truth.
- Shared-workspace concurrency must not let one run satisfy another run's project-context preconditions or blocked-state remediation.

## Main Workstreams
- Workspace Concurrency Model
- Conflict Detection + Isolation Rules
- Runner, Broker, and TUI Integration
- Project-Substrate Drift and Snapshot Coordination
- Fixtures + Recovery Cases

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
