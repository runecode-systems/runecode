# Tasks

## Shared Session Trigger Model

- [ ] Define one broker-owned trigger model that both interactive chat turns and autonomous background turns can use.
- [ ] Keep session-triggered work aligned with canonical session, run, approval, artifact, and audit identities.
- [ ] Ensure trigger semantics do not bypass workflow, policy, or approval contracts.

## Session-to-Run Orchestration

- [ ] Bind session turns to isolate-backed workflow execution rather than client-local assistant actions.
- [ ] Support links from sessions to active runs, pending approvals, artifacts, and audit records.
- [ ] Preserve project-context binding when session-driven work depends on verified RuneContext state.
- [ ] Bind project-context-sensitive session work to the validated project-substrate snapshot digest rather than ambient repo state.
- [ ] Fail closed for normal session-driven execution when repository project-substrate posture is missing, invalid, non-verified, or unsupported.

## Wait, Resume, and Reconnect

- [ ] Reuse shared broker and runner durability semantics for partial turns, waits, and resume behavior.
- [ ] Ensure reconnect after TUI close or local restart restores the same canonical session truth.
- [ ] Keep autonomous background work and later interactive follow-up on the same session lifecycle model.
- [ ] Detect and surface project-substrate drift between initial execution binding and later resume or follow-up execution.
- [ ] Route incompatible project-substrate drift to broker-owned blocked-state and remediation posture rather than continuing on stale project assumptions.

## Client Integration

- [ ] Support session-driven execution from the TUI chat route without making the TUI authoritative.
- [ ] Support equivalent non-chat initiation surfaces for autonomous operation.
- [ ] Keep all user-facing progress and state derived from broker-owned read models and watch streams.

## Acceptance Criteria

- [ ] Live chat and autonomous operation share one session execution model.
- [ ] Session-driven work executes through isolates and the existing secure workflow path.
- [ ] Sessions link canonically to runs, approvals, artifacts, audit records, and relevant project context, including validated project-substrate snapshot identity where relevant.
- [ ] Wait, resume, and reconnect semantics remain broker-owned and durable.
