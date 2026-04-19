---
schema_version: 1
id: security/model-provider-substrate-and-secret-ingress
title: Model Provider Substrate And Secret Ingress
status: active
suggested_context_bundles:
    - go-control-plane
    - protocol-foundation
---

# Model Provider Substrate And Secret Ingress

When trusted RuneCode services expose model-provider configuration, credential custody, readiness, or execution surfaces:

- Keep one shared provider substrate for direct credentials, future OAuth-derived material, and future bridge-runtime session material; do not create separate provider architectures per auth mode
- Model provider identity as a stable `provider_profile_id`; auth-mode changes, credential rotation, validation retries, and later auth-material evolution must not redefine logical provider identity
- Keep provider profile metadata and auth material distinct:
  - provider profiles carry trusted identity, endpoint identity, allowlisted model identity, adapter kind, lifecycle metadata, and broker-projected posture
  - auth material carries custody and activation state only; it must not become a second source of provider identity, readiness semantics, or model-selection authority
- Keep endpoint identity on the shared typed destination-descriptor model and canonical `destination_ref`; do not let raw provider URLs, SDK configuration blobs, or daemon-private caches become the trust root
- Keep manual allowlisted model IDs canonical for control-plane selection and routing; provider discovery and compatibility probes may inform posture, but must remain advisory rather than mutating authority silently
- Keep provider-specific wire payloads below the canonical typed model boundary; canonical `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` contracts remain authoritative even when adapters target provider-native APIs internally
- Expose broker-owned setup and inspection contracts through typed protocol objects; CLI and TUI flows must stay thin adapters over the same trusted setup-session, secret-ingress, validation, and read-model semantics
- Treat raw secret values as ingress-only material:
  - raw secret bytes must not appear in ordinary typed broker request or response bodies
  - raw secret bytes must not travel through CLI arguments, environment variables, logs, fixtures, or daemon-private convenience files
  - trusted secret entry may use stdin-backed or masked interactive ingress over one-time broker-issued handles
- Keep `secretsd` as the only long-lived provider credential store; other trusted services may hold scope-bound leased material transiently only and must not persist a second long-lived provider credential cache
- Issue provider execution credentials as short-lived leases bound to the exact consumer, role, scope, and provider profile needed for model execution; retrieval, renewal, and revoke behavior must fail closed on binding mismatch
- Broker-projected provider posture must distinguish configuration, credential, connectivity, compatibility, and effective-readiness dimensions explicitly; do not collapse setup truth into one opaque status string or client-local heuristic
- Keep reason codes and validation attempt identity stable and typed so CLI and TUI clients can render setup and validation state without scraping prose or inferring lifecycle from transport details
- Readiness and compatibility truth must remain broker-projected and durable across restart and restore; daemon-private caches or TUI-local state must not become the effective source of truth
- Validation flows may update compatibility and readiness posture, but they must not bypass the shared provider substrate, mutate canonical allowlists implicitly, or expose raw credential material
- Tests should cover stable profile identity across repeated setup, secret ingress one-time-token behavior, fail-closed recovery when secret material is missing, projected read-model redaction of secret custody internals, and readiness gating of provider-backed execution
