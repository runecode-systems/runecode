## Summary
RuneCode has an untrusted workflow runner with durable pause/resume state, typed propose-to-attest control-flow semantics, explicit broker-validated checkpoint reporting, and the ability to continue independent work while approval-bound scopes are waiting on signed human decisions.

## Problem
The prior combined change bundled runner, execution roles, and gates into one very large feature, reducing implementation and verification granularity.

## Proposed Change
- Runner contract and untrusted scheduler constraints.
- Durable state machine and crash recovery semantics.
- Typed propose, validate, authorize, execute, and attest loop.
- Event-style runner-to-broker checkpoint/result reporting with broker-owned public projection.
- Stable logical workflow identity with separate execution-attempt identity.
- Versioned runner journal/snapshot persistence with deterministic broker-wins reconciliation.

## Why Now
Splitting runner and durable-state foundations improves sequencing, ownership, and verification while preserving the original end-to-end objective.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Workspace role command execution details.
- Deterministic gate implementation details.

## Impact
Keeps runner and durable-state contract work reviewable as an independent feature under the workflow execution project while freezing the recovery and reconciliation rules that later workflow features must reuse.
