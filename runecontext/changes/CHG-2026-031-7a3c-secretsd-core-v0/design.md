# Design

## Overview
Implement the reusable `secretsd` foundation for secret storage and lease management as a standalone feature boundary.

## Key Decisions
- Long-lived secrets are stored only in `secretsd`; other components use leases only.
- Secrets storage fails closed by default when secure key storage is unavailable.
- Secret values are never accepted via CLI args or environment variables.
- Lease issuance is short-lived, scope-bound, and fully audited.
- Local `secretsd` health/readiness is primarily a daemon/supervision surface; user-facing operational posture should converge through broker operator-facing summaries rather than a separate long-term client API.

## Canonical Lease Contract

- `secret_access` remains the policy action family for lease operations.
- This feature introduces a typed `SecretLease` object family as the reusable trusted contract for:
  - persisted long-lived secret access
  - future auth-derived short-lived tokens
- `SecretLease` should carry binding and lifecycle metadata rather than acting as a generic secret-payload object.
- The lease contract should cover at least:
  - stable lease identity
  - secret or derived-token subject identity
  - bound consumer identity and `role_kind`
  - bound target or destination scope
  - lease mode and renewal posture
  - issued, expiry, and revocation state
  - action, policy, and audit binding hashes

## Lease Lifecycle Rules

- The initial lifecycle vocabulary should cover issue, renew, and revoke.
- Renew and revoke operations should bind to an existing canonical lease identity rather than only restating a `secret_ref`.
- Leases are short-lived by default.
- Requested TTLs are advisory; `secretsd` enforces the effective TTL cap.
- Effective TTL should remain tight enough for least privilege and should never outlive underlying secret or token validity.
- Renew succeeds only when the requesting principal, bound scope, and policy context still match the active lease.
- Revocation must be durable and survive restart.
- Startup and recovery fail closed unless durable lease and revocation state can be reconstructed consistently enough to preserve prior deny, expiry, and revoke outcomes.

## Delivery Rules

- Secret values and raw tokens must not be delivered through CLI args, environment variables, logs, or ordinary boundary-visible protocol objects.
- The typed lease contract is public to trusted services, but secret material delivery remains a trusted local channel.
- Lease consumption should be by canonical lease identity through a trusted local retrieval path bound to the intended consumer principal.
- The design should support both direct trusted daemon consumers and later narrow bridge handoff flows without making bridge/runtime payloads the canonical custody contract.

## Storage And Custody Posture

- Use one storage abstraction that can represent both the secure default posture and an explicit degraded portable posture.
- Default posture prefers OS or hardware-backed key protection and fails closed if that posture cannot be established.
- Portable mode is an explicit opt-in using passphrase-derived encryption with audited degraded posture.
- Custody posture should be recorded explicitly so broker/operator surfaces can distinguish secure default posture from approved degraded portable posture.
- Durable local state includes secret metadata, encrypted secret material, lease state, revocation state, and linkage metadata needed for audit and policy recovery.

## Secret Onboarding

- The onboarding baseline should be local-only and secret-safe.
- `stdin` should be the portable canonical import path.
- File-descriptor based import may be supported where platform-appropriate.
- Secret values must never appear in shell history, argv surfaces, or audit content.
- Audit records should capture metadata, posture, and lifecycle identity without carrying secret material.

## Operator Posture

- `secretsd` may expose local-only supervision health for broker/service management.
- Broker remains the long-lived operator-facing posture surface.
- User-facing or operator-facing secrets posture should be projected through broker subsystem readiness or status models rather than a second daemon-specific user API.

## Main Workstreams
- Storage and key posture policy.
- Typed lease contract and lifecycle rules.
- Trusted local delivery path rules.
- Safe secret onboarding/import flow.
- Daemon-local supervision health plus broker-projected operator posture.
