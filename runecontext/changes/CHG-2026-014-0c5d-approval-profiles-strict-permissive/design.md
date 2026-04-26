# Design

## Overview
Define post-MVP approval profiles that adjust approval timing without weakening core policy invariants or trust boundaries.

## Key Decisions
- Approval profiles affect *when* an allowed action requires explicit human approval; they must not change isolation boundaries or weaken invariants.
- Approval profiles govern formal approval timing and assurance only; operator-question frequency and autonomous continuation posture belong to shared `autonomy_posture`, not the approval profile.
- Profiles must never convert `deny -> allow`.
- Profiles are signed inputs (part of the run/stage capability manifest) and are fully auditable.
- Adding or constraining profile values is a schema-versioned protocol change for every object family that carries the enum.
- MVP ships with `moderate` only; `strict` and `permissive` are post-MVP extensions.
- Profiles are user-involvement presets for ordinary actions: they map approval frequency, batching, TTL, and minimum assurance, but they do not define transport trust or channel semantics.
- Profiles cannot lower the fixed hard-floor categories defined by policy; those continue to require their minimum assurance regardless of selected profile.
- Profiles should operate over canonical policy action kinds and hard-floor classes rather than ad hoc UI-facing action labels so later policy, runner, broker, and TUI work reuse one action taxonomy.
- Delivery channel remains non-authoritative; local TUI, remote TUI, and messaging delivery are interchangeable only when they converge on the same signed approval artifacts and assurance checks.
- Any broker-exposed run or approval summaries that surface the active profile or required assurance are part of the shared logical API contract and must evolve consistently as new profile values are added.
- Profiles must preserve the shared approval split between exact-action approvals and stage sign-off rather than collapsing them into one profile-local approval type.
- Profiles must preserve the shared distinction between `waiting_approval` and `waiting_operator_input`; ordinary operator guidance must not be relabeled as formal approval, and profile tuning must not hide required formal approvals.
- Gate overrides remain exact explicit approvals across all profiles.
- `git_remote_ops` and any equivalent future remote-state-mutation hard-floor classes remain exact explicit approvals across all profiles.
- Dependency-fetch checkpoint semantics remain shared across all profiles; profiles may tune timing or batching only within the reviewed scope-change model and may not invent a competing dependency-local approval taxonomy.
- Profiles must not batch, defer, or replace final git remote-mutation approval with stage sign-off, milestone approval, or ambient session acknowledgment.
- Git remote-mutation approval payloads must remain bound to canonical repository identity, target refs, referenced patch artifact digests, expected result tree hash, and canonical action request hash.
- Profiles may not lower the minimum assurance floor for git remote mutation below the reviewed baseline, which remains at least `reauthenticated`.
- Profile expansion must respect the shared role-kind versus executor-class model and must not silently let `workspace-test` or other ordinary workspace roles inherit `system_modifying` behavior.
- Profile expansion must also respect the shared dependency-fetch split:
  - dependency scope enablement or expansion remains the approval-bearing checkpoint surface
  - ordinary `fetch_dependency` work inside already-approved scope must not be relabeled as a new per-cache-miss approval model by profile-specific UX or policy mappings
- Reduced-assurance backend selection, including explicit `container mode` opt-in, remains an exact-action approval across all profiles and must not become a stage-sign-off, batched ambient acknowledgment, or durable backend preference.
- Approval profiles may tune ordinary approval timing and assurance where policy allows, but they must not weaken the minimum assurance or exact binding required for reduced-assurance backend posture changes.
- Approval profiles must not weaken or override blocked project-substrate posture; diagnostics/remediation-only behavior for missing, invalid, non-verified, or unsupported repository substrate is not profile-controlled.

## Main Workstreams
- Approval Profile Model (Post-MVP)
- Strict Profile Semantics
- Permissive Profile Semantics
- Policy + Runner + TUI Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
