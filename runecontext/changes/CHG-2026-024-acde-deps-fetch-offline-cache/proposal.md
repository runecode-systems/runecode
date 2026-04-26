## Summary
RuneCode can fetch dependencies without giving workspace roles internet access, while keeping the cache model, approval posture, and artifact handoff semantics strong enough to serve as the long-term foundation for future workflow and scaling work.

## Problem
This feature now has a canonical RuneContext change record, preserving the migrated planning content without relying on legacy Agent OS folders or path aliases.

The current repository already defines `dependency-fetch` as a first-class gateway role/action/destination family, but it does not yet define the missing foundation needed to implement it safely and durably:
- a typed dependency-fetch identity contract suitable for exact cache keys, replay safety, and audit binding
- a broker-owned offline cache model that keeps the runner non-authoritative
- explicit offline artifact handoff semantics for cached dependency consumption inside workspace execution
- a performance posture that scales from small local devices to larger shared or distributed deployments without changing boundary-visible contracts

## Proposed Change
- Shared dependency-fetch gateway contract with checkpoint-style approval semantics for scope enablement or expansion rather than per-fetch approval spam.
- Typed dependency fetch identity and canonical hash binding so `payload_hash` represents a reviewed dependency request object rather than ad hoc lockfile bytes or tool-private cache state.
- Hybrid offline cache artifact model: lockfile-bound batch request/result semantics, resolved-unit canonical CAS storage, and derived read-only workspace materialization.
- Broker-owned fetch/cache authority, including cache lookup, miss coalescing, bounded concurrency, stream-to-CAS persistence, and broker-mediated offline artifact handoff.
- Public-registry-first end-to-end delivery as the recommended first slice, with private-registry-ready auth abstractions from day one so credentialed registries remain an additive follow-up rather than a cache-foundation rewrite.
- Policy + audit integration aligned with the shared gateway taxonomy and audit field set, with minimal dependency-specific extensions for cache outcome and artifact linkage.

## Why Now
This work now lands in `v0.1.0-alpha.8`, because first-party implementation workflows need dependency material without granting workspace roles internet access.

Landing dependency fetch and offline cache before the first productive workflow pack keeps isolated implementation flows on the intended no-workspace-egress architecture instead of relying on later retrofits.

Locking in the right foundation now also avoids later rewrites in three places that are expensive to fix once workflow packs and runner integrations depend on them:
- approval semantics
- cache identity and artifact persistence
- broker/runner authority boundaries

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- Shared gateway checkpoint semantics are the intended approval model for dependency fetch: ordinary fetches inside already-approved scope should not require a fresh approval on every cache miss.
- Private registry support affects execution plumbing more than the core cache foundation; the first end-to-end slice should stay public-registry-first as long as the auth extension points are designed correctly from day one.

## Out of Scope
- Re-introducing legacy Agent OS planning paths as canonical references.
- Making materialized workspace dependency trees, package-manager private store layouts, or host-local cache directories part of the canonical contract.
- Shipping private registry credential flows in the first end-to-end slice.

## Impact
Keeps Deps Fetch + Offline Cache reviewable as a RuneContext-native change and removes the need for a second semantics rewrite later.

If implemented according to this clarified foundation, later workflow, runner, and scaling work can build on:
- one shared gateway and approval model
- one broker-owned cache authority
- one immutable artifact handoff model
- one topology-neutral identity story that does not depend on local paths or one-machine assumptions

Choosing public-registry-first for the first end-to-end slice keeps the initial implementation focused on the actual cache and trust-boundary foundation rather than coupling it immediately to secret custody, lease renewal, and credentialed registry failure modes.
