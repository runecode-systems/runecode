# Tasks

## Dependency Fetch Gateway Contract

- [ ] Define the dedicated dependency-fetch role.
- [ ] Keep workspace roles offline while fetches happen through the explicit gateway role.
- [ ] Model package-registry destinations through the shared typed `DestinationDescriptor` / allowlist-entry pattern.
- [ ] Keep dependency-fetch operations aligned with the shared gateway operation taxonomy rather than dependency-local outbound verbs where the shared model is sufficient.

## Offline Cache Artifact Model

- [ ] Define lockfile-driven fetch inputs.
- [ ] Store fetched dependencies as read-only artifacts in the offline cache.

## Policy + Audit Integration

- [ ] Keep dependency fetch posture explicit and auditable.
- [ ] Record destinations, bytes, timing, and cache outcomes without weakening trust boundaries.
- [ ] Keep the distinction explicit between gateway fetch/cache-fill actions and offline consumption of cached read-only dependencies inside workspace execution.
- [ ] Keep dependency-fetch audit fields aligned with the shared gateway audit evidence model.

## Acceptance Criteria

- [ ] Dependencies can be fetched without giving workspace roles direct internet access.
- [ ] Offline cache outputs stay read-only and auditable.
