# Auth Gateway Role v0

User-visible outcome: provider login/refresh flows run in a dedicated `auth-gateway` role with auth-only egress, no workspace access, no environment-variable secret injection, and complete auditability.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-12-1030-auth-gateway-role-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Auth-Gateway Boundary + Egress Model

- Introduce a dedicated `auth-gateway` role kind:
  - no workspace mounts/access
  - auth-only public egress to an explicit allowlist of provider auth endpoints
  - deny-by-default network posture; any egress is an explicit signed-manifest opt-in
- `auth-gateway` must not execute tools, read/write workspace files, or apply patches.
- `auth-gateway` emits typed artifacts/events only (no raw secrets).

Parallelization: can be designed in parallel with provider specs; it depends on a stable gateway allowlist model in `agent-os/specs/2026-03-08-1039-policy-engine-v0/`.

## Task 3: OAuth + Device-Code Flow Contract (Provider-Agnostic)

- Support two official login paths (provider-specific details live in provider specs):
  - browser-based flow with a localhost callback bound to loopback only
  - device-code flow for headless/remote environments
- Require PKCE where applicable.
- Fail closed on OAuth state/redirect mismatches.
- Do not accept secrets via environment variables or CLI args.
  - token material flows only via local IPC to `secretsd` (or via FD/stdin where needed).

Parallelization: can be implemented in parallel with `secretsd` and the broker local API once token/lease schemas are defined.

## Task 4: Secretsd Integration (No Second Store)

- Store long-lived auth material (refresh tokens, rotation metadata) only in `secretsd`.
- Issue short-lived, scope-bound leases to `model-gateway` (and only when allowed by signed manifest + policy).
- If a provider runtime requires persisted auth state, it must live only in a secretsd-managed encrypted directory and be treated as secret material.

Parallelization: can be implemented in parallel with model-gateway/provider bridge work; it depends on stable lease semantics.

## Task 5: Audit + Error Semantics

- Emit typed audit events for:
  - login start/completed/cancelled
  - token refresh success/failure
  - lease issuance/renewal/revocation (without logging raw secrets)
- Use the shared protocol error envelope and stable reason codes (see `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`).

Parallelization: can be implemented in parallel with audit log verify work; it depends on stable audit event schemas.

## Acceptance Criteria

- Auth flows run only inside `auth-gateway` and never in workspace roles.
- No environment-variable secret injection is used.
- `secretsd` is the only long-lived secrets store.
- Auth activity is auditable with typed events and stable errors.
