# Design

## Overview
Use this change as the project-level tracker for secure workflow execution while implementation lands in child feature changes.

## Key Decisions
- Child features own runtime implementation detail.
- Parent project owns sequencing and integration posture.
- Security invariants remain untrusted-runner, policy enforcement, and evidence-backed execution.
- Integration posture includes one shared broker logical API vocabulary for runs, approvals, audit posture, and operator-visible blocked state across child features.
- Integration posture also includes one shared policy vocabulary for canonical action identity, role taxonomy, gateway destination semantics, and exact-action-vs-stage-sign-off approval behavior across child features.

## Main Workstreams
- `CHG-2026-033-6e7b-workflow-runner-durable-state-v0`
- `CHG-2026-034-b2d4-workspace-roles-v0`
- `CHG-2026-035-c8e1-deterministic-gates-v0`
- Integration milestone: minimal end-to-end demo run across child features

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
