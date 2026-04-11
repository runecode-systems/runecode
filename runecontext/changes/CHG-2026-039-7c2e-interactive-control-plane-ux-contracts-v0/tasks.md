# Tasks

## Child Feature Tracking

- [ ] Track `CHG-2026-040-2b7f-session-transcript-model-v0` to completion.
- [ ] Track `CHG-2026-041-4d8a-approval-review-detail-models-v0` to completion.
- [ ] Track `CHG-2026-042-6f3c-audit-record-drill-down-v0` to completion.
- [ ] Track `CHG-2026-043-8e9b-live-activity-watch-streams-v0` to completion.

## Cross-Feature Coordination

- [ ] Keep the four child features aligned to the same broker/local-API contract posture.
- [ ] Keep the sequencing explicit that these contracts land before `CHG-2026-013-d2c9-minimal-tui-v0` implementation depends on them.
- [ ] Keep the rule explicit that the TUI should not invent local substitutes for these control-plane surfaces.

## Acceptance Criteria

- [ ] Child features remain linked, sequenced, and aligned as one prerequisite contract lane for the alpha TUI.
- [ ] Parent-project docs remain an integration view rather than duplicating child feature implementation details.
- [ ] The TUI lane can point to explicit broker/API prerequisite features instead of inferred follow-on work.
