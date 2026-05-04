## Summary
Move supported attested runtime posture to the reviewed order already intended by the isolate-attestation design: launch, establish secure session with runtime-side proof, collect attestation evidence after that live session is established, then perform trusted verification and only then treat the runtime as validly attested.

## Problem
`CHG-2026-030-98b8-isolate-attestation-v0` defined the right trust path, but the current implementation still marks attestation valid from launcher-produced receipt material before a real runtime-side secure-session exchange or runtime-produced proof exists.

That leaves one important semantic gap:
- the code distinguishes signed runtime identity, session binding, and attestation fields
- but the current attested posture can still be produced by launcher-owned values alone
- the secure-session verifier exists as reviewed trusted code, yet it is not wired into the launcher runtime flow that gates attested posture

Without a follow-up change, RuneCode would keep an implementation shape where `attested` can appear before the runtime has actually proven boot/bind participation in the live session. That weakens the intended meaning of the normal supported attested path even though the surrounding evidence model is now stronger.

## Proposed Change
- Require the launcher runtime flow to complete trusted secure-session validation before attestation can upgrade a runtime to supported `attested` posture.
- Define the runtime-side proof and evidence collection seam that occurs after secure session establishment and before launch facts are persisted as valid attested evidence.
- Keep the existing signed runtime-asset admission pipeline as the primary runtime identity source; runtime-side proof must bind to that identity rather than replacing it.
- Keep attestation additive to the existing session-binding model rather than creating a new parallel identity contract.
- Make `attested` unavailable until all of the following are true:
  - secure session is established and validated by trusted Go code
  - runtime-side proof or attestation material is collected for the current live session
  - trusted verification succeeds against admitted runtime identity and freshness requirements
  - resulting evidence and verification outputs are persisted for restart-safe reconstruction
- Preserve one topology-neutral architecture across constrained local devices and larger deployments; the proof/evidence sequencing must not split into separate small-device and scaled-deployment trust paths.

## Why Now
This should land as a beta hardening follow-up rather than being folded into the portability/session-binding lane that just closed PR `#55` risks.

The current lane fixed portability and strengthened shared session-binding inputs, but wiring real runtime-side boot/bind proof into launch gating would widen across launcher lifecycle, handshake sequencing, evidence persistence, audit timing, and broker projection. That is a separate product change, not a small patch.

Scheduling it for `v0.1.0-alpha.11` keeps the work visible and reviewed in the explicit pre-beta hardening lane before the first beta assurance story is treated as settled.

## Assumptions
- The reviewed secure-session contract in trusted Go remains the authoritative validation boundary for runtime-side session proof.
- The launcher, not the runner or other untrusted code, remains responsible for deciding when attested posture is earned.
- Runtime-side evidence may differ by backend or platform, but the operator-visible semantics and trusted verification contract must remain backend-neutral.
- Existing additive evidence fields such as `attestation_evidence_digest` remain the compatibility seam for audit and broker projection.

## Out of Scope
- Redefining runtime identity away from the signed runtime-asset pipeline.
- Moving attestation authority into untrusted components.
- Adding a second operator-visible trust model for constrained devices versus scaled deployments.
- Broad performance optimization work beyond what is necessary to preserve the reviewed trust ordering.

## Impact
This change closes the remaining semantic gap between the reviewed attestation architecture and the current implementation order.

If completed, RuneCode's normal attested path will mean:
- admitted signed runtime identity says what should launch
- secure session proves the live runtime joined the expected session
- runtime-side attestation evidence says what actually booted or bound
- trusted verification upgrades the runtime to `attested`
- persisted evidence and broker projection preserve that result across restart, audit, and operator surfaces
