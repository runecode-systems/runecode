## Summary
RuneCode may optionally evaluate LangGraph as an internal runner runtime for local checkpoint, wait, and resume mechanics after the native thin-kernel runner foundation is complete, but only if it is still needed and without changing trust boundaries or canonical broker-owned contracts.

## Problem
RuneCode needs durable stop, wait, persist, and resume behavior for approvals and user input across process restarts. LangGraph provides generic persistence and interrupt primitives, but adopting it too early risks coupling the runner to a third-party thread/checkpoint model before RuneCode's own plan-bound recovery, approval, and broker-reconciliation semantics are fully hardened.

## Proposed Change
- Reassess whether LangGraph is needed after the native runner durable-state and approval-wait foundation is complete.
- If still useful, evaluate LangGraph only as an internal runtime implementation behind the runner runtime seam.
- Keep broker-owned run truth, approval truth, lifecycle state, and `RunPlan` authority canonical.
- Keep LangGraph checkpoints, threads, and interrupt state non-canonical and outside the trust root unless exported through existing typed protocol objects.
- Define explicit adoption criteria and non-goals before any implementation begins.
- Require the evaluation to preserve exact-action wait semantics for hard-floor approvals such as `git_remote_ops`, including bound hashes and fail-closed remote-drift behavior.
- Require the evaluation to preserve distinct `waiting_operator_input` and `waiting_approval`, multiple simultaneous scoped waits, and dependency-aware partial blocking for unrelated eligible work.

## Why Now
This work belongs on the roadmap as an explicit optional post-MVP follow-on so the deferral is intentional rather than ambiguous. Capturing it now preserves traceability while making clear that RuneCode should first finish the native runner foundation and only implement LangGraph later if it is still justified.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Making LangGraph mandatory for MVP or alpha runner delivery.
- Letting LangGraph become the source of planning, approval truth, or operator-facing lifecycle state.
- Changing the broker local API, protocol schema families, or trust-boundary ownership model just to match LangGraph internals.
- Letting LangGraph redefine or soften exact-action approval semantics for git remote mutation or other hard-floor remote-state-mutation actions.

## Impact
Creates a clear optional post-MVP placeholder for LangGraph evaluation, preserving the decision that native runner hardening comes first while keeping future internal runtime experimentation visible and reviewable.
