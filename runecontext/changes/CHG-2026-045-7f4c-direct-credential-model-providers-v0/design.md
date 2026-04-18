# Design

## Overview
Add direct-credential provider access on top of the existing `secretsd` and `model-gateway` foundation without creating a parallel provider stack.

## Key Decisions
- Direct credentials are one auth-material path within a shared provider substrate, not a separate provider architecture.
- Provider setup should distinguish provider identity, endpoint identity, supported auth modes, and model-capability metadata instead of flattening them into one untyped profile blob.
- OpenAI-compatible and Anthropic-compatible families should share as much trusted orchestration as possible while keeping provider-specific wire details below the canonical `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` boundary.
- `secretsd` remains the only long-lived credential store; `model-gateway` receives only scope-bound leased material.
- Endpoint identity remains on the shared typed destination and allowlist model rather than provider-local raw URL handling.
- Compatibility, readiness, and failure posture must be broker-projected and reusable by later OAuth and bridge-provider features.
- Direct credential entry must use trusted interactive setup flows; environment variables and command-line secret injection remain forbidden.

## Shared Provider Substrate

- Define a provider profile model that can represent at least:
  - provider family
  - canonical destination identity
  - supported auth modes
  - model capability metadata
  - compatibility posture
  - quota and usage-accounting posture
- Define an auth-material model that allows one provider profile to later use:
  - long-lived direct credentials
  - short-lived OAuth-derived material
  - bridge-runtime session material
- Keep provider adapters below the typed model boundary so later auth-mode expansion does not change canonical request, response, or stream contracts.

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
