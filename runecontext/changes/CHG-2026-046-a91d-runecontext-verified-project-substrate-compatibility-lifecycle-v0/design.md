# Design

## Overview
Make canonical RuneContext project state a first-class product substrate instead of an implicit assumption.

## Key Decisions
- Canonical project truth remains under `runecontext/` with `runecontext.yaml` at the repo root; there is no `.runecontext/` mirror and no daemon-private planning store.
- RuneCode must support both adopting existing compatible RuneContext state and initializing new compatible RuneContext state.
- Each RuneCode release should declare the RuneContext compatibility range it supports and should expose the active project compatibility posture through broker diagnostics.
- Hard compatibility enforcement for RuneCode-managed repos remains in RuneCode; RuneContext may still provide generic advisory warnings.
- Unsupported or non-verified project states are fail-closed normal-operation blocks with safe diagnostics and remediation flows only.
- Upgrade flows must be previewable, explicit, auditable, and compatible with direct RuneContext usage outside RuneCode.
- Run planning, verification, audit, attestation, and git-proof flows should bind to concrete RuneContext project state rather than to ambient local repository assumptions.

## Compatibility Lifecycle

- Discovery should identify:
  - presence or absence of canonical RuneContext state
  - verified-mode posture
  - active RuneContext version and compatibility range
  - blocked or degraded reasons when normal operation is unsafe
- Upgrade lifecycle should support:
  - inspect current posture
  - preview compatible upgrade actions
  - apply reviewed upgrade steps
  - re-run validation and assurance checks
- Broker version and readiness surfaces should report both product compatibility policy and active project posture.

## Main Workstreams
- Project Discovery, Adoption, and Initialization.
- Compatibility Policy + Version Reporting.
- Upgrade + Remediation Lifecycle.
- Assurance and Verification Binding.
- Broker/TUI/CLI Diagnostics and Blocked-State UX.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
