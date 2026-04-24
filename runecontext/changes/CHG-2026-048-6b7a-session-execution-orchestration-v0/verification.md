# Verification

## Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Completed Checks
- `go test ./internal/artifacts ./internal/brokerapi ./cmd/runecode-tui`
- `go test ./internal/brokerapi`
- `just ci`

## Verification Notes
- Confirm live chat and autonomous mode both route through the same session execution model.
- Confirm session-triggered work does not bypass workflow, policy, approval, or isolate-execution semantics.
- Confirm execution-bearing turns use a broker-owned trigger contract distinct from plain transcript append.
- Confirm session and turn links to runs, approvals, artifacts, audit records, and relevant project context stay canonical and broker-visible.
- Confirm project-context-sensitive execution binds to the validated project-substrate snapshot digest rather than ambient repo assumptions or summary-only identity fields.
- Confirm blocked repository substrate posture fails closed for normal session-driven execution.
- Confirm resume and reconnect handle project-substrate drift fail closed when the original bound validated snapshot digest is no longer valid.
- Confirm wait, resume, and reconnect behavior reuses existing durability semantics rather than inventing chat-local truth.
- Confirm `waiting_operator_input` and `waiting_approval` remain distinct broker-owned states.
- Confirm pending operator input or approval blocks only the exact dependent scope and direct downstream scopes that cannot proceed safely.
- Confirm unrelated eligible work may continue only when active plan, dependency tracking, broker policy, coordination state, and project-substrate posture all allow it.
- Confirm multiple pending waits may coexist without being collapsed into one global blocked state.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.7`.
- Confirm this change reuses the repo-scoped product lifecycle and canonical `runecode` attach/start model established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` rather than inventing an execution-specific bootstrap path.
- Confirm successful diagnostics/remediation-only attach does not by itself authorize execution continuation or new execution.
- Confirm diagnostics/remediation-only attach still allows inspect-only session, run, approval, artifact, and audit views through broker-owned contracts.
- Confirm reconnect depends on broker-owned product lifecycle posture plus broker-owned session/run truth rather than on session existence alone.
- Confirm session object lifecycle, projected session work posture, and client attachment state remain distinct in reconnect and resume handling.
- Confirm transcript checkpointing and in-flight execution watch state remain distinct contracts.
- Confirm a dedicated turn-execution watch family carries in-flight turn execution semantics instead of overloading session-summary watch events.
- Confirm formal approval frequency and operator-question frequency are represented as separate broker-owned controls.
- Confirm no autonomous-mode path mints or substitutes for signed human approval decisions.

## Close Gate
Use the repository's standard verification flow before closing this change.

## Implementation Notes
- Session execution triggers now bridge into the canonical run/runtime pipeline instead of acting as chat-local state.
- Turn execution state now distinguishes transcript checkpoints from advisory execution watch updates and preserves canonical links to runs, approvals, artifacts, and audit records.
- `continue` now fails closed for `waiting_approval` turns until approval resolution clears the wait through broker-owned approval sync.
- Session detail/read models now project `pending_turn_executions` and per-turn orchestration scope/dependency IDs as durable substrate for dependency-aware partial blocking and multi-wait evolution.
