# Design

## Overview
Define the dependency-fetch role and offline cache so workspace roles stay offline while builds can consume fetched artifacts, without creating a dependency-specific side path that weakens the shared gateway, broker, artifact, approval, or trust-boundary foundations.

## Key Decisions
- Inputs are minimal and low-sensitivity (lockfiles only).
- Outputs are read-only artifacts and read-only derived materializations; the canonical cache identity remains on immutable broker-owned artifacts rather than on materialized install trees.
- Dependency fetch should use the shared typed gateway destination/allowlist model, while offline consumption of cached dependencies remains ordinary workspace execution rather than implicit egress.
- This split must also preserve the shared executor-class model: gateway-backed dependency fetch is not ordinary workspace execution, while offline cached dependency use inside the workspace remains `workspace_ordinary` execution.
- Dependency fetch should also stay aligned with the shared gateway operation taxonomy and shared gateway audit field set rather than inventing dependency-local outbound semantics where the shared model is sufficient.
- Moderate-profile approvals should follow shared gateway checkpoint semantics: approval is required for dependency scope enablement or expansion, not for each ordinary fetch against already-approved scope.
- That approval model does not weaken security when the remaining controls stay fail-closed: exact typed request binding, signed allowlists, shared gateway runtime hardening, and broker-owned cache authority continue to gate every fetch.
- The first implementation slice should be public-registry-first and private-registry-ready: private registry support should be additive through trusted lease-based auth plumbing rather than a later cache-identity rewrite.
- The user-facing and workflow-facing surface should be lockfile-batch oriented, while canonical cache storage should be resolved-unit oriented for dedupe, replay safety, and long-term scale.
- A narrow typed dependency request object should be introduced so `payload_hash` binds to a canonical dependency-fetch contract rather than raw lockfile text, tool-private request blobs, or non-portable cache state.
- Cache hits should be exact against canonical typed identity and resolved-unit digests. Ambiguous partial reuse, stale metadata, or unverifiable identity must fail closed instead of silently becoming a cache hit.
- Offline dependency consumption should be modeled as broker-mediated internal artifact handoff, not as third-party egress.
- Broker-owned logical contracts must stay topology-neutral so the same design works on a Raspberry Pi, one local workstation, or later shared and distributed deployments without changing object identity or security semantics.
- Performance foundations must be built into the first design slice: stream directly to CAS, avoid full-body buffering, coalesce identical misses, and keep concurrency bounded and configurable.

## Main Workstreams
- Dependency Fetch Gateway Contract
- Offline Cache Artifact Model
- Policy + Audit Integration

## Shared Gateway And Approval Model

- Keep dependency fetch on the existing shared gateway path:
  - typed `DestinationDescriptor`
  - signed `GatewayScopeRule`
  - shared gateway operation vocabulary
  - shared gateway audit and quota context
- Keep `dependency-fetch` role-specific behavior on top of that shared path rather than creating a separate dependency-only authorization model.
- Preserve the shared policy split between:
  - checkpoint approvals that authorize changes in posture or scope
  - ordinary allow/deny evaluation for work that remains within already-approved scope
- For this feature, that means:
  - `enable_dependency_fetch`, allowlist changes, or scope expansion remain checkpoint-style approval events
  - ordinary `fetch_dependency` requests against already-approved registry scope should be automatable
  - cache misses must not become a backdoor approval trigger by themselves

This is the best long-term foundation because it keeps approval semantics tied to trust posture and blast radius rather than to transient cache state.

## Typed Dependency Fetch Identity

The current shared gateway payload family is necessary but not sufficient for dependency caching. Dependency fetch needs a reviewed typed identity object that can answer all of the following without consulting tool-private cache state:
- what exact dependency request is being made
- what lockfile or lockfile-derived batch it belongs to
- what registry destination identity is authoritative
- what resolved dependency units should exist if the request succeeds
- what exact hashes should be used for cache lookup, audit binding, replay safety, and later approval or verification joins

The recommended contract shape is:
- a lockfile-bound batch request object used by workflow, broker, and operator-facing surfaces
- resolved dependency unit identity objects used by canonical CAS storage and cache dedupe
- `payload_hash` bound to the canonical typed dependency request object hash, not to raw lockfile bytes alone

This avoids several future traps:
- batch-only identity that duplicates too much and blocks reuse
- per-tool private cache identity becoming the trust root
- non-portable path-shaped identity leaking into protocol or audit contracts

## Hybrid Offline Cache Model

The cache model should be hybrid from the start.

### Batch-Oriented Broker Surface

- Workflows and broker APIs should ask for dependency availability in lockfile-bound batches.
- The batch contract should be the operator-facing and workflow-facing unit of intent, review, and status.
- A successful batch should return a small typed manifest that binds the request to the exact resolved-unit artifacts produced or reused.

### Resolved-Unit Canonical Storage

- Canonical storage should be resolved-unit oriented.
- Each resolved unit should have a stable typed identity and immutable digest-addressed payload(s) in CAS.
- The batch manifest should reference these unit digests rather than duplicate or replace them.

This gives the best foundation for:
- small-device efficiency through reuse and bounded storage growth
- large-system efficiency through dedupe, cache sharing, and coalesced misses
- future distributed or shared storage backends without changing logical contracts

### Derived Materialization

- Workspace execution may require derived read-only materialization such as a package-manager-compatible offline store or unpacked dependency tree.
- Those materializations are derived runtime products only.
- They must not become the canonical cache identity, approval binding, or audit truth.

## Artifact Model And Internal Handoff

Fetched dependency material should be represented as dedicated dependency-cache artifacts rather than overloaded onto unrelated data classes.

The preferred foundation is to separate:
- dependency batch manifests
- resolved dependency unit payloads

This keeps policy, audit, GC, and operator UX clear about the distinction between:
- metadata that explains what was fetched
- payload blobs that are actually consumed or materialized

Offline dependency consumption should then use explicit broker-mediated internal artifact handoff semantics:
- producer role: `dependency-fetch`
- consumer roles: workspace role kinds that are allowed to consume cached read-only dependencies
- non-egress handoff path rather than third-party egress semantics

This distinction matters because cached dependency use inside the workspace is not the same security event as third-party network egress.

## Broker-Owned Fetch And Cache Authority

The broker should own all authoritative dependency-fetch and cache behavior:
- cache lookup
- policy evaluation
- registry access on miss
- fetch coalescing for identical in-flight requests
- digest verification and CAS persistence
- audit emission
- broker-mediated artifact read or materialization for offline use

The runner must not become the authority for:
- registry access
- cache keys
- cache hit truth
- credential handling
- offline cache lifecycle state

Runner-local state may remember that a requested dependency set was already staged for the current run, but that remains advisory only.

## Public-First, Private-Ready Registry Auth

The first end-to-end slice should target public registries only.

This is the recommended first slice because it strengthens the core foundation without paying the extra implementation and review cost of credentialed registry flows before the cache model itself is proven. Public-registry-first keeps the first slice focused on:
- typed dependency request identity
- broker-owned cache and fetch authority
- immutable artifact storage and internal handoff
- shared gateway audit and approval semantics

Adding private registries immediately would increase first-slice complexity, but would not materially improve the agreed cache, artifact, or approval foundation as long as auth-ready extension points are established now.

This does not weaken the foundation as long as the following auth-ready rules are established now:
- registry identity remains separate from auth material identity
- broker internals use one credential-source abstraction even when the initial implementation is no-auth
- cache keys never include secret values
- audit fields are ready to record lease usage and auth posture later without exposing raw secrets
- the runner never receives registry credentials

With these rules in place, private-registry support becomes a follow-up execution lane, not a foundational redesign.

When private registries are added later:
- `secretsd` remains the only long-lived credential store
- broker services may hold only short-lived leased material transiently
- leases must bind to exact consumer, role, scope, and registry identity
- no second long-lived credential cache may be created in broker, runner, or workspace surfaces

The design should therefore explicitly avoid any public-only shortcut that would block this later extension, but it should not pull credentialed registry support into the first end-to-end slice.

## Performance And Scaling Posture

This feature should be designed to operate efficiently on both constrained local hardware and larger installations.

The first implementation slice should therefore include these performance foundations:
- stream registry responses directly into digest verification and CAS persistence
- avoid buffering full dependency payloads in memory
- coalesce identical concurrent misses by stable canonical request key
- keep fetch parallelism bounded and configurable
- keep canonical cache identity independent from storage backend or local filesystem layout
- make local filesystem CAS the initial deployment posture without baking host-local paths into protocol or audit identity

These choices preserve local-first operation while keeping later shared-cache, sharded storage, or horizontally scaled broker work additive.

## Failure And Security Posture

The cache must fail closed on ambiguity.

Examples:
- unknown or unsupported dependency request object version
- unverifiable or missing typed request hash binding
- destination not present in the active signed allowlist
- redirect escape or runtime destination hardening violation
- incomplete or inconsistent cache metadata
- stale or partial cache entries that cannot prove exact identity

No negative security impact is introduced by using shared gateway checkpoint semantics when these fail-closed rules remain in place.

## Recommended First Implementation Slice

The best first slice is not runner-first. It should land in this order:
1. typed dependency request and identity objects
2. broker-owned fetch/cache endpoint and persistence path
3. dependency cache artifact classes and internal handoff semantics
4. audit and policy binding for cache hit, miss, and fill outcomes
5. workflow and runner integration for offline consumption

That order preserves the core tenets of the project:
- trusted control plane owns cross-boundary authority
- runner remains untrusted and non-authoritative
- workspace roles remain offline
- artifacts remain immutable and hash-addressed
- topology-neutral contracts remain stable across future deployment shapes

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
