## Summary
RuneCode provides offline workspace execution roles with explicit capability boundaries for read, edit, and test operations, reviewed role-to-executor policy mapping, and typed non-shell-passthrough executors.

## Problem
Workspace execution concerns were previously embedded in a larger combined change, obscuring ownership and dependency boundaries.

## Proposed Change
- Define the MVP workspace role set.
- Implement role-specific execution boundaries.
- Enforce offline posture and non-shell-passthrough executors.
- Freeze the role-kind and executor-class matrix so later workflow and gate features reuse one execution-boundary model.

## Why Now
This split isolates execution-surface hardening and capability boundaries into a feature that can be reviewed and verified independently.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runner state orchestration details.
- Gate framework details.

## Impact
Keeps role-level execution boundaries explicit and reviewable as a standalone feature while freezing the reviewed executor boundary that later workflow features must reuse.
