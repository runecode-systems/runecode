# Design

## Overview
Use this change as the project-level tracker for the RuneCode TUI lane while implementation and contract work land in child feature changes.

The parent project owns sequencing and shared integration posture. It exists to make sure the TUI lane grows in the right order: contract-first, then alpha implementation, then later advanced TUI work in a separate feature lane.

## Key Decisions
- Child features own runtime implementation detail.
- Parent project owns sequencing and integration posture.
- The TUI lane must remain aligned with strict broker, policy, approval, audit, and trust-boundary semantics.
- The first TUI implementation feature is `CHG-2026-013-d2c9-minimal-tui-v0`.
- The broker/API-side UX contracts needed before that implementation are tracked through `CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0`.
- `CHG-2026-037-91be-tui-multi-session-power-workspace-v0` remains a separate later feature lane and is not part of this umbrella.
- The TUI lane should not justify weakening core semantics for MVP speed.

## Main Workstreams
- `CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0`
- `CHG-2026-013-d2c9-minimal-tui-v0`
- Cross-feature sequencing and verification

## Cross-Feature Outcomes To Preserve
- One contract-first TUI foundation rather than a UI that invents missing backend semantics locally.
- One strict rule that the TUI consumes broker-visible typed contracts instead of daemon-private data and CLI scraping.
- One explicit sequencing rule that broker/API UX contracts land before the first TUI implementation depends on them.
- One clear boundary between the alpha foundation and the later advanced TUI workbench feature.
