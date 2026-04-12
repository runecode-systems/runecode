## Summary
RuneCode has an untrusted workflow runner that consumes one broker-compiled immutable `RunPlan`, maintains durable pause/resume state, executes a typed propose-to-attest control-flow loop, reports explicit broker-validated checkpoints and results, and continues independent work while approval-bound scopes are waiting on signed human decisions.

## Problem
The prior combined change bundled runner, execution roles, and gates into one very large feature, reducing implementation and verification granularity. It also left too much ambiguity around whether workflow order, retries, and gate placement would be planned in trusted Go or inferred locally in the runner.

## Proposed Change
- Runner contract and untrusted scheduler constraints.
- Broker-compiled immutable `RunPlan` consumption contract for runner execution.
- Durable state machine and crash recovery semantics.
- Typed propose, validate, authorize, execute, and attest loop.
- Event-style runner-to-broker checkpoint/result reporting with broker-owned public projection.
- Stable logical workflow identity with separate execution-attempt identity.
- Versioned runner journal/snapshot persistence with deterministic broker-wins reconciliation.
- Thin runner-kernel architecture rather than a runner-local planner or policy engine.
- Native runner-first hardening for approval waits, restart recovery, idempotent side effects, and partial-blocked scheduling before any optional internal orchestration runtime is introduced.
- A narrow internal runtime seam for local checkpoint/wait/resume mechanics only, preserving broker-owned planning, approval truth, and lifecycle authority.

## Why Now
Splitting runner and durable-state foundations improves sequencing, ownership, and verification while preserving the original end-to-end objective. Freezing the runner as a `RunPlan` consumer now prevents future workflow families from depending on runner-local planning semantics that would later have to be undone.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Workspace role command execution details.
- Deterministic gate implementation details.
- Letting the runner become the source of planning, authorization, or operator truth.
- Making LangGraph or any other runner-local orchestration library the canonical execution or trust-root model for this feature.

## Impact
Keeps runner and durable-state contract work reviewable as an independent feature under the workflow execution project while freezing the `RunPlan`, recovery, and reconciliation rules that later workflow features must reuse.

This change also freezes the near-term delivery posture: RuneCode should first complete the native thin-kernel durable-state and approval-wait foundation, while any future LangGraph adoption remains optional, internal-only, and separately tracked for a post-MVP reassessment.
