# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its `v0.1.0-alpha.10` roadmap bucket and title after migration.
- Confirm authenticated external anchor submissions align with the shared remote-state-mutation gateway class rather than an external-only outbound category.
- Confirm target identity uses typed exact-match semantics rather than raw URL-only policy.
- Confirm the change explicitly decides `v0` external anchor submission uses exact-action approval per submission and reserves signed-manifest automation as an additive later posture over the same typed request path.
- Confirm authenticated target access, if any, remains lease-bound through the shared secrets model.
- Confirm audit evidence includes canonical target identity, anchoring subject identity, outbound payload or subject hash, bytes, timing, outcome, and relevant lease or policy bindings.
- Confirm anchored audit chains preserve attestation evidence and verification references when attested runtime evidence is part of the anchored subject rather than flattening those references away.
- Confirm project-context-sensitive anchored evidence reuses validated project-substrate snapshot identity rather than inventing a second project-context reference.
- Confirm the change freezes a durable prepared and execute lifecycle with deferred completion as a first-class outcome rather than assuming all external anchor submissions complete inline.
- Confirm the change keeps one architecture across constrained and scaled environments by forbidding network I/O under audit-ledger lock and by requiring bounded worker concurrency and idempotent retry semantics instead of topology-specific variants.
- Confirm the shared `AuditReceipt(kind=anchor)` envelope remains minimal and authoritative while larger target-specific proof bytes remain typed sidecar evidence rather than a second receipt family.
- Confirm the change freezes one concrete first runtime adapter, with transparency-log style anchoring recommended first, while keeping timestamp-authority and public-chain kinds additive.
- Confirm the change requires target-set foundations to use `all required targets satisfied` aggregate semantics and explicitly defers quorum-style policies.
- Confirm the performance foundation includes incremental receipt-admission expectations so external anchor submissions do not require full verifier replay as the only hot path when the verified seal state is unchanged.

## Close Gate
Use the repository's standard verification flow before closing this change.
