# Tasks

## Shared Session Trigger Model

- [ ] Define one broker-owned trigger model that both interactive chat turns and autonomous background turns can use.
- [ ] Keep plain transcript append semantics separate from execution-trigger submission rather than overloading `SessionSendMessage` into the long-term orchestration contract.
- [ ] Keep session-triggered work aligned with canonical session, run, approval, artifact, and audit identities.
- [ ] Ensure trigger semantics do not bypass workflow, policy, or approval contracts.
- [ ] Reuse the repo-scoped product lifecycle and canonical `runecode` attach/start model established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` rather than inventing an execution-specific bootstrap or attach path.
- [ ] Carry broker-visible trigger-source classification for interactive, autonomous-background, and resume/follow-up turns without creating a second authorization lane.

## Session-to-Run Orchestration

- [ ] Bind session turns to isolate-backed workflow execution rather than client-local assistant actions.
- [ ] Introduce broker-owned per-turn execution state as a first-class durable/read-model surface distinct from session summary and transcript-turn lifecycle state.
- [ ] Support links from sessions to active runs, pending approvals, artifacts, and audit records.
- [ ] Preserve project-context binding when session-driven work depends on verified RuneContext state.
- [ ] Bind project-context-sensitive session work to the validated project-substrate snapshot digest rather than ambient repo state or read-model summary identity.
- [ ] Fail closed for normal session-driven execution when repository project-substrate posture is missing, invalid, non-verified, or unsupported.
- [ ] Keep at most one active execution-bearing turn per session in `v0` so wait/resume identity remains canonical and replay-safe.

## Wait, Resume, and Reconnect

- [ ] Reuse shared broker and runner durability semantics for partial turns, waits, and resume behavior.
- [ ] Ensure reconnect after TUI close or local restart restores the same canonical session truth.
- [ ] Keep autonomous background work and later interactive follow-up on the same session lifecycle model.
- [ ] Detect and surface project-substrate drift between initial execution binding and later resume or follow-up execution.
- [ ] Route incompatible project-substrate drift to broker-owned blocked-state and remediation posture rather than continuing on stale project assumptions.
- [ ] Treat any bound validated project-substrate snapshot digest change as fail-closed drift for project-context-sensitive continuation and execution-sensitive follow-up in `v0`.
- [ ] Require execution continuation and execution-sensitive follow-up to depend on broker-owned product lifecycle posture plus broker-owned session/run truth rather than on session existence alone.
- [ ] Preserve diagnostics/remediation-only attach for inspection when repository substrate is blocked, while keeping execution continuation and new execution fail-closed until compatible posture returns.
- [ ] Keep session object lifecycle, projected session work posture, and client attachment state distinct in reconnect and wait/resume behavior.
- [ ] Keep wait kinds for operator input, formal approval, external dependency, and blocked project posture distinct in broker-owned execution state.

## Client Integration

- [ ] Support session-driven execution from the TUI chat route without making the TUI authoritative.
- [ ] Support equivalent non-chat initiation surfaces for autonomous operation.
- [ ] Keep all user-facing progress and state derived from broker-owned read models and watch streams.
- [ ] Keep clients from interpreting successful diagnostics-only attach as execution authorization.
- [ ] Add a dedicated turn-execution watch contract rather than overloading session-summary watch events with high-rate execution-state semantics.
- [ ] Keep canonical transcript durability checkpoint-based while treating in-flight execution output as advisory execution-watch state.

## Approval Profile And Autonomy Controls

- [ ] Keep formal approval frequency under the canonical approval-profile model rather than turning autonomous mode into a second approval authority.
- [ ] Add a separate broker-owned autonomy-posture control for operator-question frequency and autonomous continuation posture.
- [ ] Preserve the distinction between `waiting_operator_input` and `waiting_approval` in turn execution state and operator UX.
- [ ] Ensure autonomy posture can pause otherwise-allowed work for operator guidance without minting approval decisions or weakening hard-floor approval semantics.

## Acceptance Criteria

- [ ] Live chat and autonomous operation share one session execution model.
- [ ] Session-driven work executes through isolates and the existing secure workflow path.
- [ ] Execution-bearing turns use a broker-owned trigger contract distinct from plain transcript append.
- [ ] Sessions and turns link canonically to runs, approvals, artifacts, audit records, and relevant project context, including validated project-substrate snapshot identity where relevant.
- [ ] Transcript checkpointing and execution streaming remain distinct broker-owned contracts.
- [ ] Wait, resume, and reconnect semantics remain broker-owned and durable.
- [ ] Reconnect and continuation semantics remain aligned with the repo-scoped product lifecycle and diagnostics-only attach model frozen by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`.
- [ ] Formal approval frequency and operator-question frequency remain separate broker-owned controls, while hard-floor approvals remain human-only.
