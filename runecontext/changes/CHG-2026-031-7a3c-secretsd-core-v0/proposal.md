## Summary
RuneCode has a dedicated secrets daemon that stores long-lived secrets safely, records explicit custody posture, and issues short-lived, scope-bound, typed leases with complete auditing.

## Problem
The previous combined change mixed shared secret lifecycle foundations with model egress behavior, making sequencing and verification less clear.

After the split, the reusable secret-management contract still needed more detail: the system had `secret_access` action intent but not an explicit first-class lease contract, renew and revoke semantics were still underspecified, and the line between daemon-local supervision versus broker-visible operator posture needed to be made explicit before downstream auth and provider lanes depended on it.

## Proposed Change
- Define `secretsd` storage and key posture requirements.
- Define a typed `SecretLease` family as the reusable contract for persisted secrets and future short-lived derived tokens.
- Define lease issue, renew, revoke, expiry, and restart-time recovery semantics.
- Define secret delivery rules that keep secret bytes out of CLI args, environment variables, logs, and ordinary boundary-visible protocol objects.
- Define safe secret onboarding/import rules and degraded portable-custody posture.
- Keep daemon-local health/readiness supervision separate from broker-projected operator-facing posture.

## Why Now
This feature isolates reusable secret-management foundations so downstream auth, gateway, bridge, and provider features can depend on one reviewed contract rather than inventing their own token or secret handoff semantics.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Secret values are trusted local data, not ordinary boundary-visible control-plane payloads.

## Out of Scope
- Model provider egress and payload-shaping behavior.
- Provider-specific runtime bridge contracts.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Creating a second user-facing secrets status API distinct from broker operator-facing posture surfaces.

## Impact
Keeps secrets lifecycle behavior independently reviewable while preserving project-level traceability to the broader secure model/provider access plan and giving downstream lanes one stronger reusable lease and custody boundary.
