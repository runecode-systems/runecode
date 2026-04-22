# Design

## Overview
Bind canonical session turns to real isolate-backed work orchestration.

## Key Decisions
- Live chat and autonomous operation must share one broker-owned session-to-execution path.
- Session, run, approval, artifact, and audit identities remain canonical broker-visible objects rather than client-local orchestration state.
- Assistant turns may propose workflow actions, but all real repository or project mutation still flows through the shared workflow, policy, approval, and isolate-execution path.
- Session wait, reconnect, and partial-turn behavior should reuse existing broker and runner durability semantics rather than inventing chat-local lifecycle rules.
- Session execution should bind to verified project context when project state is relevant.
- When project context is relevant, session-triggered work should bind to the validated project-substrate snapshot digest rather than to ambient repo state or a version string alone.
- Session-triggered work must fail closed for normal operation when repository project-substrate posture is missing, invalid, non-verified, or unsupported.
- Resume and reconnect semantics must account for project-substrate drift between original execution binding and later continuation.
- This change inherits the repo-scoped product-instance model, canonical `runecode` product lifecycle surface, and broker-owned attachability semantics established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`; it must not redefine them locally.
- Session execution reconnect must depend on broker-owned product lifecycle posture plus broker-owned session/run truth rather than on session existence alone.
- Diagnostics/remediation-only attach remains valid when repository substrate blocks normal operation, but execution continuation, new execution, and execution-sensitive follow-up remain broker-blocked until project-substrate posture becomes compatible again.
- Session object lifecycle, projected session work posture, and client attachment state remain distinct concepts; execution orchestration must not collapse them into one resume/active flag.

## Session Trigger Model

- A session turn should be able to:
  - request model completion only
  - enqueue workflow selection or planning
  - attach to an existing run or approval wait
- The same canonical trigger model should work whether the source is:
  - an interactive user turn
  - an autonomous background turn
  - a resume or follow-up turn after reconnect

The trigger model should assume that the operator reached the session through the canonical repo-scoped RuneCode product lifecycle for that repository rather than inventing a second execution-specific bootstrap or attach path.

## Project-Context Binding

- Session and run linkage to project context should reuse the shared project-substrate contract from `CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0`.
- Project-context-sensitive turns should record the validated project-substrate snapshot identity used for planning or execution.
- If project-substrate posture changes incompatibly before resume or follow-up execution, the session should surface broker-owned blocked-state and remediation posture rather than continuing on stale assumptions.

### Reconnect And Product Lifecycle Alignment

`CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` freezes the product-layer rule that reconnect may still succeed in diagnostics/remediation-only posture when repository substrate blocks normal operation.

This change should build on that rule as follows:
- reconnect may still restore inspection of canonical session, run, approval, artifact, and audit state
- execution continuation must still fail closed when the original project-context binding is no longer valid for normal managed operation
- execution-sensitive follow-up turns must not infer permission from successful attach alone
- clients must not reinterpret diagnostics-only attach as execution authorization

This keeps attachability and execution permission as distinct concepts across session orchestration.

## Main Workstreams
- Shared Session-to-Execution Trigger Model.
- Broker-Orchestrated Session Linking to Runs and Approvals.
- Partial-Turn, Wait, and Reconnect Semantics.
- Project-Substrate Snapshot Binding and Drift Handling.
- TUI and Client Integration Over Existing Typed Contracts.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
