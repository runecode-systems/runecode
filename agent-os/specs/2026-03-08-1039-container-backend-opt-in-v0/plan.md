# Container Backend v0 (Explicit Opt-In)

User-visible outcome: RuneCode can run roles in a hardened container isolation mode when explicitly opted in, while clearly surfacing and auditing the reduced assurance.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-container-backend-opt-in-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Opt-In UX + Audit

- Add an explicit “run with container backend” opt-in flow.
- Require an explicit user acknowledgment of reduced assurance.
- Record the opt-in and the active backend in the audit log.

## Task 3: Hardened Container Baseline

- Define MVP hardening targets:
  - rootless where possible
  - seccomp + dropped Linux capabilities
  - read-only root filesystem + ephemeral writable layers
  - deny-by-default egress (unless the role is a gateway role)
- Specify concrete networking enforcement (MVP):
  - run each role in its own network namespace
  - default: no network connectivity (or loopback only)
  - if egress is explicitly granted, enforce via explicit host-level rules (firewall/proxy allowlists), not in-container configuration
- Ensure the isolation boundary is represented as “container (reduced assurance)” in UI/logs.

## Task 4: No Host Mounts + Artifact Movement

- Maintain the same “no host filesystem mounts” rule.
- Provide artifacts/workspace state via explicit images/volumes that preserve the same data-movement semantics.

## Task 5: Policy Integration

- Ensure the policy engine blocks containers by default.
- Ensure microVM launch failures do not auto-trigger container mode.

## Acceptance Criteria

- Container mode cannot be enabled without an explicit recorded opt-in.
- The reduced assurance posture is unmissable in UX and audit.
- Role capabilities and artifact routing semantics remain consistent across backends.
- Deny-by-default egress is real (attempted outbound connections fail unless explicitly allowed and audited).
