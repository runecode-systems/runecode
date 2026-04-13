## Summary
Provider login and refresh run in an auth-only gateway role, long-lived tokens live only in `secretsd`, and short-lived auth material is handed off through the canonical lease boundary rather than provider-specific token-delivery shortcuts.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

The recently frozen `secretsd` and `model-gateway` foundation clarifies several auth-lane decisions that should now be reflected here: auth and model egress must remain strictly separate; `SecretLease` is the reusable contract for short-lived token handoff; auth operations should use the same canonical destination identity and shared gateway-operation vocabulary as other gateway lanes; and any user-facing posture should remain broker-projected rather than turning auth flows into a second daemon-specific status surface.

## Proposed Change
- Auth Gateway Role Contract.
- Provider-Agnostic Auth Objects.
- Secret Handling + Token Storage.
- Audit + Policy Integration.
- Canonical short-lived token handoff through `SecretLease`.

## Why Now
This work remains scheduled for v0.2, and the auth lane should now explicitly inherit the reviewed secret-custody and gateway-separation foundation rather than quietly redefining it during later implementation.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- `SecretLease` is the canonical short-lived token handoff contract for downstream consumers such as `model-gateway`.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Turning auth-gateway into a model egress role or a second long-lived credential store.

## Impact
Keeps Auth Gateway Role v0 reviewable as a RuneContext-native change, aligned with the reviewed secrets and model-gateway foundation, and avoids a later rewrite of token handoff and gateway-separation semantics.
