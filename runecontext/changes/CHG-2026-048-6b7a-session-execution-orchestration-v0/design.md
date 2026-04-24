# Design

## Overview
Bind canonical session turns to real isolate-backed work orchestration.

## Key Decisions
- Live chat and autonomous operation must share one broker-owned session-to-execution path.
- Session, run, approval, artifact, and audit identities remain canonical broker-visible objects rather than client-local orchestration state.
- Assistant turns may propose workflow actions, but all real repository or project mutation still flows through the shared workflow, policy, approval, and isolate-execution path.
- Session wait, reconnect, and partial-turn behavior should reuse existing broker and runner durability semantics rather than inventing chat-local lifecycle rules.
- Session execution should bind to verified project context when project state is relevant.
- When project context is relevant, session-triggered work should bind to the validated project-substrate snapshot digest rather than to ambient repo state, a summary identity field, or a version string alone.
- Session-triggered work must fail closed for normal operation when repository project-substrate posture is missing, invalid, non-verified, or unsupported.
- Resume and reconnect semantics must account for project-substrate drift between original execution binding and later continuation.
- This change inherits the repo-scoped product-instance model, canonical `runecode` product lifecycle surface, and broker-owned attachability semantics established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`; it must not redefine them locally.
- Session execution reconnect must depend on broker-owned product lifecycle posture plus broker-owned session/run truth rather than on session existence alone.
- Diagnostics/remediation-only attach remains valid when repository substrate blocks normal operation, but execution continuation, new execution, and execution-sensitive follow-up remain broker-blocked until project-substrate posture becomes compatible again.
- Session object lifecycle, projected session work posture, and client attachment state remain distinct concepts; execution orchestration must not collapse them into one resume/active flag.
- Plain transcript append semantics must remain distinct from execution-trigger submission; chat message recording must not become the long-term public contract for broker-owned turn execution.
- Per-turn execution state must be a first-class broker-owned durable surface distinct from session summary fields and transcript-turn status fields.
- Transcript durability and in-flight execution streaming must remain distinct; live execution updates are advisory execution-state projections, while canonical transcript messages are durable checkpoints.
- `v0` should allow multiple active/pending execution-bearing turns per session when dependency tracking and policy permit, while preserving canonical per-turn execution state and replay-safe sequencing.
- Formal human approval, operator-input pauses, and autonomous continuation are distinct broker-owned concepts; this change must not collapse them into one generic waiting state.
- User-facing autonomy controls should split into formal approval profile and operator-question frequency controls rather than implying system-authored approval decisions.
- Hard-floor approvals remain outside profile and autonomy controls; this change must not soften exact-action human approval for those lanes.
- Pending operator input or formal approval must block only the exact dependent scope and direct downstream scopes that cannot safely proceed, not the whole RuneCode product instance.
- Unrelated eligible work may continue only when the active plan, dependency tracking, broker policy, coordination state, and project-substrate posture all allow it.
- Multiple pending waits may coexist at once; resolving one wait must resume only the affected blocked scope(s) rather than implicitly unblocking unrelated waits.

## Session Trigger Model

- Introduce one broker-owned session-turn trigger contract for execution-bearing turns rather than overloading plain transcript append.
- Plain transcript append may remain as a narrow contract for recording durable messages, annotations, or other non-authoritative transcript updates.
- A session turn trigger should be able to:
  - request model completion only
  - request workflow selection or planning
  - attach to or continue an existing run or approval wait
  - continue from a broker-owned wait state after reconnect
- The same canonical trigger model should work whether the source is:
  - an interactive user turn
  - an autonomous background turn
  - a resume or follow-up turn after reconnect
- Trigger source classification should be broker-visible explanatory state, for example:
  - `interactive_user`
  - `autonomous_background`
  - `resume_follow_up`
- Trigger source classification must not become a second authorization lane.

The trigger model should assume that the operator reached the session through the canonical repo-scoped RuneCode product lifecycle for that repository rather than inventing a second execution-specific bootstrap or attach path.

## Turn Execution State Model

- Add a broker-owned per-turn execution state/read model distinct from:
  - session object lifecycle
  - session projected work posture
  - transcript turn lifecycle
- That per-turn execution state should carry at least:
  - `session_id`
  - `turn_id`
  - trigger source classification
  - execution state
  - wait kind
  - primary run identity when present
  - pending approval identity when present
  - bound validated project-substrate snapshot digest when relevant
  - blocked reason code when blocked or degraded
  - terminal outcome when completed, failed, or cancelled
- Session summary may still project high-level broker-owned work posture, but that summary remains an aggregate operator cue rather than the canonical execution state for an in-flight turn.

### Execution State Vocabulary

- Keep transcript turn `status` narrow to transcript lifecycle.
- Keep session `status` narrow to session-object lifecycle.
- Introduce a distinct turn-execution vocabulary able to distinguish at least:
  - queued
  - planning
  - running
  - waiting
  - blocked
  - failed
  - completed
- Introduce a distinct wait-kind vocabulary able to distinguish at least:
  - `operator_input`
  - `approval`
  - `external_dependency`
  - `project_blocked`

This allows autonomous continuation, operator questions, formal approval waits, and blocked project posture to remain machine-distinct.

## Project-Context Binding

- Session and run linkage to project context should reuse the shared project-substrate contract from `CHG-2026-046-a91d-runecontext-verified-project-substrate-compatibility-lifecycle-v0`.
- Project-context-sensitive turns should record the validated project-substrate snapshot digest used for planning or execution.
- Run and session summaries may continue to project `project_context_identity_digest` for operator-facing correlation, but execution binding and drift checks should use the exact bound validated snapshot digest.
- `v0` should fail closed on any incompatible execution binding drift: if the current validated project-substrate snapshot digest differs from the bound digest for a project-context-sensitive execution turn, continuation and execution-sensitive follow-up must block.
- If project-substrate posture changes incompatibly before resume or follow-up execution, the session should surface broker-owned blocked-state and remediation posture rather than continuing on stale assumptions.

### Reconnect And Product Lifecycle Alignment

`CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` freezes the product-layer rule that reconnect may still succeed in diagnostics/remediation-only posture when repository substrate blocks normal operation.

This change should build on that rule as follows:
- reconnect may still restore inspection of canonical session, run, approval, artifact, and audit state
- execution continuation must still fail closed when the original project-context binding is no longer valid for normal managed operation
- execution-sensitive follow-up turns must not infer permission from successful attach alone
- clients must not reinterpret diagnostics-only attach as execution authorization

This keeps attachability and execution permission as distinct concepts across session orchestration.

## Transcript Durability And Execution Streaming

- Canonical transcript durability should record stable checkpoints rather than treating token deltas or framework-local partial state as the public contract.
- The broker should durably record at least:
  - triggering user or autonomous input
  - committed planning/execution summaries when they become stable
  - approval-wait entry or clearance summaries where relevant
  - final assistant or tool messages when they become canonical transcript content
- In-flight execution progress should flow through a dedicated turn-execution watch family rather than requiring transcript mutation to act as the live execution transport.
- Advisory live output fragments may exist later, but they must remain additive execution-watch state rather than replacing canonical transcript checkpointing.

This preserves replay safety, transcript durability, and broker-owned recovery semantics.

## Partial Blocking Across Work Items

- Operator-input waits and formal approval waits should behave as dependency-aware partial-blocking semantics rather than global-stop semantics.
- The broker-owned execution model should block:
  - the exact turn or narrower bound scope that is waiting
  - any direct downstream scope that cannot safely proceed without the pending input
- The broker-owned execution model may continue unrelated eligible work only when:
  - the active plan explicitly permits that work
  - dependency tracking shows it is unblocked
  - broker policy does not require the pending user input for that scope
  - coordination state does not block it
  - project-substrate posture allows it
- Multiple pending waits may coexist simultaneously.
- Resolution of one wait must resume only the affected blocked scope(s); it must not implicitly authorize or unblock unrelated waiting scopes.

This freezes the execution semantics that later multi-track implementation and isolated-worktree execution should reuse rather than redefining a second wait/scheduler model.

## Diagnostics-Only Attach And Operation Classes

- Diagnostics/remediation-only attach must still allow inspection of canonical broker-owned state, including sessions, runs, approvals, artifacts, and audit records.
- Broker operation gating should distinguish at least:
  - inspect-only operations
  - diagnostics/remediation operations
  - normal-execution-required operations
  - continuation-sensitive operations
- Session and run inspection flows must remain available in diagnostics/remediation-only attach posture.
- New execution-trigger submission, execution continuation, and execution-sensitive follow-up must remain blocked when normal operation is not allowed.
- Approval resolution must remain action-sensitive:
  - approvals that enable diagnostics/remediation flows may still resolve in blocked repository posture when policy allows
  - approvals that would resume or authorize productive execution remain blocked until compatible normal-operation posture returns

This keeps successful attach, inspection permission, remediation permission, and productive execution authorization as distinct broker-owned concepts.

## Approval Profile And Autonomy Controls

- Formal approval frequency should remain controlled by the canonical approval-profile model rather than by a session-local heuristic.
- Operator-question frequency should be a separate broker-owned orchestration control distinct from formal approval profile.
- A user-facing autonomy UI may present both as separate sliders or equivalent controls, but the broker should compile them into distinct typed inputs:
  - approval profile for formal policy-controlled human approval timing
  - autonomy posture for operator-question frequency and autonomous continuation posture
- `autonomy_posture` may influence when otherwise-allowed work pauses for operator guidance, but it must not:
  - mint approval decisions
  - satisfy policy-required human approval by itself
  - lower hard-floor assurance
  - override blocked repository substrate posture
- System-authored or delegated approval decisions are explicitly out of scope for this change.

## Main Workstreams
- Shared Session-to-Execution Trigger Model.
- Broker-Owned Turn Execution State And Wait Vocabulary.
- Broker-Orchestrated Session Linking to Runs, Approvals, Artifacts, And Audit.
- Partial-Turn, Wait, Resume, And Reconnect Semantics.
- Project-Substrate Snapshot Binding And Strict Drift Handling.
- Dedicated Turn-Execution Watch Contracts Plus Transcript Checkpointing.
- TUI And Other Clients Over Existing Typed Contracts.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
