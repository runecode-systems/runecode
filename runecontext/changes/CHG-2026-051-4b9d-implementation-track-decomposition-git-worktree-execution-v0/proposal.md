## Summary
RuneCode can decompose approved or operator-supplied implementation work into low-coupling tracks, execute eligible tracks in isolated git worktrees, and keep unrelated work moving while blocked tracks wait on user input or approval.

## Problem
Even with session execution orchestration and durable wait/resume semantics, implementation work will stall unnecessarily if one pending user decision halts an entire implementation effort.

At the same time, naive parallelization in one shared workspace risks collisions, hidden dependency mistakes, and client-local scheduling semantics that bypass the broker-owned lifecycle and policy model.

## Proposed Change
- One broker-owned implementation-track model with stable track identity, dependency edges, and explicit blocked/unblocked readiness.
- Track decomposition that can use explicit track declarations from change/spec/implementation docs when they exist and can infer candidate tracks when they do not.
- A broker-owned proposed execution-plan artifact so inferred decomposition remains auditable, reviewable, and operator-visible rather than a hidden runtime heuristic.
- Isolated git-worktree execution for low-coupling eligible tracks when confidence, dependency state, policy, and coordination posture allow it.
- Dependency-aware partial blocking so pending operator input or approval freezes only the directly affected tracks and downstream dependent tracks, while unrelated eligible tracks may continue.
- Canonical linkage from tracks and worktree-backed execution to sessions, runs, approvals, artifacts, audit records, and validated project-context bindings.
- Explicit integration and verification flow for combining track results rather than heuristic ambient merging.

## Why Now
This work belongs after session execution orchestration, workflow definition binding, and the first-party workflow pack foundations exist, because that is the point where RuneCode can productively implement real change work and needs a reviewed way to keep safe independent work moving.

Planning it now avoids a later split between:
- one-off workflow-local decomposition heuristics
- ad hoc worktree management
- and the canonical broker-owned execution model.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Explicit track declarations in approved change/spec/implementation inputs should take precedence over inferred track grouping.
- When explicit track declarations are absent, inferred tracks should still become broker-owned proposed execution-plan state rather than remaining hidden agent-local reasoning.
- Git worktrees are the preferred isolation substrate for low-coupling implementation tracks, but only when overlap risk and dependency ambiguity remain low enough for safe reviewed use.
- Worktree paths, branch names, and local filesystem mechanics remain implementation-private and non-authoritative.

## Out of Scope
- Re-introducing legacy Agent OS planning paths as canonical references.
- Replacing signed human approvals with delegated or system-authored approval decisions.
- Treating shared-workspace concurrency as identical to isolated-worktree execution.
- Forcing parallel execution when decomposition confidence is low or overlap risk is high.

## Impact
Creates one reviewed future path for multi-track implementation work: explicit or inferred track grouping, dependency-aware partial blocking, isolated git-worktree execution where appropriate, and broker-owned coordination truth across sessions, runs, approvals, artifacts, audit, and project context.
