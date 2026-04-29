# Tasks

## Attestation Evidence Model

- [ ] Define the attestation evidence objects that upgrade an isolate or session from TOFU to an attested binding.
- [ ] Keep evidence bound at least to:
  - session identity
  - `session_nonce`
  - `handshake_transcript_hash`
  - signed runtime image descriptor identity
  - boot-profile identity
  - concrete launched boot component digests
  - launch-admission or equivalent trusted runtime-identity evidence where needed
  - prior provisioning evidence
- [ ] Preserve compatibility with the MVP audit/event envelope so previously recorded TOFU fields do not need a format break.
- [ ] Preserve compatibility with the `IsolateSessionBinding` model and the `isolate_session_started` / `isolate_session_bound` audit event families so attestation upgrades the same session-key model rather than replacing it.

## Launch, Verification, and Policy Integration

- [ ] Define how the launcher obtains and verifies attestation evidence before trusting the upgraded isolate binding, using the admitted signed runtime-asset identity as the baseline runtime truth.
- [ ] Distinguish TOFU-only, attested-valid, unavailable, and invalid attestation postures across policy and verification flows.
- [ ] Fail closed when an attested posture is required and evidence is invalid or replayed.

## TUI + Audit Posture

- [ ] Make provisioning posture explicit in audit metadata and TUI surfaces.
- [ ] Replace degraded TOFU-only posture with an attested posture only when verification succeeds.
- [ ] Record why an attested posture was unavailable or rejected without leaking sensitive local details.
- [ ] Keep attested runtime identity distinct from validated project-substrate identity when later audit or verification evidence references both.

## Fixtures + Cross-Platform Considerations

- [ ] Add checked-in fixtures for valid, invalid, replayed, and unavailable attestation evidence.
- [ ] Account for platform-specific attestation sources without making the shared verifier contract or operator-visible runtime identity semantics platform-specific.

## Acceptance Criteria

- [ ] RuneCode can represent and verify an attested isolate or session binding without changing the MVP audit/event contract.
- [ ] Verifiers and TUI distinguish TOFU-only, attested-valid, unavailable, and invalid attestation states.
- [ ] Replay and invalid-evidence cases fail closed when attestation is required.
