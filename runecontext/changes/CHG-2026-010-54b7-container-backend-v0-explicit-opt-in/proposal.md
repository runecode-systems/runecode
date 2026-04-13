## Summary
RuneCode can run workspace-role isolation through an explicit reduced-assurance `container mode` for the active running RuneCode instance, while preserving the same broker, policy, audit, artifact, and TUI experience used by the primary microVM path.

## Problem
The project needs a reduced-assurance container backend that remains useful without weakening RuneCode's core tenets.

Without a tighter foundation, container support would likely drift into one or more bad outcomes:
- a second backend-specific UX or approval flow
- a container-only control contract that forks launcher semantics from the microVM path
- stringly typed policy and posture logic that does not scale to future backends or future instance scheduling
- ambiguous audit/runtime posture where reduced assurance, implementation detail, and audit verification get collapsed together
- ad hoc operator settings that bypass exact-action approval and durable audit evidence

## Proposed Change
- Define `container mode` as an instance-scoped backend posture for the active running RuneCode instance rather than a per-run, per-stage, or per-role toggle.
- Keep the user/operator experience uniform across backends: the same TUI routes, approval review surfaces, audit inspection flows, and broker read models are used for microVM and container launches.
- Reuse the shared backend-neutral launcher contracts established by `CHG-2026-009-1672-launcher-microvm-backend-v0` and tighten them where they still assume microVM-specific transport, attachment, or evidence semantics.
- Require an exact explicit approval-bound backend posture change for reduced-assurance container opt-in; no silent fallback and no ambient UI setting.
- Scope container v0 to offline workspace-role launches only, with no host filesystem mounts, explicit artifact movement, real deny-by-default networking, and hardened container defaults.
- Keep backend kind, runtime isolation assurance, provisioning/binding posture, audit posture, and backend-specific implementation evidence distinct in broker run surfaces, TUI posture surfaces, and audit payloads.
- Add generic exact-action approval-consumption support so reduced-assurance backend opt-in does not require a container-specific TUI or broker approval-resolution flow.

## Why Now
This work remains scheduled for `v0.1.0-alpha.4`, and the roadmap's Alpha Implementation Callouts make the sequencing requirements explicit:
- the primary secure path must remain the real microVM-backed path
- the TUI must stay a strict client of the brokered local API and real approval/audit/policy surfaces
- follow-on hardening work like container mode must not displace the first honest secure end-to-end slice

Capturing those requirements in this change now makes the container feature build on the same reviewed foundation instead of retrofitting the right semantics after implementation has already spread through launcher, broker, policy, protocol, and TUI code.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- The broker local API remains the only public trust-boundary contract.
- MicroVM remains the preferred primary runtime-isolation boundary.
- For MVP, there is one running RuneCode instance per user per machine.
- Future vertical or horizontal scaling should preserve the same backend-selection semantics by binding them to explicit runtime-instance identity rather than to client-local preferences or per-run ad hoc flags.
- Container mode is ephemeral for the lifetime of the running RuneCode instance by default; restart returns to the preferred microVM posture unless a future reviewed feature explicitly adds durable operator policy.
- Trusted control-plane components that are not realized through the launcher backend selection path remain outside this change's container-runtime scope.

## Out of Scope
- Making containers the default runtime-isolation backend.
- Automatic fallback from microVM to containers.
- Per-run, per-stage, or per-role backend mixing for this v0 change.
- Live migration of already-running isolates when the active instance backend posture changes.
- Gateway-role container launches, public egress, or other internet-facing container runtime scope in v0.
- New public APIs, runner bypasses, or client-local approval/policy truth.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
This change becomes the authoritative plan for a container backend that is operationally consistent, explicitly reduced-assurance, and reusable as a foundation for later gateway-role container work, future scheduler-driven instance placement, and additional backend implementations without redefining core runtime posture or approval semantics.
