# Tasks

## Shared Backend Contract Alignment

- [ ] Reuse the backend-neutral logical seams established by `CHG-2026-009-1672-launcher-microvm-backend-v0` where applicable, including launch intent, attachment planning, hardening posture recording, and terminal reporting.
- [ ] Keep `backend_kind` and runtime isolation assurance separate from audit posture and any backend-specific implementation evidence.
- [ ] Keep container runtime implementation details out of operator-facing run identity and public contracts.

Parallelization: should be agreed before container-specific implementation details expand; it depends on the finalized CHG-009 contract vocabulary.

## Opt-In UX + Audit

- [ ] Add an explicit “run with container backend” opt-in flow.
- [ ] Require an explicit user acknowledgment of reduced assurance.
- [ ] Record the opt-in, active `backend_kind`, runtime isolation assurance, and degraded posture in audit and shared broker run surfaces.

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

Parallelization: can be implemented in parallel with the microVM backend; coordinate on shared policy invariants and audit posture fields.

## No Host Mounts + Artifact Movement

- [ ] Maintain the same “no host filesystem mounts” rule.
- [ ] Provide artifacts/workspace state via explicit images/volumes that preserve the same data-movement semantics and logical attachment roles established by the shared `AttachmentPlan` model.

Parallelization: can be implemented in parallel with artifact store work; it depends on stable artifact attachment semantics.

## Policy Integration

- [ ] Ensure the policy engine blocks containers by default.
- [ ] Ensure microVM launch failures do not auto-trigger container mode.

Parallelization: can be implemented in parallel with policy engine and launcher; it depends only on explicit posture decisions (never implicit fallback).

## Acceptance Criteria

- [ ] Container mode cannot be enabled without an explicit recorded opt-in.
- [ ] The reduced assurance posture is unmissable in UX and audit.
- [ ] Role capabilities, attachment semantics, and artifact routing semantics remain consistent across backends.
- [ ] Deny-by-default egress is real (attempted outbound connections fail unless explicitly allowed and audited).
