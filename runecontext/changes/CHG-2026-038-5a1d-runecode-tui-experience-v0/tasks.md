# Tasks

## Child Feature Tracking

- [ ] Track `CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0` to completion.
- [ ] Track `CHG-2026-013-d2c9-minimal-tui-v0` to completion.

## Cross-Feature Coordination

- [ ] Keep the broker/API contract work explicitly sequenced before the first TUI implementation feature.
- [ ] Keep the TUI lane aligned with `CHG-2026-008-62e1-broker-local-api-v0`, `CHG-2026-007-2315-policy-engine-v0`, and audit work without duplicating those foundation changes.
- [ ] Keep the rule explicit that the alpha TUI must not invent local approval, audit, session, or live-activity semantics to cover missing backend contracts.
- [ ] Keep the later `CHG-2026-037-91be-tui-multi-session-power-workspace-v0` feature separate from this umbrella and sequenced after the alpha foundation.

## Acceptance Criteria

- [ ] Child features remain linked, sequenced, and aligned to the same TUI-lane invariants.
- [ ] Parent-project docs remain an integration view rather than duplicating feature implementation detail.
- [ ] The TUI lane is planned in a way that prevents foundational rework caused by implementing the alpha UI before its required broker/API UX contracts exist.
