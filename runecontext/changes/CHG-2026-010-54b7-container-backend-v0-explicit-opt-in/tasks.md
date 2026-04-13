# Tasks

## Instance-Scoped Backend Selection Model

- [ ] Define `container mode` as the backend posture for the active running RuneCode instance in MVP, not as a per-run, per-stage, or per-role toggle.
- [ ] Keep instance-scoped selection topology-neutral so future vertically or horizontally scaled deployments can preserve the same semantics without changing the public control-plane model.
- [ ] Ensure instance backend posture changes affect compatible future launches only and do not imply live migration of already-running isolates.
- [ ] Ensure restart returns to the preferred primary microVM posture unless a later explicitly planned feature adds reviewed durable operator policy for backend preference.

Parallelization: should be agreed before TUI, broker, and launcher implementation work expands; it is foundational to approval binding, operator semantics, and future scheduler alignment.

## Shared Backend Contract Alignment

- [ ] Reuse the backend-neutral logical seams established by `CHG-2026-009-1672-launcher-microvm-backend-v0` where applicable, including launch intent, attachment planning, hardening posture recording, and terminal reporting.
- [ ] Keep `backend_kind` and runtime isolation assurance separate from audit posture and any backend-specific implementation evidence.
- [ ] Keep container runtime implementation details out of operator-facing run identity and public contracts.
- [ ] Review the shared runtime vocabulary for microVM-colored assumptions and tighten it where needed so container semantics do not get mislabeled as degraded or unknown only because a field is microVM-specific.
- [ ] Distinguish backend-neutral operator posture, backend-neutral runtime evidence, and backend-specific implementation evidence.
- [ ] Ensure shared contracts can represent `degraded`, `unknown`, and `not_applicable` distinctly where backend mechanics differ.

Parallelization: should be agreed before container-specific implementation details expand; it depends on the finalized CHG-009 contract vocabulary.

## Generic Exact-Action Approval Consumption

- [ ] Define a generic exact-action approval consumption path for backend posture actions so reduced-assurance container opt-in does not require a container-specific broker or TUI flow.
- [ ] Keep approval consumption trust rooted in signed approval request/decision artifacts bound to the canonical backend posture action hash.
- [ ] Ensure approval review surfaces can explain reduced-assurance backend selection through typed broker-projected detail rather than client-local payload scraping.
- [ ] Keep exact-action approval and stage sign-off semantics distinct.

Parallelization: should be agreed before container opt-in UX implementation; it depends on stable approval detail/read-model semantics rather than container runtime code.

## Backend Posture Action Contract

- [ ] Tighten the backend posture action payload away from freeform posture strings toward closed semantics for target backend selection, fallback attempt posture, and assurance change kind.
- [ ] Ensure explicit opt-in remains represented through canonical action identity + approval binding rather than through a sticky UI toggle or ambient client flag.
- [ ] Ensure automatic fallback attempts remain representable so policy can deny and audit them deterministically without fragile string matching.

Parallelization: should be implemented before policy logic and TUI approval review depend on expanded backend-selection semantics.

## Opt-In UX + Audit

- [ ] Add an explicit instance-scoped “switch active instance to container mode” opt-in flow.
- [ ] Require an explicit user acknowledgment of reduced assurance.
- [ ] Ensure the operator-facing TUI flow remains otherwise the same across backends; backend choice is surfaced through shared posture and approval/read-model cues rather than a container-specific execution flow.
- [ ] Record the opt-in, active `backend_kind`, runtime isolation assurance, degraded posture, and bound approval/policy references in audit and shared broker run surfaces.
- [ ] Surface `container mode` as an unmissable reduced-assurance posture in the same run safety/read-detail surfaces used by the primary microVM path.

Parallelization: can be implemented in parallel with TUI work; it depends on stable approval/audit event schemas.

## Hardened Container Baseline

- [ ] Define MVP hardening targets:
  - rootless where possible
  - seccomp + dropped Linux capabilities
  - read-only root filesystem + ephemeral writable layers
  - deny-by-default egress (unless the role is a gateway role)
- [ ] Specify concrete networking enforcement (MVP):
  - run each role in its own network namespace
  - default: no network connectivity (or loopback only)
  - if egress is explicitly granted, enforce via explicit host-level rules (firewall/proxy allowlists), not in-container configuration
- [ ] Ensure the isolation boundary is represented as “container (reduced assurance)” in UI/logs.
- [ ] Keep the initial container v0 runtime scope to offline workspace-role launches only; do not expand initial scope to gateway-role container runtime support.
- [ ] Record the effective hardening posture as actually enforced, including degraded reasons and backend-specific evidence refs where relevant.

Parallelization: can be implemented in parallel with the microVM backend; coordinate on shared policy invariants and audit posture fields.

## No Host Mounts + Artifact Movement

- [ ] Maintain the same “no host filesystem mounts” rule.
- [ ] Provide artifacts/workspace state via explicit images/volumes that preserve the same data-movement semantics and logical attachment roles established by the shared `AttachmentPlan` model.
- [ ] Keep broker authorization at logical attachment role + digest identity only; backend-private realization must not leak host-local paths or mount identities through shared contracts.
- [ ] Review attachment/channel vocabulary so container realization does not require misleading microVM-only shared channel semantics.

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

Parallelization: can be implemented in parallel with broker read-model work; it depends on stable runtime evidence and approval reference semantics.

## Acceptance Criteria

- [ ] Container mode cannot be enabled without an explicit recorded opt-in.
- [ ] The reduced assurance posture is unmissable in UX and audit.
- [ ] Role capabilities, attachment semantics, and artifact routing semantics remain consistent across backends.
- [ ] Deny-by-default egress is real (attempted outbound connections fail unless explicitly allowed and audited).
- [ ] Container mode selection is bound to the active running RuneCode instance in MVP and does not require per-run or per-role UX branching.
- [ ] TUI/operator interaction remains the same across backends apart from shared posture, approval, and audit cues.
- [ ] Reduced-assurance backend opt-in is consumed through a generic exact-action approval path rather than through a container-specific broker/TUI flow.
- [ ] The initial container v0 scope is limited to offline workspace-role launches and does not dilute role-family separation.
- [ ] Shared launcher/broker contracts remain reusable by future gateway-role container work and future scaled instance placement without redefining core runtime posture semantics.
