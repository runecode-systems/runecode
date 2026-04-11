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

## Shared Contract Alignment

### Identity + Scope
- Shared-workspace coordination should key on stable logical identities such as `run_id`, `stage_id`, `step_id`, and `role_instance_id`.
- Retries and reruns should continue to use separate attempt identities so concurrency logic does not overload logical scope identity.

### Lifecycle + Coordination
- Public run lifecycle should remain on the shared broker lifecycle vocabulary.
- Lock waits, conflict waits, and partially blocked progress should be represented through `RunCoordinationSummary`, stage summaries, and role summaries instead of a new concurrency-specific lifecycle state.

### Approval + Gate Binding
- Exact-action approvals, stage sign-off, gate attempts, gate evidence, and gate overrides must stay bound to the correct run even when workspaces are shared.
- Shared-workspace concurrency must not allow one run to consume or satisfy another run's approval or gate result.

## Main Workstreams
- Workspace Concurrency Model
- Conflict Detection + Isolation Rules
- Runner, Broker, and TUI Integration
- Fixtures + Recovery Cases

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
