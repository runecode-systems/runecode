# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Standards inventory:** `runecontext/project/standards-inventory.md`
- **Trust boundaries:** `docs/trust-boundaries.md`

## UX And Implementation References

- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Bubbles: https://github.com/charmbracelet/bubbles
- Lip Gloss: https://github.com/charmbracelet/lipgloss
- Crush: https://github.com/charmbracelet/crush
- OpenCode: https://github.com/anomalyco/opencode
- Tips for building Bubble Tea programs: https://leg100.github.io/en/posts/building-bubbletea-programs/
- Hamburger Menus and Hidden Navigation Hurt UX Metrics: https://www.nngroup.com/articles/hamburger-menus/
- Designing for Progressive Disclosure: https://www.uxmatters.com/mt/archives/2020/05/designing-for-progressive-disclosure.php

## Related Changes

- `runecontext/changes/CHG-2026-013-d2c9-minimal-tui-v0/`
- `runecontext/changes/CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0/`
- `runecontext/changes/CHG-2026-040-2b7f-session-transcript-model-v0/`
- `runecontext/changes/CHG-2026-041-4d8a-approval-review-detail-models-v0/`
- `runecontext/changes/CHG-2026-042-6f3c-audit-record-drill-down-v0/`
- `runecontext/changes/CHG-2026-043-8e9b-live-activity-watch-streams-v0/`
- `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`
- `runecontext/changes/CHG-2026-003-b567-audit-log-v0-verify/`
- `runecontext/changes/CHG-2026-006-84f0-audit-anchoring-v0/`
- `runecontext/changes/CHG-2026-007-2315-policy-engine-v0/`
- `runecontext/changes/CHG-2026-011-7240-secretsd-model-gateway-v0/`
- `runecontext/changes/CHG-2026-014-0c5d-approval-profiles-strict-permissive/`
- `runecontext/changes/CHG-2026-019-40c5-bridge-runtime-protocol-v0/`

## Planning Notes

- Visual direction was informed by fullscreen workbench references reviewed during planning, especially OpenCode-style compositions with centered overlays, strong focus cues, integrated output surfaces, and a calm dark palette.
- Follow-up architectural analysis also reviewed Charmbracelet `crush` directly. The durable conclusion was to borrow its bounded-component rendering discipline, reusable list and overlay patterns, and layout-test posture, but not its root screen-buffer architecture.
- User-local screenshot filesystem paths are intentionally not recorded in repo docs; only the durable design conclusions are captured here.
- This change intentionally freezes the shell, navigation, activity, and copy or paste foundation before the larger visual pass begins.
