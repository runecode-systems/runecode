## Summary
RuneCode deterministically allows or denies canonical action requests against a compiled effective policy context derived from signed manifests, referenced allowlists, and fixed invariants, with signed typed approval artifacts, explicit assurance levels, and a fixed hard floor for high-blast-radius operations.

## Problem
The high-level security posture for Policy Engine v0 is clear, but later broker, runner, gateway, TUI, and approval-profile features still need one shared contract for action identity, manifest composition, role taxonomy, gateway descriptors, approval scope, and reason-code ownership. Without those foundations, later work is likely to drift into parallel semantics and require a second policy rewrite.

## Proposed Change
- Freeze effective policy composition and `manifest_hash` semantics around one compiled policy context.
- Add a canonical typed `ActionRequest` model with a closed `action_kind` registry and typed payload families.
- Define layered role taxonomy and typed gateway destination/allowlist models that preserve RuneCode trust boundaries.
- Separate hard-floor assurance classes from approval trigger codes.
- Keep exact-action approval and stage sign-off as distinct signed approval shapes.
- Standardize policy decision details, reason-code ownership, audit binding, and the shared trusted Go package boundary for evaluation.

## Why Now
This work remains scheduled for v0.1.0-alpha.3, and freezing these policy foundations now prevents broker, runner, gateway, TUI, and later profile work from shipping incompatible interpretations of the same security model.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.

## Out of Scope
- Runtime implementation of the feature during this planning update.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
Establishes a stable policy substrate for the broker local API, workflow runner, workspace roles, gateway features, TUI approval UX, approval-profile expansion, and later formal-spec work without requiring a second semantics rewrite.
