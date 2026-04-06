# Tasks

## Storage + Key Posture

- [ ] Implement durable secret storage with secure key storage preference.
- [ ] Fail closed by default when secure storage is unavailable.
- [ ] Support explicit, audited passphrase-derived encryption opt-in for portable setups.

## Lease Lifecycle

- [ ] Implement short-lived scoped leases.
- [ ] Define TTL bounds, renewal, and revocation semantics.
- [ ] Audit all lease issuance and revocation events.

## Secret Onboarding

- [ ] Support stdin/file-descriptor secret onboarding only.
- [ ] Audit secret metadata without secret values.

## Health + Metrics

- [ ] Expose local-only health/readiness signals.
- [ ] Emit minimal local-only operational metrics.
- [ ] Keep any user-facing visibility for secrets posture aligned with broker operator-facing status surfaces rather than a second long-lived client API.

## Acceptance Criteria

- [ ] No component outside `secretsd` persists long-lived secrets.
- [ ] Lease usage is explicit, bounded, and auditable.
