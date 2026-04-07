# Design

## Overview
Define explicit shared-workspace concurrency modes, deterministic locking, and auditable conflict handling beyond the one-run-per-workspace default.

## Key Decisions
- One active run per workspace remains the default fail-closed posture.
- Shared-workspace concurrency requires an explicit model, not opportunistic scheduling.
- Concurrency state must be visible to the runner, broker, policy layer, and TUI.
- Approval and artifact bindings remain run-specific even when runs execute concurrently.
- Concurrency posture must surface through the shared broker run-detail/read-model contract rather than through a second UI-only status vocabulary.

## Main Workstreams
- Workspace Concurrency Model
- Conflict Detection + Isolation Rules
- Runner, Broker, and TUI Integration
- Fixtures + Recovery Cases

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
