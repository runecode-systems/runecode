# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm live chat and autonomous mode both route through the same session execution model.
- Confirm session-triggered work does not bypass workflow, policy, approval, or isolate-execution semantics.
- Confirm session links to runs, approvals, artifacts, audit records, and relevant project context stay canonical and broker-visible.
- Confirm project-context-sensitive execution binds to validated project-substrate snapshot identity rather than ambient repo assumptions.
- Confirm blocked repository substrate posture fails closed for normal session-driven execution.
- Confirm resume and reconnect handle project-substrate drift fail closed when the original binding is no longer valid.
- Confirm wait, resume, and reconnect behavior reuses existing durability semantics rather than inventing chat-local truth.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.7`.
- Confirm this change reuses the repo-scoped product lifecycle and canonical `runecode` attach/start model established by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` rather than inventing an execution-specific bootstrap path.
- Confirm successful diagnostics/remediation-only attach does not by itself authorize execution continuation or new execution.
- Confirm reconnect depends on broker-owned product lifecycle posture plus broker-owned session/run truth rather than on session existence alone.
- Confirm session object lifecycle, projected session work posture, and client attachment state remain distinct in reconnect and resume handling.

## Close Gate
Use the repository's standard verification flow before closing this change.
