# Tasks

## Instance-Scoped Backend Selection Model

- [ ] Define `container mode` as the backend posture for the active running RuneCode instance in MVP, not as a per-run, per-stage, or per-role toggle.
- [ ] Make the active launcher runtime instance a first-class trusted identity by minting a stable `instance_id` for the lifetime of the running instance.
- [ ] Bind canonical backend posture actions to `target_instance_id` so approvals for one instance cannot be replayed against a later restarted instance.
- [ ] Extend typed approval detail and posture evidence to carry `target_instance_id` while keeping exact action hash as the trust root.
- [ ] Keep instance-scoped selection topology-neutral so future vertically or horizontally scaled deployments can preserve the same semantics without changing the public control-plane model.
- [ ] Ensure instance backend posture changes affect compatible future launches only and do not imply live migration of already-running isolates.
- [ ] Ensure restart returns to the preferred primary microVM posture unless a later explicitly planned feature adds reviewed durable operator policy for backend preference.

Parallelization: should be agreed before TUI, broker, and launcher implementation work expands; it is foundational to approval binding, operator semantics, replay safety, and future scheduler alignment.

## Instance-Control Policy Path

- [ ] Add a dedicated broker instance-control policy evaluation path for backend posture changes instead of treating them as ordinary run-scoped runtime actions.
- [ ] Reuse the shared policy engine, canonical `ActionRequest` hashing, and signed trusted-context discipline for instance-control actions.
- [ ] If new trusted context artifacts are needed, keep them generic to control-plane instance posture rather than container-specific.
- [ ] Ensure backend-posture policy decisions remain explicit about denied fallback attempts, explicit opt-in, and reduced-assurance semantics.

Parallelization: should be implemented before approval issuance and broker local API posture work; it is foundational to keeping backend posture selection honest and trust-boundary-correct.

## Shared Backend Contract Alignment

- [ ] Reuse the backend-neutral logical seams established by `CHG-2026-009-1672-launcher-microvm-backend-v0` where applicable, including launch intent, attachment planning, hardening posture recording, and terminal reporting.
- [ ] Keep `backend_kind` and runtime isolation assurance separate from audit posture and any backend-specific implementation evidence.
- [ ] Keep container runtime implementation details out of operator-facing run identity and public contracts.
- [ ] Review the shared runtime vocabulary for microVM-colored assumptions and tighten it where needed so container semantics do not get mislabeled as degraded or unknown only because a field is microVM-specific.
- [ ] Distinguish backend-neutral operator posture, backend-neutral runtime evidence, and backend-specific implementation evidence.
- [ ] Ensure shared contracts can represent `degraded`, `unknown`, and `not_applicable` distinctly where backend mechanics differ.

Parallelization: should be agreed before container-specific implementation details expand; it depends on the finalized CHG-009 contract vocabulary.

## Approval Issuance + Generic Approval Resolve Shape

- [ ] Add a production broker path that turns backend-posture `require_human_approval` policy decisions into canonical signed `ApprovalRequest` envelopes plus persisted pending approval records.
- [ ] Ensure backend-posture approval requests describe the reduced-assurance effect, target backend kind, target instance, and “future launches only” semantics through typed detail.
- [ ] Keep approval issuance broker-owned; clients must not synthesize unsigned approval requests or pending approval records.
- [ ] Refactor `ApprovalResolve` so common fields remain shared while action-specific resolution detail moves into typed detail objects rather than promotion-specific top-level fields.
- [ ] Ensure backend-posture approval resolution verifies signed request/decision artifacts, applies the approved posture through the private launcher contract, and only then marks the approval consumed.
- [ ] Ensure backend-posture approvals fail closed when the target `instance_id` no longer matches the running launcher instance.

Parallelization: should be agreed before container opt-in UX implementation; it depends on stable approval detail/read-model semantics rather than container runtime code.

## Backend Posture Action Contract

- [ ] Tighten the backend posture action payload away from freeform posture strings toward closed semantics for target instance, target backend selection, fallback attempt posture, and assurance change kind.
- [ ] Ensure explicit opt-in remains represented through canonical action identity + approval binding rather than through a sticky UI toggle or ambient client flag.
- [ ] Ensure automatic fallback attempts remain representable so policy can deny and audit them deterministically without fragile string matching.

Parallelization: should be implemented before policy logic and TUI approval review depend on expanded backend-selection semantics.

## Broker Local API For Backend Posture

- [ ] Add a dedicated broker local API read operation for current backend posture that returns active `instance_id`, current backend, pending approval state, and backend availability.
- [ ] Add a dedicated broker local API write operation for backend posture changes that can return denied, pending-approval-created, or applied outcomes.
- [ ] Keep backend posture operations generic to future backend choices rather than container-specific.
- [ ] Ensure the broker local API remains the only public trust-boundary contract; TUI must not call launcher directly.

Parallelization: depends on the approval issuance path and typed backend posture request semantics.

## Private Broker-To-Launcher Posture Contract

- [ ] Add a typed private launcher posture-application operation bound to `instance_id` via compare-and-set semantics.
- [ ] Add a typed private launcher posture-query operation so broker restart can recover current posture while the same launcher instance remains running.
- [ ] Ensure launcher-applied posture records carry stable references back to the approved action and approval evidence.
- [ ] Keep transport realization private to the trusted domain and do not turn posture application into a second public API.

Parallelization: can proceed in parallel with broker local API work once instance identity and action binding are settled.

## Opt-In UX + Audit

- [ ] Add an explicit instance-scoped “switch active instance to container mode” opt-in flow.
- [ ] Require an explicit user acknowledgment of reduced assurance.
- [ ] Ensure the operator-facing TUI flow remains otherwise the same across backends; backend choice is surfaced through shared posture and approval/read-model cues rather than a container-specific execution flow.
- [ ] Route opt-in through the broker local API posture operations and the shared approvals route rather than through direct launcher calls or TUI-local state.
- [ ] Ensure successful approval resolution refreshes the same shared status/run/read-model surfaces rather than sending the user through a separate backend-specific route.
- [ ] Record the opt-in, active `backend_kind`, runtime isolation assurance, degraded posture, and bound approval/policy references in audit and shared broker run surfaces.
- [ ] Surface `container mode` as an unmissable reduced-assurance posture in the same run safety/read-detail surfaces used by the primary microVM path.

Parallelization: can be implemented in parallel with TUI work; it depends on stable approval/audit event schemas.

## Container Runtime Controller

- [ ] Implement a real container controller behind the shared launcher contract instead of stopping at policy/read-model scaffolding.
- [ ] Route backend launches by requested backend kind without allowing silent substitution between microVM and container implementations.
- [ ] Keep container v0 Linux-first and limited to offline workspace-role launches.
- [ ] Keep runtime implementation names private to launcher evidence rather than public control-plane vocabulary.

Parallelization: should start only after shared backend contract alignment and private posture application seams are explicit.

## Hardened Container Baseline

- [ ] Treat the following as required admission controls rather than optional degraded posture for successful launch:
  - rootless execution
  - `no_new_privs`
  - seccomp + dropped Linux capabilities
  - read-only root filesystem + ephemeral writable layers
  - deny-by-default egress for workspace-role container v0
- [ ] Specify concrete networking enforcement (MVP):
  - run each role in its own network namespace
  - default: no network connectivity (or loopback only)
  - if egress is explicitly granted, enforce via explicit host-level rules (firewall/proxy allowlists), not in-container configuration
- [ ] Ensure the isolation boundary is represented as “container (reduced assurance)” in UI/logs.
- [ ] Keep the initial container v0 runtime scope to offline workspace-role launches only; do not expand initial scope to gateway-role container runtime support.
- [ ] Record the effective hardening posture as actually enforced, including degraded reasons and backend-specific evidence refs where relevant.

Parallelization: can be implemented in parallel with container runtime code; coordinate on shared policy invariants and audit posture fields.

## No Host Mounts + Artifact Movement

- [ ] Maintain the same “no host filesystem mounts” rule.
- [ ] Provide artifacts/workspace state via explicit images/volumes that preserve the same data-movement semantics and logical attachment roles established by the shared `AttachmentPlan` model.
- [ ] Keep broker authorization at logical attachment role + digest identity only; backend-private realization must not leak host-local paths or mount identities through shared contracts.
- [ ] Review attachment/channel vocabulary so container realization does not require misleading microVM-only shared channel semantics.
- [ ] Prefer launcher-private named volumes, image layers, tmpfs, or equivalent realization primitives instead of host bind mounts.
- [ ] Ensure host-local paths, mount identities, and runtime-private storage layout never appear in public schemas, run surfaces, audit payloads, or degraded-reason strings.

Parallelization: can be implemented in parallel with artifact store work; it depends on stable artifact attachment semantics.

## Policy Integration

- [ ] Ensure the policy engine blocks containers by default.
- [ ] Ensure microVM launch failures do not auto-trigger container mode.
- [ ] Ensure instance-scoped container mode selection is represented as a canonical backend posture action and evaluated by shared policy logic.
- [ ] Ensure reduced-assurance backend opt-in remains an exact-action approval under the shared approval model.
- [ ] Ensure workspace-role offline invariants remain intact when container mode is active.

Parallelization: can be implemented in parallel with policy engine and launcher; it depends only on explicit posture decisions (never implicit fallback).

## Audit And Run-Surface Alignment

- [ ] Project reduced-assurance container posture through shared broker `RunSummary` and `RunDetail` surfaces without introducing a second backend-specific operator model.
- [ ] Ensure `RunDetail.authoritative_state` can reference the bound approval/policy evidence for reduced-assurance backend selection.
- [ ] Keep audit/runtime posture distinct from backend kind and runtime isolation assurance.
- [ ] Keep backend-specific provenance in typed implementation evidence or evidence refs rather than in public run identity.
- [ ] Add a broker-owned audit event for backend posture application itself, separate from later run launches.
- [ ] Ensure runtime launch evidence for runs started under container mode carries a stable reference back to the applied posture-selection evidence.

Parallelization: can be implemented in parallel with broker read-model work; it depends on stable runtime evidence and approval reference semantics.

## Conformance And End-To-End Verification

- [ ] Add backend conformance checks that assert no automatic fallback, posture separation, evidence persistence, and absence of host-path leakage across backend implementations.
- [ ] Add end-to-end verification that container opt-in produces a real pending approval, binds to `target_instance_id`, and applies only to future compatible launches.
- [ ] Add verification that launcher restart mints a new `instance_id`, resets posture to microVM, and prevents replay of old backend-posture approvals.
- [ ] Add verification that container launches fail closed when required baseline hardening controls are unavailable.
- [ ] Add verification that outbound connections fail for workspace-role container runs unless later explicitly allowed by reviewed scope.
- [ ] Add verification that workspace and artifact realization uses no host filesystem bind mounts.

Parallelization: depends on the runtime, broker, and audit workstreams being in place; it is required before this change can be closed.

## Acceptance Criteria

- [ ] Container mode cannot be enabled without an explicit recorded opt-in.
- [ ] Container mode approval is bound to the active launcher `instance_id` and cannot be replayed after launcher restart.
- [ ] The reduced assurance posture is unmissable in UX and audit.
- [ ] Role capabilities, attachment semantics, and artifact routing semantics remain consistent across backends.
- [ ] Deny-by-default egress is real (attempted outbound connections fail unless explicitly allowed and audited).
- [ ] Container mode selection is bound to the active running RuneCode instance in MVP and does not require per-run or per-role UX branching.
- [ ] TUI/operator interaction remains the same across backends apart from shared posture, approval, and audit cues.
- [ ] Reduced-assurance backend opt-in is issued and consumed through a generic exact-action approval path rather than through a container-specific broker/TUI flow.
- [ ] The initial container v0 scope is limited to offline workspace-role launches and does not dilute role-family separation.
- [ ] Missing required container hardening baseline controls cause launch failure rather than a weaker successful launch.
- [ ] No host filesystem bind mounts are used for container workspace or artifact realization.
- [ ] Shared launcher/broker contracts remain reusable by future gateway-role container work and future scaled instance placement without redefining core runtime posture semantics.
