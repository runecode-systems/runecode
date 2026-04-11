# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change remains scheduled for `v0.1.0-alpha.3` and no longer references `v0.1.0-alpha.4`.
- Confirm the change now defines the MVP TUI as a hybrid dashboard-first shell with a first-class chat route.
- Confirm the TUI remains a strict broker/local-API client with no daemon-private data dependencies and no CLI scraping for state that already has typed contracts.
- Confirm Bubble Tea is still the required framework and that the design follows its message-driven architecture rather than a monolithic or side-effect-heavy model.
- Confirm keyboard-only operation and additive mouse support are explicit MVP requirements.
- Confirm visible primary navigation on wide layouts, command palette support, and focus visibility are part of the interaction foundation.
- Confirm approval UI notes keep `policy_reason_code`, `approval_trigger_code`, and system error states distinct.
- Confirm the TUI plan distinguishes exact-action approvals from stage sign-off and explains supersession/staleness from typed data rather than scraped prose.
- Confirm the TUI plan distinguishes authoritative broker state from advisory runner state.
- Confirm the TUI plan distinguishes backend kind, runtime isolation assurance, provisioning posture, and audit posture.
- Confirm audit drill-down is planned as typed broker-owned detail reads rather than direct ledger or storage access.
- Confirm live-activity expectations are framed around typed watch/event families rather than log-only heuristics.
- Confirm the MVP does not promise raw model chain-of-thought capture or display.
- Confirm the change clearly defers full multi-session and advanced power-user workspace management to the pre-MVP follow-on TUI change.

## Close Gate
Use the repository's standard verification flow before closing this change.
