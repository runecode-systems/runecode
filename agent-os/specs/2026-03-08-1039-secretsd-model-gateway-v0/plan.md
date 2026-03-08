# Secretsd + Model-Gateway v0

User-visible outcome: third-party model access is possible only via an explicitly allowed gateway role, using short-lived scoped secrets leases, with boundary redaction and complete auditing.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-secretsd-model-gateway-v0/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Secretsd MVP Interface

- Provide a minimal secrets daemon that:
  - stores long-lived secrets at rest (prefer hardware-backed / OS key storage where available)
  - fails closed by default if secure key storage is unavailable (no silent plaintext fallback)
  - allows an explicit, audited opt-in to passphrase-derived encryption for developer/portable setups
  - issues short-lived, scope-bound leases only as allowed by the signed manifest
  - defines lease TTL bounds, renewal rules, and revocation semantics
  - records every lease as an audit event (without logging raw secrets)
- Define a safe secret onboarding/import flow (MVP):
  - secrets are provided via stdin or a file descriptor (never CLI args; avoid env vars)
  - only secret metadata/IDs are logged/audited (never secret values)

## Task 3: Model-Gateway Role

- Implement a dedicated gateway role with:
  - network egress allowlist (model provider domains only)
  - no workspace access
  - provider keys obtained only via secrets leases
  - schema-validated request/response boundary
- Harden egress controls against SSRF and DNS rebinding:
  - resolve and validate destinations (block RFC1918/link-local/reserved ranges)
  - restrict redirects (or disable by default)
  - require TLS with certificate validation and SNI matching
  - apply strict timeouts and response size limits

## Task 4: Data-Class Policy for Model Egress

- Default deny for third-party model usage.
- When explicitly opted in, allow only specific data classes (e.g., `spec_text`, optionally `diffs`/`approved_file_excerpts` per manifest).
- Enforce redaction at the boundary structurally:
  - use schema field classification metadata (`secret` fields are rejected/stripped)
  - prefer allowlists of permitted fields/classes over heuristic redaction

## Task 5: Audit + Quotas

- Log outbound requests (destination, bytes, timing) as audit events.
- Enforce basic quotas (requests/bytes/time) for the gateway role.

## Acceptance Criteria

- No other role can directly reach the public internet for model traffic.
- Secrets are never persisted in the launcher/broker/scheduler; only leases are used.
- Model-gateway blocks SSRF/DNS rebinding classes of attacks (private IPs, unsafe redirects) by default.
- Opt-in model egress is explicit, enforceable, and auditable.
