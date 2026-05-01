## Summary
RuneCode upgrades isolate provisioning from TOFU-only compatibility metadata to required measured attestation for the normal supported runtime path, without changing the core audit event families or introducing a second runtime architecture for constrained versus scaled deployments.

## Problem
The current runtime foundation already has the right seams for later attestation, but the actual attestation contract is still underspecified.

Today the codebase has:
- signed runtime-image identity and verified runtime-asset admission
- per-session isolate key binding with TOFU posture
- audit payloads that already reserve `attestation_evidence_digest`
- broker-projected posture fields that keep provisioning posture distinct from isolation posture

What is still missing is the durable reviewed contract for:
- what the canonical attestation evidence object is
- how attestation upgrades the existing session-key model without replacing it
- how replay detection and fail-closed verification work
- how operator-visible posture distinguishes valid, unavailable, invalid, and replayed evidence
- how the same trust path stays efficient on both small local devices and larger scaled deployments without forking the architecture

Without this change, RuneCode would either keep TOFU as the long-term normal posture for the first usable release or add attestation later through ad hoc backend-specific logic that rewrites runtime identity, audit semantics, or broker posture a second time.

## Proposed Change
- Define an additive immutable-record attestation model:
  - baseline session binding remains the same per-session isolate key model from `CHG-2026-009-1672-launcher-microvm-backend-v0`
  - new `IsolateAttestationEvidence` records capture attestation claims and freshness bindings
  - new `IsolateAttestationVerificationRecord` records trusted verification results against a reviewed verifier-policy state
- Bind attestation to persisted launch/runtime identity from the signed runtime-asset pipeline established by `CHG-2026-026-98be-image-toolchain-signing-pipeline`.
- Require valid attestation for all supported production and user-facing runtime paths in `v0.1.0-alpha.9` and fail closed on unavailable, invalid, replayed, or freshness-deficient evidence.
- Preserve TOFU only as a non-production compatibility mechanism for tests, fixtures, fake backends, and implementation scaffolding; no production, user-facing, or supported operator flow may use TOFU, whether by automatic fallback or manual override.
- Keep the existing `isolate_session_started` and `isolate_session_bound` audit event families and `IsolateSessionBinding` model, with attestation projected through additive evidence rather than a format break.
- Keep one topology-neutral architecture for constrained local devices and scaled deployments:
  - identical trust path
  - identical fail-closed behavior
  - identical audit semantics
  - different cache population, prewarming, or warm-pool mechanics only as private optimizations below the same contract
- Add explicit performance-oriented verification-result caching keyed by immutable identity rather than by environment-specific heuristics.

## Why Now
This work lands in `v0.1.0-alpha.9`, because the first usable release should not normalize TOFU-only provisioning as the steady-state trust posture.

Landing attestation after signed runtime-image identity but before the beta cut keeps the assurance model cumulative:
- signed runtime identity establishes what was supposed to launch
- attestation establishes what actually launched and bound to the session
- audit and verification preserve both without collapsing runtime identity into project identity

This is also the right point to lock the topology-neutral performance foundation so later warm caches, prewarming, and scaled deployment optimizations build on one reviewed trust path instead of creating separate small-device and large-deployment architectures.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode keeps the end-user command surface while using bundled RuneContext capabilities under the hood where project context or assurance is involved.
- Context-aware delivery for this feature is planned directly against verified-mode RuneContext rather than a later retrofit from legacy Agent OS semantics.
- The normal supported launch path should prioritize strict core tenets over convenience fallback behavior, and any TOFU use outside tests, fixtures, fake backends, or scaffolding should be treated as a defect rather than as a degraded supported posture.

## Out of Scope
- Introducing a second isolate identity model separate from the per-session isolate key.
- Introducing separate trust or launch architectures for constrained devices versus scaled deployments.
- Treating project-substrate identity as part of runtime identity.
- Making backend- or platform-specific attestation claims the public operator-facing contract.
- Allowing automatic fallback from required attestation to TOFU in supported runtime flows.
- Allowing manual production or user-facing override from required attestation to TOFU.

## Impact
This change locks the long-term isolate-attestation foundation before implementation:
- one runtime identity seam
- one session-binding model
- one fail-closed operator posture
- one audit/event contract
- one topology-neutral architecture from Raspberry Pi-class devices to larger scaled environments

That prevents another semantics rewrite later and gives future changes a durable attestation substrate for policy, audit, verification, anchoring, and performance work.
