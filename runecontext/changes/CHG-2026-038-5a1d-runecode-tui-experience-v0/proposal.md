## Summary
Track the RuneCode TUI lane as one project-level plan covering the alpha.3 hybrid dashboard-plus-chat foundation and the contract-first sequencing needed to keep the terminal experience aligned with strict broker, approval, audit, and trust-boundary semantics.

## Problem
TUI work now spans more than one kind of change:
- the first user-facing TUI foundation in `CHG-2026-013-d2c9-minimal-tui-v0`
- prerequisite broker/API contract work that should land before that first implementation

Without a project-level umbrella, the TUI lane is harder to review as one coherent effort, and foundational backend contract work risks looking like optional follow-ons rather than required prerequisites.

## Proposed Change
- Keep this change as the TUI parent project.
- Track `CHG-2026-013-d2c9-minimal-tui-v0` as the first implementation feature.
- Track `CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0` as the project for the broker/API-side contract work that must be implemented first.
- Keep cross-feature sequencing and verification visible in one place.
- Freeze the shared TUI-lane foundation before implementation breadth expands:
  - the TUI is a strict broker client, not a shortcut control plane
  - the first TUI is a hybrid shell, not only an operator console
  - broker/API UX contracts land before the TUI depends on them
  - advanced TUI work remains a later separate feature lane rather than being folded into the alpha foundation

## Why Now
The TUI lane now has enough structure that it should be reviewed as a project rather than a single feature. Creating this umbrella now makes the intended sequencing explicit: contract work first, alpha TUI foundation second, later advanced TUI work after that.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode owns the end-user UX and command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Verified-mode RuneContext remains the normal operating assumption for this repository.

## Out of Scope
- Implementing runtime behavior directly in this umbrella change.
- Folding the later `CHG-2026-037-91be-tui-multi-session-power-workspace-v0` feature into this umbrella; that later feature remains separately tracked.
- Re-introducing legacy planning paths as canonical references.

## Impact
Creates one project-level anchor for the alpha TUI lane so the first implementation feature and its prerequisite broker/API contract work stay explicitly linked, sequenced, and reviewable.
