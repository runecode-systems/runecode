## Summary
Track the repository-wide migration from `agent-os/` planning artifacts to canonical RuneContext artifacts under `runecontext/`.

## Problem
Cross-cutting migration assumptions currently exist mostly in temporary migration notes rather than durable RuneContext decisions.
Without early canonical decisions and a single umbrella tracking change, later feature-folder migration can drift, duplicate assumptions, or require a second semantic rewrite.

## Proposed Change
- Capture and commit cross-cutting migration decisions as durable records under `runecontext/decisions/`.
- Keep this umbrella change as the reviewable tracker for migration sequencing, tasks, and verification checkpoints.
- Ensure future migration phases reference these decisions as canonical context.

## Why Now
Phase 1 must land before broad product, standards, specs, and changes migration so later artifacts can migrate directly to their final RuneContext-era meaning.

## Assumptions
- RuneContext foundational preflight (Phase 0) remains valid for this repository.
- Decision files under `runecontext/decisions/` are valid durable targets in the current RuneContext contract.
- Follow-on phases will rewrite content toward the RuneContext final state rather than preserve Agent OS assumptions for later cleanup.

## Out of Scope
- Bulk migration of `agent-os/specs/` content.
- Final legacy deletion of remaining `agent-os/` canonical material outside completed product-doc and standards migration scope.

## Impact
Creates canonical migration governance context so subsequent migration batches are consistent, reviewable, and directly aligned with final RuneContext-era planning assumptions.
