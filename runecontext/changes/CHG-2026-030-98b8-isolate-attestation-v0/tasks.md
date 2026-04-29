# Tasks

## Attestation Evidence Model

- [ ] Define `IsolateAttestationEvidence` as an immutable trusted-domain evidence family rather than overloading the existing session binding record.
- [ ] Define `IsolateAttestationVerificationRecord` as a separate immutable trusted verification result keyed to one attestation evidence digest plus verifier-policy identity.
- [ ] Keep the existing per-session isolate key binding as the durable isolate identity root.
- [ ] Preserve compatibility with the `IsolateSessionBinding` model and the `isolate_session_started` / `isolate_session_bound` audit event families so attestation upgrades the same session-key model rather than replacing it.
- [ ] Keep evidence bound at least to:
  - `{run_id, isolate_id, session_id}`
  - `session_nonce`
  - `handshake_transcript_hash`
  - `isolate_session_key_id_value`
  - persisted launch/runtime evidence identity from the signed runtime-asset pipeline
  - signed runtime image descriptor identity
  - boot-profile identity
  - concrete launched boot component digests
  - normalized attestation source kind
  - `measurement_profile`
  - source freshness material and freshness-binding claims
- [ ] Keep attestation evidence and runtime identity distinct from validated project-substrate identity.
- [ ] Preserve compatibility with the MVP audit/event envelope so previously recorded TOFU fields do not need a format break.

## Measurement Profile Contract

- [ ] Turn `measurement_profile` into a closed reviewed vocabulary owned by trusted Go code.
- [ ] Define, for each profile:
  - required claims
  - canonicalization rules
  - accepted source classes
  - freshness requirements
  - normalized output shape
  - normalized digest derivation rules
- [ ] Define `expected_measurement_digests` as exact allowed normalized measurement digests for the declared profile rather than as ambiguous partial matching.
- [ ] Keep platform-specific raw evidence details out of the shared operator-facing contract.

## Replay And Freshness Enforcement

- [ ] Define the canonical replay identity for attestation verification.
- [ ] Require replay checks to bind at least to:
  - `{run_id, isolate_id, session_id}`
  - `session_nonce`
  - `handshake_transcript_hash`
  - `isolate_session_key_id_value`
  - launch/runtime evidence identity
  - attestation evidence identity
  - `measurement_profile`
- [ ] Fail closed when an attested posture is required and evidence is replayed, stale, or missing required freshness material.
- [ ] Persist replay-relevant identity and verification outputs so restart-time reconstruction does not rely on in-memory state.

## Launch, Verification, and Policy Integration

- [ ] Define how the launcher obtains attestation evidence after secure session establishment and before treating the session as valid for the supported attested posture.
- [ ] Define trusted verification against the admitted signed runtime-asset identity as the baseline runtime truth.
- [ ] Use persisted launch/runtime evidence identity from the signed runtime-asset pipeline as the primary binding seam rather than ambient backend launch assumptions.
- [ ] Require valid attestation for all supported production and user-facing runtime paths in `v0.1.0-alpha.9`.
- [ ] Keep TOFU only as a non-production compatibility mechanism for tests, fixtures, fake backends, and implementation scaffolding.
- [ ] Disallow automatic fallback, manual override, default configuration, documented operator flow, CLI flag, TUI action, or constrained-device exception that would permit TOFU trust decisions in production or user-facing runtime flows.
- [ ] Distinguish coarse provisioning posture from detailed attestation posture across policy and verification flows.
- [ ] Keep the coarse provisioning posture model as:
  - `tofu`
  - `attested`
  - `not_applicable`
  - `unknown`
- [ ] Add a detailed trusted/operator attestation posture that can distinguish at least:
  - `tofu_only`
  - `valid`
  - `unavailable`
  - `invalid`
  - `not_applicable`
  - `unknown`
- [ ] Represent replay as an `invalid` attestation result with a dedicated machine reason code rather than as a benign degraded state.
- [ ] Fail closed when an attested posture is required and evidence is unavailable, invalid, or replayed.

## Verification Result Caching And Restart Safety

- [ ] Cache attestation verification results by immutable identity, at least:
  - `attestation_evidence_digest`
  - verifier authority state digest
  - `measurement_profile`
- [ ] Ensure warm-path cache hits reduce repeated verification cost without bypassing first-use verification.
- [ ] Ensure cache invalidates automatically when verifier authority state or normalized evidence identity changes.
- [ ] Reconstruct authoritative posture after restart from persisted evidence and cached verification results rather than transient launcher or broker memory.

## TUI + Audit Posture

- [ ] Make provisioning posture explicit in audit metadata and TUI surfaces.
- [ ] Make detailed attestation posture explicit in trusted/operator surfaces so evidence-exists, evidence-valid, and evidence-rejected are not collapsed together.
- [ ] Replace TOFU-only posture with an attested posture only when verification succeeds.
- [ ] Record why attestation was unavailable or rejected without leaking sensitive local details.
- [ ] Keep attested runtime identity distinct from validated project-substrate identity when later audit or verification evidence references both.
- [ ] Preserve `attestation_evidence_digest` as the additive linkage field in existing audit event families.
- [ ] Update runtime audit dedupe identity so a later successful attestation upgrade is treated as a material posture change rather than suppressed behind the original TOFU marker.
- [ ] Ensure operator-facing explanations can distinguish:
  - baseline session binding exists
  - attestation evidence exists
  - attestation verification succeeded or failed

## Fixtures + Cross-Platform Considerations

- [ ] Add checked-in fixtures for:
  - valid attestation evidence
  - invalid attestation evidence
  - replayed attestation evidence
  - unavailable attestation evidence
  - stale or freshness-missing evidence
- [ ] Add fixtures that prove attestation binds to signed runtime-image identity, boot-profile identity, and concrete boot-component digests.
- [ ] Account for platform-specific attestation sources without making the shared verifier contract or operator-visible runtime identity semantics platform-specific.
- [ ] Keep the same logical attestation architecture for constrained local devices and scaled deployments.
- [ ] Restrict deployment-specific differences to private optimizations such as verified-cache population, prewarming, or warm pools.
- [ ] Ensure performance work does not bypass signature verification, attestation verification, replay checks, or evidence persistence.

## Acceptance Criteria

- [ ] RuneCode can represent and verify an attested isolate or session binding without changing the MVP audit/event contract.
- [ ] All supported production and user-facing runtime paths require valid attestation and do not silently or explicitly fall back to TOFU.
- [ ] TOFU remains only as non-production scaffolding for tests, fixtures, fake backends, and implementation work rather than as a supported operator posture or runtime option.
- [ ] No release-mode policy, default configuration, documented operator flow, CLI flag, TUI action, or constrained-device exception permits TOFU trust decisions.
- [ ] Verifiers and TUI distinguish TOFU-only, valid, unavailable, and invalid attestation states.
- [ ] Replay and invalid-evidence cases fail closed when attestation is required.
- [ ] Attestation binds to the admitted signed runtime identity seam rather than to ambient platform assumptions.
- [ ] Restart-time authoritative posture remains reconstructible from persisted evidence and immutable verification-cache identity.
- [ ] The same logical attestation architecture remains suitable for constrained local devices and larger scaled deployments without separate trust-path forks.
