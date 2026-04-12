## Summary
Define broker-owned audit record drill-down surfaces so the first RuneCode TUI can inspect audit detail through typed reads instead of direct ledger access or shallow timeline-only views.

## Problem
The alpha TUI audit route needs deeper inspection than timeline plus verification summary, but that drill-down should not come from daemon-private ledger access or ad hoc file inspection.

## Proposed Change
- Define typed audit record drill-down reads.
- Define stable canonical record identities on timeline entries or related views.
- Define the minimum detail surface the alpha TUI needs for linked inspection.
- Keep audit drill-down broker-owned and derived-view-based rather than exposing private ledger structure.

## Why Now
This is a prerequisite for the alpha TUI audit experience to be a real inspection surface instead of a shallow timeline viewer.

## Assumptions
- Audit timeline and verification remain foundational and already exist as separate work.
- Audit drill-down must preserve the same audit truth model and not create a parallel UI-only interpretation layer.

## Out of Scope
- Direct ledger file access as a user-facing API.
- Remote audit transport changes.

## Impact
Creates the audit detail surface the alpha TUI should depend on so audit inspection remains typed, broker-owned, and aligned with audit boundaries.
