# Design

## Overview
This change hardens RuneCode's attested runtime path so `attested` is only awarded after a live runtime participates in the secure-session flow and after trusted code verifies runtime-side evidence collected for that exact session.

The key correction is ordering, not a new trust surface.

## Key Decisions
- Keep the runtime trust path defined in `CHG-2026-030-98b8-isolate-attestation-v0` and make the implementation obey it strictly:

`publish -> sign -> trusted admission -> verified local cache -> launch -> secure session -> collect runtime-side proof/attestation -> trusted verification -> persisted evidence -> broker projection -> audit and verification`

- `attested` must not be set from launcher-generated binding material alone.
- Secure session remains a required prerequisite to supported attestation rather than a parallel optional signal.
- Runtime-side proof must bind to the same session tuple and admitted runtime identity seam already established by signed runtime assets and session binding.
- Audit, artifacts, and broker projection remain additive; this change corrects when evidence is earned, not the overall evidence-first model.
- One architecture must hold across microVM and container backends and across constrained versus scaled deployments.

## Goals
- Prevent launcher-owned synthetic values from being sufficient to produce supported `attested` posture.
- Wire the reviewed secure-session verifier into the launcher runtime lifecycle.
- Introduce a reviewed runtime-side evidence collection seam after handshake and before attestation success is persisted.
- Preserve restart-safe reconstruction from persisted evidence and verification outputs.
- Keep operator-facing runtime identity semantics stable while making their provenance stronger.

## Non-Goals
- Replacing the session-binding model.
- Replacing signed runtime-image admission as the primary runtime identity source.
- Publishing backend-vendor-specific raw attestation claims as the stable operator contract.
- Introducing a separate deployment-size-specific launch or verification architecture.

## Architecture

### Current Gap
Today the implementation can produce attestation verification outputs during receipt construction, before any runtime-side handshake or proof path has been exercised.

That ordering is weaker than the reviewed model because the same trusted process that launched the runtime can also populate enough fields to appear attested.

### Required Flow
The required beta flow is:
1. Launch runtime from admitted signed runtime assets.
2. Establish secure session using reviewed trusted handshake validation.
3. Produce a persisted session-binding result for the live session.
4. Collect runtime-side proof or attestation evidence bound to that validated live session.
5. Verify that evidence in trusted Go against admitted runtime identity, freshness rules, and replay rules.
6. Persist evidence and verification results.
7. Project broker posture and audit outputs from the persisted results.

### Launch Gating Rule
Until steps 2 through 5 succeed, supported runtime posture must not be projected as `attested`.

Allowed intermediate behavior:
- the launch may be in progress
- a session may be partially established
- evidence may be collected but not yet verified

Disallowed behavior:
- marking the runtime validly attested before secure-session validation
- marking the runtime validly attested before runtime-side evidence is collected
- marking the runtime validly attested before trusted verification succeeds

### Session And Evidence Binding
The post-handshake evidence seam must bind at least to:
- `{run_id, isolate_id, session_id}`
- `session_nonce`
- `launch_context_digest`
- `handshake_transcript_hash`
- `isolate_session_key_id_value`
- persisted launch/runtime evidence digest from the signed runtime-asset path
- admitted runtime-image descriptor identity
- boot-profile identity
- concrete launched boot-component digests
- normalized attestation source kind
- measurement profile and freshness claims

### Persistence And Projection
Persisted evidence remains the source for restart-time reconstruction.

This change should ensure that:
- launcher transient state is not treated as the durable attestation source of truth
- broker read models continue to prefer persisted evidence snapshots
- audit event families remain compatible, with `attestation_evidence_digest` as the additive linkage seam

### Backend-Neutral Contract
MicroVM and container backends may collect different concrete raw proof material, but both must satisfy the same trusted sequence:
- live session validation first
- runtime-side proof second
- trusted attestation verification third
- persisted evidence and posture projection last

### Failure Semantics
If secure session does not validate, the runtime does not qualify for supported attested posture.

If runtime-side proof or attestation evidence is missing, stale, replayed, or invalid after handshake, the runtime does not qualify for supported attested posture.

No fallback from those failures to supported TOFU is allowed.

## Implementation Surfaces
- `internal/launcherdaemon/` launch lifecycle and runtime update sequencing
- `internal/launcherbackend/` secure-session validation, evidence contracts, and attestation verification integration
- `internal/artifacts/` persistence of post-handshake evidence and verification outputs
- `internal/brokerapi/` authoritative posture projection and audit timing
- tests and fixtures that currently assume launcher-side attestation success during receipt construction

## Acceptance Shape
The implementation is complete when the only path to supported `attested` posture is:
- admitted runtime launch
- validated secure session
- collected runtime-side proof/evidence
- trusted verification success
- persisted evidence-based projection
