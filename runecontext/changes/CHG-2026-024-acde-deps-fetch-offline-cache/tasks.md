# Tasks

## Dependency Fetch Gateway Contract

- [ ] Define the dedicated dependency-fetch role.
- [ ] Keep workspace roles offline while fetches happen through the explicit gateway role.
- [ ] Model package-registry destinations through the shared typed `DestinationDescriptor` / allowlist-entry pattern.
- [ ] Keep dependency-fetch operations aligned with the shared gateway operation taxonomy rather than dependency-local outbound verbs where the shared model is sufficient.
- [ ] Clarify and implement moderate-profile approval semantics as shared gateway checkpoints:
  - enabling dependency fetch or expanding dependency-fetch scope requires approval
  - ordinary `fetch_dependency` actions inside already-approved scope do not require a fresh approval per cache miss
- [ ] Add a narrow typed dependency-fetch identity contract so `payload_hash` binds to a canonical dependency request object rather than raw lockfile bytes or tool-private cache state.
- [ ] Keep destination identity, allowlist matching, and request identity portable and topology-neutral; do not let local filesystem paths, package-manager private cache layout, or host-local handles become canonical identity.

## Broker-Owned Fetch And Cache Authority

- [ ] Define a broker-owned dependency fetch/cache endpoint or operation family.
- [ ] Keep the broker as the authoritative owner of cache lookup, miss coalescing, fetch execution, digest verification, CAS persistence, and audit emission.
- [ ] Keep runner-local state advisory only; the runner must not become the authority for cache hits, cache keys, or registry access.
- [ ] Coalesce identical in-flight fetch misses by stable canonical request identity.
- [ ] Keep bounded parallelism explicit and configurable rather than assuming unbounded fan-out.
- [ ] Stream fetch results directly into verification and CAS persistence rather than buffering full dependency payloads in memory.

## Offline Cache Artifact Model

- [ ] Define lockfile-driven fetch inputs.
- [ ] Define the hybrid cache model:
  - lockfile-bound batch request and result semantics for workflow and operator surfaces
  - resolved-unit canonical CAS storage for reuse and dedupe
  - derived read-only materialization for workspace consumption
- [ ] Introduce dedicated dependency-cache artifact classes, keeping batch manifests distinct from resolved dependency payload units.
- [ ] Store fetched dependencies as immutable read-only artifacts in the offline cache.
- [ ] Keep materialized package-manager stores or unpacked dependency trees derived and non-canonical.
- [ ] Define exact cache-hit semantics against canonical typed identity and resolved-unit digests.
- [ ] Fail closed on ambiguous partial reuse, stale metadata, unverifiable cache identity, or incomplete cache state.
- [ ] Define retention and GC rules that preserve cache correctness and do not treat materialized trees as the source of truth.

## Offline Consumption And Artifact Handoff

- [ ] Keep the distinction explicit between third-party egress and broker-mediated internal artifact handoff for cached dependency use.
- [ ] Define explicit artifact flow rules for `dependency-fetch` producers and allowed workspace consumers.
- [ ] Ensure offline dependency consumption remains ordinary workspace execution and does not silently regain public egress authority.

## Policy + Audit Integration

- [ ] Keep dependency fetch posture explicit and auditable.
- [ ] Record destinations, bytes, timing, allowlist attribution, and cache outcomes without weakening trust boundaries.
- [ ] Keep dependency-fetch audit fields aligned with the shared gateway audit evidence model.
- [ ] Add the minimal dependency-specific audit detail needed for cache semantics, such as cache outcome and referenced dependency artifact digests.
- [ ] Keep policy and audit joins bound to canonical request identity, policy decision identity, and relevant artifact digests.

## Registry Auth Readiness

- [ ] Keep the first end-to-end implementation public-registry-first as the recommended initial delivery slice.
- [ ] Design registry identity and auth material as separate concerns from day one so private registries remain additive later.
- [ ] When private registries are added, keep `secretsd` as the only long-lived credential store and use only short-lived broker-held leased material.
- [ ] Ensure runner and workspace surfaces never receive or persist private registry credentials.
- [ ] Avoid public-only shortcuts in cache identity, audit, or broker plumbing that would force a redesign when private registries are added later.

## Performance And Portability

- [ ] Keep the cache and fetch contracts efficient on constrained local hardware and portable to later larger-scale deployments.
- [ ] Avoid full-response buffering and path-shaped identity assumptions.
- [ ] Keep canonical cache identity independent from storage backend, local blob paths, or single-host assumptions.

## Acceptance Criteria

- [ ] Dependencies can be fetched without giving workspace roles direct internet access.
- [ ] Offline cache outputs stay read-only and auditable.
- [ ] Dependency approval semantics use shared gateway checkpoints for scope changes without introducing per-fetch approval spam.
- [ ] Canonical cache identity is typed, hash-bound, and independent from tool-private materialization paths.
- [ ] Cached dependency consumption is modeled as broker-mediated internal artifact handoff, not third-party egress.
- [ ] The broker remains the authoritative dependency cache owner; runner-local state remains advisory.
- [ ] The first end-to-end slice is explicitly public-registry-first without blocking later private-registry support through leased trusted credentials.
- [ ] Fetch and cache-fill behavior remains efficient under constrained local resources and does not require full-body buffering.
