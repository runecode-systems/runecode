# Tasks

## Shared Provider Substrate

- [ ] Define a shared provider-profile model that direct-credential, OAuth, and bridge-runtime lanes can all reuse.
- [ ] Ensure provider-profile identity stays stable across auth-mode changes, credential rotation, and validation retries.
- [ ] Support multiple configured provider profiles rather than one global provider-family settings blob.
- [ ] Define a shared auth-material model that distinguishes direct credentials from later derived or session-bound material without changing provider identity or readiness semantics.
- [ ] Keep provider-specific wire payloads below the canonical typed model boundary.
- [ ] Reuse shared destination identity, request binding, quota, and broker-projected posture semantics instead of introducing provider-local variants.
- [ ] Keep popular SDKs and public provider docs reference-only for adapter behavior rather than authoritative control-plane contracts.

## Direct Credential Setup

- [ ] Add a broker-owned setup-session model that separates non-secret provider metadata from secret ingress.
- [ ] Support trusted interactive setup for operator-entered endpoint configuration and API credentials.
- [ ] Keep raw secret values out of ordinary typed broker request and response bodies.
- [ ] Support CLI secret entry through the trusted secret-ingress flow without using CLI args or environment variables.
- [ ] Support TUI secret entry through a masked input flow that follows the existing Bubble Tea and Lip Gloss shell patterns.
- [ ] Store long-lived direct credentials only in `secretsd`.
- [ ] Lease scope-bound credential material to `model-gateway` for request execution.
- [ ] Forbid environment-variable and command-line secret injection.

## Adapter Families

- [ ] Add an OpenAI-compatible Chat Completions adapter family beneath the canonical model boundary.
- [ ] Add an Anthropic-compatible Messages adapter family beneath the canonical model boundary.
- [ ] Keep future OpenAI Responses support additive beneath the same provider substrate rather than requiring a setup-model rewrite.
- [ ] Keep auth-mode-specific behavior separate from shared provider-profile, readiness, and audit semantics.

## Model Catalog And Posture

- [ ] Keep manual allowlisted model IDs as the canonical authority for model selection.
- [ ] Keep provider discovery and compatibility probes advisory rather than authoritative for allowlist mutation or model authorization.
- [ ] Define broker-projected provider posture with explicit configuration, credential, connectivity, compatibility, and effective-readiness dimensions.
- [ ] Surface supported auth modes and current auth mode explicitly in inspection and setup results.

## Broker, TUI, and CLI Surfaces

- [ ] Add broker-owned setup, inspection, readiness, and compatibility surfaces for direct-credential providers.
- [ ] Surface supported auth modes and current compatibility posture in TUI and CLI flows.
- [ ] Keep CLI and TUI as thin adapters over the same broker-owned provider setup contracts.
- [ ] Keep provider setup, readiness, and error posture broker-projected rather than daemon-private.

## Acceptance Criteria

- [ ] RuneCode can use operator-entered OpenAI-compatible and Anthropic-compatible credentials for remote model access.
- [ ] Operators can enter endpoints and API credentials through either CLI or TUI flows without placing secret values in CLI args, environment variables, or ordinary typed broker request or response objects.
- [ ] Multiple provider profiles can coexist without redefining provider identity or setup authority.
- [ ] Direct credentials reuse the same provider substrate that later OAuth and bridge-runtime features will extend.
- [ ] `secretsd` remains the only long-lived credential store and `model-gateway` remains the canonical model-egress lane.
- [ ] Canonical typed model contracts, shared destination identity, quota handling, and broker-projected readiness remain unchanged by auth mode.
- [ ] OpenAI-compatible support lands on Chat Completions and Anthropic-compatible support lands on Messages, while future adapter expansion remains additive beneath the same provider substrate.
- [ ] Provider SDK request shapes and discovery APIs do not become authoritative control-plane contracts.
