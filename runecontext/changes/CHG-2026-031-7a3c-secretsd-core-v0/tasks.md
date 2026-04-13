# Tasks

## Storage + Key Posture

- [ ] Implement durable secret storage with secure key storage preference.
- [ ] Fail closed by default when secure storage is unavailable.
- [ ] Support explicit, audited passphrase-derived encryption opt-in for portable setups.
- [ ] Record effective custody posture so broker/operator surfaces can distinguish secure default posture from degraded portable posture.
- [ ] Persist secret metadata, lease state, revocation state, and linkage metadata as trusted local durable state.

## Typed Lease Contract

- [ ] Define a typed `SecretLease` object family for persisted secrets and future short-lived derived tokens.
- [ ] Define stable lease identity, consumer binding, destination or target binding, action/policy binding, and revocation fields.
- [ ] Extend `secret_access` lifecycle semantics so renew and revoke bind to an existing lease identity rather than only restating `secret_ref`.

## Lease Lifecycle

- [ ] Implement short-lived scoped leases.
- [ ] Define effective TTL defaults and hard caps that preserve least privilege.
- [ ] Define renewal semantics that require the same consumer, scope, and still-valid policy context.
- [ ] Define durable revocation semantics that survive restart and preserve deny outcomes fail closed.
- [ ] Audit lease issue, renew, revoke, deny, and expiry-relevant lifecycle events.

## Delivery Rules

- [ ] Keep secret values out of CLI args, environment variables, logs, and ordinary boundary-visible protocol objects.
- [ ] Define the trusted local retrieval path for secret material by canonical lease identity.
- [ ] Ensure downstream bridge/runtime integrations can depend on the lease boundary without redefining custody semantics.

## Secret Onboarding

- [ ] Support stdin as the canonical portable secret onboarding path.
- [ ] Support file-descriptor secret onboarding where platform-appropriate.
- [ ] Audit secret metadata without secret values.

## Health + Metrics

- [ ] Expose local-only health/readiness signals.
- [ ] Emit minimal local-only operational metrics.
- [ ] Keep any user-facing visibility for secrets posture aligned with broker operator-facing status surfaces rather than a second long-lived client API.

## Acceptance Criteria

- [ ] No component outside `secretsd` persists long-lived secrets.
- [ ] Lease usage is explicit, bounded, and auditable.
- [ ] The reusable lease contract is strong enough for downstream auth-derived short-lived tokens without weakening secret-custody semantics.
- [ ] Restart or recovery preserves lease and revocation safety fail closed.
