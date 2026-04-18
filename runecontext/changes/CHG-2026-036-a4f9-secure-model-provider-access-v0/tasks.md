# Tasks

## Project Coordination

- [ ] Track child feature sequencing and dependencies.
- [ ] Keep shared trust-boundary assumptions explicit and consistent.
- [ ] Keep the inherited `SecretLease`, canonical model boundary, destination identity, request binding, broker-projected posture, and quota semantics explicit across child features.
- [ ] Keep one shared provider-profile and auth-material substrate explicit across direct-credential, OAuth, and bridge-runtime lanes.

## Integration Oversight

- [ ] Ensure child features preserve deny-by-default and least-privilege behavior.
- [ ] Ensure child features keep typed contracts and full audit traceability.
- [ ] Ensure provider-specific lanes inherit auth/model separation rather than redefining combined egress or credential roles.
- [ ] Ensure provider-specific lanes do not redefine token handoff, destination identity, or quota semantics.
- [ ] Ensure direct-credential providers land on the same provider substrate that later OAuth and bridge-runtime providers will reuse.

## Acceptance Criteria

- [ ] Child features remain linked and consistently scoped.
- [ ] Provider lane delivery stays aligned with secure model-access invariants.
- [ ] Downstream provider changes inherit the reviewed secret-custody and canonical model-boundary foundation rather than introducing parallel semantics.
- [ ] Direct-credential provider support does not require a later provider-setup, readiness, or compatibility rewrite when OAuth and bridge-runtime lanes land.
