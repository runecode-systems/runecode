# Design

## Overview
Define the dedicated auth gateway role, provider-agnostic auth object families, and secret-safe OAuth-style login and refresh flows.

## Key Decisions
- Auth egress and model egress are separated.
- `secretsd` is the only long-lived secrets store; there is no second credential cache.
- Shared auth object families are provider-agnostic, typed, and versioned; provider specs extend them rather than redefining the control flow.
- No environment-variable or CLI-arg secret injection.
- Auth flows are typed, auditable, and fail closed on state/protocol mismatches.
- Auth egress should use the shared typed gateway destination/allowlist model so provider identity and allowed auth operations are expressed through logical canonical descriptors rather than raw URL decisions.
- Authoritative auth setup, account linking, and operator-facing posture are broker-owned typed control-plane concerns rather than daemon-local or provider-runtime-local APIs.

## Lease And Token Handoff

- `SecretLease` is the canonical short-lived token handoff contract for auth-derived credentials.
- `auth-gateway` may issue or renew auth-derived short-lived token material only through the reviewed `secretsd` lease boundary.
- `model-gateway` and later bridge/runtime consumers should consume short-lived auth material by lease identity rather than through custom provider-specific token-delivery paths.
- Long-lived refresh or session authority remains isolated to `secretsd`.
- Auth-derived leases should be scope-bound and action-bound to the reviewed destination identity, allowed operation set, and relevant action or policy hashes where the shared lease model supports that binding.

## Gateway Separation

- `auth-gateway` is the only gateway role for auth-provider egress.
- `auth-gateway` must not perform model inference or become a back door to general model-provider access.
- `model-gateway` must not perform login, code exchange, or token refresh in place.
- The auth lane should reuse the same canonical destination identity and shared gateway-operation vocabulary as the broader gateway foundation.
- The auth lane should also reuse the shared broker-owned setup/configuration posture so TUI and CLI remain thin adapters over one typed control-plane model.

## Auth Contract Shape

- Shared auth object families should stay provider-agnostic, typed, and versioned.
- Provider-specific flows may extend the shared auth families but must not replace them with raw provider payloads as the control-plane contract source of truth.
- Auth request-execution operations should be bindable to canonical typed request identity in the same way model request execution now binds `payload_hash` to `LLMRequest` identity.

## Operator Posture

- Daemon-local auth supervision is not the long-lived operator-facing API.
- Any user-facing or operator-facing auth posture should be broker-projected together with other subsystem posture rather than creating a second public truth source.
- Guided TUI setup and straightforward CLI setup are both expected, but both must remain thin clients of the same broker-owned typed auth and account-linking flows.
- Any manual token-entry fallback must remain a trusted interactive broker-mediated prompt path rather than an environment-variable or command-line injection path.

## Main Workstreams
- Auth Gateway Role Contract
- Provider-Agnostic Auth Objects
- Secret Handling + Token Storage
- Canonical lease-based token handoff
- Audit + Policy Integration

## RuneContext Migration Notes
- Canonical references now point at `runecontext/project/`, `runecontext/specs/`, and `runecontext/changes/` paths.
- Future-facing planning assumptions are rewritten to use RuneContext as the canonical planning substrate for this repository.
- Where this feature touches project context, approvals, assurance, or typed contracts, the migrated plan assumes bundled verified-mode RuneContext integration from the feature surface rather than a later retrofit.
