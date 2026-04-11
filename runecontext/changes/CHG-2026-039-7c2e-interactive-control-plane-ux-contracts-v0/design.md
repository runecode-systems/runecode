# Design

## Overview
Use this change as the project-level tracker for broker/API-side UX contracts that the first TUI feature should depend on before implementation.

The parent project owns sequencing and shared contract posture. The child features own the concrete model surfaces.

## Key Decisions
- Child features own runtime contract detail.
- Parent project owns sequencing and integration posture.
- The alpha TUI should depend on explicit broker/API UX contracts rather than approximating them locally.
- Session, approval review, audit drill-down, and live watch surfaces are separate feature boundaries.
- The contract project is a prerequisite lane for `CHG-2026-013-d2c9-minimal-tui-v0`.

## Main Workstreams
- `CHG-2026-040-2b7f-session-transcript-model-v0`
- `CHG-2026-041-4d8a-approval-review-detail-models-v0`
- `CHG-2026-042-6f3c-audit-record-drill-down-v0`
- `CHG-2026-043-8e9b-live-activity-watch-streams-v0`

## Cross-Feature Outcomes To Preserve
- One canonical session/transcript model rather than client-local chat truth.
- One broker-projected approval review detail model rather than client payload scraping.
- One broker-owned audit drill-down model rather than direct ledger access.
- One typed live-activity watch foundation rather than polling-plus-logs as the primary live operator surface.
