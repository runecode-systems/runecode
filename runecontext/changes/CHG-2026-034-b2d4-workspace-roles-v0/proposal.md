## Summary
RuneCode provides offline workspace execution roles with explicit capability boundaries for read, edit, and test operations, reviewed role-to-executor policy mapping, a trusted authoritative executor registry, and typed non-shell-passthrough executors consumed by the runner through `RunPlan` bindings.

## Problem
Workspace execution concerns were previously embedded in a larger combined change, obscuring ownership and dependency boundaries. It also left open whether executor semantics would be defined once in the trusted control plane or drift into separate Go and runner-local conventions.

## Proposed Change
- Define the MVP workspace role set.
- Implement role-specific execution boundaries.
- Enforce offline posture and non-shell-passthrough executors.
- Freeze the role-kind and executor-class matrix so later workflow and gate features reuse one execution-boundary model.
- Introduce one reviewed executor registry authoritative in trusted Go and projected read-only to the runner for plan-bound dispatch validation.
- Bind workspace execution to `RunPlan` entries so ordinary runner execution cannot escape reviewed executor contracts.

## Why Now
This split isolates execution-surface hardening and capability boundaries into a feature that can be reviewed and verified independently. Freezing one executor registry now prevents later workflow work from depending on duplicated or forked execution semantics.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runner state orchestration details.
- Gate framework details.
- Allowing workspace execution to depend on shell strings, ambient tool discovery, or runner-local executor policy.

## Impact
Keeps role-level execution boundaries explicit and reviewable as a standalone feature while freezing the reviewed executor boundary that later workflow features must reuse.
