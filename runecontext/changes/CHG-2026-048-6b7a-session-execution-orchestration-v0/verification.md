# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm live chat and autonomous mode both route through the same session execution model.
- Confirm session-triggered work does not bypass workflow, policy, approval, or isolate-execution semantics.
- Confirm session links to runs, approvals, artifacts, audit records, and relevant project context stay canonical and broker-visible.
- Confirm wait, resume, and reconnect behavior reuses existing durability semantics rather than inventing chat-local truth.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.7`.

## Close Gate
Use the repository's standard verification flow before closing this change.
