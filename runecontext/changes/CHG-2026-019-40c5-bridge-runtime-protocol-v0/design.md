# Design

## Overview
Define the shared bridge/runtime contracts for user-installed provider runtimes in explicit LLM-only mode.

## Key Decisions
- Shared bridge/runtime object families are defined once and reused by later provider specs.
- Compatibility is probe-driven and fail-closed; newer vendor versions are not trusted implicitly.
- Bridge runtimes remain LLM-only and never receive workspace or patch capabilities.
- Token delivery must avoid environment variables and raw secret logging.
- Audit and TUI surfaces must make untested-version and persisted-session posture visible.
- Bridge runtimes must not become their own setup, account-linking, or auth-status authority surface; those remain broker-owned.

## Canonical Boundary Inheritance

- Bridge runtimes live below the canonical `LLMRequest`, `LLMResponse`, and `LLMStreamEvent` contracts.
- Bridge/runtime adapters may translate those typed contracts into provider-specific runtime protocols, but they should not replace the canonical model boundary with bridge-native payloads.
- Provider-specific runtime payloads remain implementation details unless a later typed extension is required for policy, audit, or replay semantics.

## Token Delivery And Lease Use

- Bridge runtimes should receive short-lived auth-derived material only through the reviewed lease boundary.
- Bridge token delivery must not become a second credential cache or a second canonical custody contract.
- Token handoff should be by canonical lease identity through a trusted local channel rather than environment variables, CLI args, or ad hoc provider-runtime credential files by default.

## Destination Identity And Quotas

- Bridge integrations should inherit canonical destination identity from the shared destination descriptor and `destination_ref` model rather than inventing bridge-local URL semantics.
- Bridge integrations should also inherit the shared trusted quota model so provider-specific runtime adapters do not redefine usage accounting semantics.
- Runtime-reported usage and provider headers are advisory inputs to trusted quota handling rather than sole authority.
- Bridge integrations should inherit the shared broker-owned setup and account posture model so TUI and CLI remain thin adapters over one typed control-plane path.

## Operator Posture

- Bridge-specific health or compatibility probes may remain local supervision and diagnostics inputs.
- Any long-lived operator-facing posture should be broker-projected together with the rest of the secure model-provider access stack rather than exposed as a second daemon-style public API.
- Provider bootstrap, account state, and auth posture should surface through broker-owned typed setup and status APIs rather than runtime-local configuration screens or files.

## Main Workstreams
- Bridge Runtime Contract
- Compatibility + Probe Model
- Token Delivery + Session Rules
- Audit + UX Surfaces
- Canonical model-boundary inheritance

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
