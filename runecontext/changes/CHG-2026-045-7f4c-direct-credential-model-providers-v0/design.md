# Design

## Overview
Add direct-credential provider access on top of the existing `secretsd` and `model-gateway` foundation without creating a parallel provider stack.

## Key Decisions
- Direct credentials are one auth-material path within a shared provider substrate, not a separate provider architecture.
- Provider setup should distinguish provider identity, endpoint identity, supported auth modes, active auth mode, and model-capability metadata instead of flattening them into one untyped profile blob.
- Provider-profile identity must remain stable across auth-mode changes, credential rotation, validation retries, and later OAuth or bridge adoption.
- Provider setup is multi-profile from the start; RuneCode must support more than one configured profile rather than one global provider-family settings blob.
- OpenAI-compatible and Anthropic-compatible families should share as much trusted orchestration as possible while keeping provider-specific wire details below the canonical `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` boundary.
- `v0` targets OpenAI-compatible Chat Completions and Anthropic Messages. Future OpenAI Responses support must be additive rather than a profile or control-plane rewrite.
- Popular SDKs and public API docs are interoperability references for adapter implementation, not authoritative control-plane contracts.
- `secretsd` remains the only long-lived credential store; `model-gateway` receives only scope-bound leased material.
- Direct credential entry uses broker-owned setup sessions plus trusted secret ingress. Ordinary typed broker JSON requests carry metadata and setup handles, not raw secret values.
- Endpoint identity remains on the shared typed destination and allowlist model rather than provider-local raw URL handling.
- Manual allowlisted model IDs remain canonical; provider discovery and probe results are advisory inputs to compatibility posture rather than authority.
- Compatibility, readiness, and failure posture must be broker-projected and reusable by later OAuth and bridge-provider features.
- Direct credential entry must use trusted interactive setup flows; environment variables and command-line secret injection remain forbidden.
- TUI setup must follow the existing Bubble Tea and Lip Gloss shell patterns and use masked secret-entry controls rather than reusing ordinary visible text-entry components.

## Shared Provider Substrate

### Provider Profile Model

- Define a provider-profile model that can represent at least:
  - `provider_profile_id`
  - operator-facing display label
  - provider family
  - adapter kind
  - canonical endpoint descriptor and destination identity
  - supported auth modes
  - current auth mode
  - allowlisted model identities
  - model-capability metadata
  - compatibility posture
  - quota and usage-accounting posture
  - broker-projected audit and lifecycle metadata needed for setup and inspection flows
- Provider-profile identity must follow these rules:
  - the same profile may move from direct credentials to later OAuth-derived or bridge-session material without changing profile identity
  - rotating credentials or revalidating compatibility must not create a new logical provider identity
  - endpoint identity remains derived from the shared typed destination-descriptor model rather than from provider-local raw URL authority
  - readiness and compatibility remain broker-projected state tied to the profile rather than daemon-private caches or TUI-local interpretation
- The substrate must support multiple profiles, including multiple endpoints within the same provider family, so future enterprise, self-hosted, or staged-provider setups do not require a control-plane rewrite.

### Auth-Material Model

- Define an auth-material model that allows one provider profile to later use:
  - long-lived direct credentials
  - short-lived OAuth-derived material
  - bridge-runtime session material
- Auth material attaches to provider profiles but does not redefine provider identity, readiness vocabulary, model compatibility posture, or audit identity.
- Direct-credential material should resolve to `secretsd` secret identity, custody posture, and lease policy metadata rather than carrying raw secret values in broker-visible objects.
- Later auth and bridge lanes may extend auth-material detail, but they must not redefine the shared provider-profile, readiness, or compatibility contracts frozen here.

### Canonical Boundary Inheritance

- Keep provider adapters below the typed model boundary so later auth-mode expansion does not change canonical request, response, or stream contracts.
- Request execution remains bound to canonical `LLMRequest` identity and shared destination identity rather than to provider-local payload shapes or provider SDK object families.
- Direct-credential providers do not get to redefine destination identity, request binding, quota semantics, or broker-projected posture.

## Broker-Owned Setup And Secret Ingress

- Direct-credential setup should use a broker-owned setup session with typed phases for:
  - provider-profile metadata creation or update
  - auth-mode declaration
  - secret ingress negotiation
  - validation and compatibility probe execution
  - final commit of broker-visible readiness posture
- Non-secret setup metadata such as display label, endpoint descriptor, expected auth mode, allowlisted model IDs, and operator validation preferences may travel through ordinary typed broker requests.
- Raw secret values must not travel through ordinary typed broker request or response bodies, CLI args, environment variables, logs, or daemon-private convenience files.
- The setup flow should therefore use a separate trusted secret-ingress path. Typed broker requests may carry setup-session identity and one-time ingress handles, but never the secret bytes themselves.
- The canonical portable CLI path should be stdin-backed secret ingress, with an interactive prompt as a convenience over the same broker-owned flow when the CLI is attached to a terminal.
- The TUI should use a masked secret-entry control inside the existing shell and route architecture. It must not reuse the ordinary visible chat composer or any unmasked text-entry component for API keys or tokens.
- The broker remains the setup authority, stores long-lived direct credentials only in `secretsd`, and receives back secret metadata and custody posture rather than a second local credential cache.

## Adapter Families

### OpenAI-Compatible Family

- `v0` targets OpenAI-compatible Chat Completions as the reviewed direct-credential adapter surface.
- The adapter translates canonical `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` objects into and out of OpenAI-compatible wire payloads internally.
- Future OpenAI Responses support must be additive beneath the same provider-profile and auth-material substrate rather than forcing a new setup model or new control-plane request family.

### Anthropic-Compatible Family

- `v0` targets Anthropic-compatible Messages as the reviewed direct-credential adapter surface.
- The adapter translates canonical `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` objects into and out of Anthropic-compatible wire payloads internally.

### Shared Adapter Rules

- Popular SDKs and public provider docs may guide request and response translation, but they are not authoritative control-plane contracts.
- Provider-specific wire payloads remain implementation details unless a later typed extension is explicitly reviewed for policy, audit, or replay reasons.
- Auth-mode-specific behavior stays below shared provider-profile, readiness, and audit semantics so later OAuth and bridge lanes extend the same trusted setup model.

## Model Catalog And Endpoint Authority

- Manual allowlisted model IDs remain the canonical authority for what the control plane may select and route through a provider profile.
- Provider discovery and compatibility probes may surface advisory information such as reachable models, capability hints, or version posture, but they must not silently authorize new models or mutate canonical allowlists by themselves.
- Endpoint identity remains derived from the shared typed destination-descriptor model and shared `destination_ref` rules rather than raw provider URL authority.
- Validation and compatibility probes should confirm endpoint-family compatibility, auth acceptance, TLS posture, and supported feature shape without turning provider discovery APIs into the trust root.

## Compatibility And Readiness Posture

- Broker-projected provider posture should distinguish at least:
  - `configuration_state`
  - `credential_state`
  - `connectivity_state`
  - `compatibility_state`
  - `effective_readiness`
- Supported auth modes and the current auth mode should be surfaced explicitly in setup, inspection, and status flows.
- Reason codes should stay stable and typed so CLI and TUI clients do not have to infer setup posture from prose.
- Broker readiness and provider-profile readiness remain related but distinct surfaces: broker readiness summarizes subsystem availability, while provider-profile readiness explains whether a given configured provider is presently usable.

## CLI And TUI Surfaces

- Broker-visible setup and inspection contracts are the source of truth. CLI and TUI clients must stay thin adapters over those contracts.
- The user-facing RuneCode command surface should not let temporary `runecode-broker` subcommands become the long-term semantic source of provider setup behavior. Broker operations come first; higher-level CLI ergonomics wrap the same typed flows.
- The TUI should expose a first-class provider setup and inspection route using the existing root shell plus child-route architecture, summary-to-detail drill-down patterns, semantic theme tokens, and keyboard-first interaction model already established for the rest of the TUI.
- The TUI should show non-secret setup posture such as provider family, endpoint identity, auth modes, allowlisted models, capability metadata, compatibility posture, and last validation results without ever rendering raw credential material.
- The CLI should provide equivalent setup and inspection flows with machine-readable output for non-secret metadata and posture while still keeping secret entry on the trusted secret-ingress path.

## Foundation Shortcuts To Avoid

- Do not collapse provider setup into one global provider-family settings blob.
- Do not let ordinary broker JSON objects carry raw API keys or tokens.
- Do not make provider SDK objects, public API payloads, or vendor discovery endpoints the control-plane source of truth.
- Do not let provider discovery silently authorize new models or mutate allowlists.
- Do not let daemon-private state or TUI-local heuristics become the readiness authority.
- Do not use unmasked TUI input fields, CLI args, or environment variables for secrets.

## Main Workstreams
- Shared Provider Profile + Auth Material Model.
- Direct Credential Setup + Secret Custody.
- OpenAI-Compatible and Anthropic-Compatible Adapter Families.
- Broker/TUI/CLI Setup, Readiness, and Compatibility Surfaces.
- Policy, Quota, and Audit Reuse.

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
