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
- Bind backend posture changes to a first-class trusted `instance_id` minted by the running launcher instance so approvals for one runtime instance cannot be replayed against a later restarted instance.
- Keep the user/operator experience uniform across backends: the same TUI routes, approval review surfaces, audit inspection flows, and broker read models are used for microVM and container launches.
- Reuse the shared backend-neutral launcher contracts established by `CHG-2026-009-1672-launcher-microvm-backend-v0` and tighten them where they still assume microVM-specific transport, attachment, or evidence semantics.
- Require an exact explicit approval-bound backend posture change for reduced-assurance container opt-in; no silent fallback, no ambient UI setting, and no client-local truth.
- Add a dedicated instance-control policy path so backend posture actions are evaluated by the shared policy engine without pretending they are run-scoped actions.
- Add a broker-owned backend posture API family for reading current posture and requesting posture changes while keeping the broker local API as the only public trust-boundary contract.
- Add a real broker approval-issuance path that turns a backend-posture `require_human_approval` policy decision into a signed pending approval request persisted through the shared approval model.
- Repair approval resolution so `ApprovalResolve` remains one operation but accepts typed action-specific resolution detail instead of forcing promotion-shaped fields onto all approval kinds.
- Add a typed private broker-to-launcher posture-application contract that compare-and-set binds posture changes to the current `instance_id`.
- Scope container v0 to offline workspace-role launches only, with no host filesystem mounts, explicit artifact movement, real deny-by-default networking, and hardened container defaults.
- Implement a real container backend behind the shared launcher contract rather than stopping at policy/read-model scaffolding.
- Treat the reviewed reduction in runtime isolation assurance as the backend selection itself and treat missing container hardening baseline controls as launch failures, not as "more degraded but still allowed" runtime posture.
- Keep backend kind, runtime isolation assurance, provisioning/binding posture, audit posture, and backend-specific implementation evidence distinct in broker run surfaces, TUI posture surfaces, and audit payloads.
- Add generic exact-action approval issuance and consumption support so reduced-assurance backend opt-in does not require a container-specific TUI or broker approval-resolution flow.

## Why Now
This work remains scheduled for `v0.1.0-alpha.4`, and the roadmap's Alpha Implementation Callouts make the sequencing requirements explicit:
- the primary secure path must remain the real microVM-backed path
- the TUI must stay a strict client of the brokered local API and real approval/audit/policy surfaces
- follow-on hardening work like container mode must not displace the first honest secure end-to-end slice
- explicit artifact handoff must remain real instead of regressing to host-path shortcuts or bind mounts

Capturing those requirements in this change now makes the container feature build on the same reviewed foundation instead of retrofitting the right semantics after implementation has already spread through launcher, broker, policy, protocol, and TUI code.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- The broker local API remains the only public trust-boundary contract.
- MicroVM remains the preferred primary runtime-isolation boundary.
- For MVP, there is one running RuneCode instance per user per machine.
- Future vertical or horizontal scaling should preserve the same backend-selection semantics by binding them to explicit runtime-instance identity rather than to client-local preferences or per-run ad hoc flags.
- Container mode is ephemeral for the lifetime of the running RuneCode instance by default; restart returns to the preferred microVM posture unless a future reviewed feature explicitly adds durable operator policy.
- The launcher remains the runtime authority for realizing the selected backend posture; the broker remains the authority for policy evaluation, approval issuance/consumption, read-model truth, and audit emission.
- Generic exact-action approval workflows should remain shared across action kinds even when action-specific resolution detail differs.
- Linux-first container v0 may use a launcher-private OCI runtime adapter, but runtime implementation names remain private evidence rather than public control-plane vocabulary.
- Trusted control-plane components that are not realized through the launcher backend selection path remain outside this change's container-runtime scope.

## Out of Scope
- Making containers the default runtime-isolation backend.
- Automatic fallback from microVM to containers.
- Per-run, per-stage, or per-role backend mixing for this v0 change.
- Live migration of already-running isolates when the active instance backend posture changes.
- Gateway-role container launches, public egress, or other internet-facing container runtime scope in v0.
- Durable persisted operator preference for containers across launcher restart.
- Runtime-specific public APIs, direct TUI-to-launcher control paths, or a second public runtime control plane.
- Treating missing baseline container hardening controls as an acceptable degraded launch state.
- New public APIs, runner bypasses, or client-local approval/policy truth.
- Re-introducing legacy Agent OS planning paths as canonical references.

## Impact
This change becomes the authoritative plan for a container backend that is operationally consistent, explicitly reduced-assurance, non-replayable across runtime instances, and reusable as a foundation for later gateway-role container work, future scheduler-driven instance placement, and additional backend implementations without redefining core runtime posture, approval semantics, broker contracts, or launcher ownership.
