## Summary
Shared bridge contracts keep user-installed provider runtimes auditable, in explicit LLM-only mode, and below the canonical RuneCode model and lease boundaries.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

The newly refined `secretsd` and `model-gateway` foundation narrows what the bridge lane should be allowed to define. Without updating this change, bridge/runtime integrations could still drift into their own token-delivery, destination identity, quota, or request-shape conventions even though those decisions are now meant to be shared and inherited.

## Proposed Change
- Bridge Runtime Contract.
- Compatibility + Probe Model.
- Token Delivery + Session Rules.
- Audit + UX Surfaces.
- Explicit inheritance of the canonical `LLMRequest` / `LLMResponse` / `LLMStreamEvent` boundary and lease-based token handoff model.

## Why Now
This work remains scheduled for v0.2, and the bridge lane should now explicitly inherit the shared model-boundary, token-handoff, destination-identity, and quota foundation rather than redefining it during later provider integrations.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Bridge runtimes are consumers of canonical RuneCode model contracts, not the source of those contracts.

## Out of Scope
- Runtime implementation of the feature during this migration step.
- Re-introducing legacy Agent OS planning paths as canonical references.
- Allowing bridge runtimes to redefine the canonical model boundary, secret custody semantics, or operator-facing posture model.

## Impact
Keeps Bridge Runtime Protocol v0 reviewable as a RuneContext-native change, aligned with the reviewed model-gateway and `secretsd` foundation, and avoids a later rewrite of bridge token-delivery and request-boundary semantics.
