# Design

## Overview
Define and continuously model-check the highest-risk security and routing invariants with TLA+ in CI.

## Key Decisions
- Formal methods focus on the security kernel invariants, not arbitrary model reasoning.
- Model checking runs continuously in CI.
- The formal scope should cover the shared workflow kernel invariants frozen by the workflow runner, workspace-role, gate, policy, and broker contracts rather than modeling feature-local approximations.

## Main Workstreams
- Define Invariants to Specify (MVP Scope)
- Write TLA+ Specification
- CI Model Checking
- Traceability

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
