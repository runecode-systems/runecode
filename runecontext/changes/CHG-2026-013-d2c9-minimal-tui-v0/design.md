# Design

## Overview
Implement the local TUI for runs, approvals, artifacts, and audit posture over the broker local API.

## Key Decisions
- TUI is a separate least-privilege client; it does not embed privileged execution.
- Use Bubble Tea as the TUI framework.
- The TUI must present backend posture as separate dimensions rather than flattening them into one overloaded “assurance” label:
  - `backend_kind` (`microvm`, later `container`)
  - runtime isolation assurance
  - provisioning/binding posture
  - audit posture
- The active approval profile is part of the user safety posture and should be visible and explained (MVP default: `moderate`).
- Approval requests must be explainable from structured data (reason codes + what changes if approved).
- The TUI must distinguish exact-action approvals from stage sign-off so it can explain what hash-bound work is blocked, what changed if a sign-off became stale, and what will actually be unblocked if approval is granted.
- TUI consumes the broker logical API object model directly and must not depend on daemon-private structs, local storage details, or scraped CLI output.
- Run browsing is built around first-class broker `RunSummary` and `RunDetail` read models so later CLI, remote, and concurrency work can reuse the same operator contract.
- TUI posture views must preserve the broker distinction between authoritative broker-derived state and optional runner advisory state.
- TUI explanation surfaces must keep `policy_reason_code`, `approval_trigger_code`, and system errors distinct rather than flattening them into one generic status string.
- Container reduced-assurance posture, TOFU-only provisioning posture, and degraded audit posture must remain visually distinct so users can tell what kind of degradation they are looking at.
- TUI should explain partial blocking and coordination waits from `RunDetail` coordination/stage/role surfaces rather than inferring a second lifecycle concept beyond the shared broker lifecycle vocabulary.
- TUI should present gate attempts, gate evidence, and gate overrides using the shared typed gate contract rather than log-only heuristics.
- TUI should surface canonical bound identities for run/stage/step/role scopes and gate attempts without promoting daemon-private identifiers to user authority.

## Main Workstreams
- Bubble Tea App Skeleton
- Core Screens (MVP)
- Local API Integration
- Safety UX

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
