# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change preserves the legacy task breakdown and acceptance criteria while expanding them into an implementation-ready design.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its `v0.1.0-alpha.9` roadmap bucket and title after migration.
- Confirm all supported production and user-facing runtime paths now treat valid attestation as required rather than normalizing TOFU as the steady-state posture.
- Confirm the change explicitly disallows automatic fallback, manual override, default configuration, documented operator flow, CLI flag, TUI action, or constrained-device exception that would permit TOFU in supported runtime flows.
- Confirm TOFU is retained only as a non-production compatibility mechanism for tests, fixtures, fake backends, and scaffolding rather than as the reviewed operator posture.
- Confirm attestation inherits runtime identity from the signed runtime-asset pipeline rather than from ambient platform-specific launch assumptions.
- Confirm the design uses additive immutable records:
  - baseline session binding evidence
  - attestation evidence
  - trusted attestation verification records
- Confirm attestation evidence binds to:
  - session identity
  - `session_nonce`
  - `handshake_transcript_hash`
  - isolate session key identity
  - persisted launch/runtime evidence identity
  - signed runtime-image descriptor identity
  - boot-profile identity
  - concrete launched boot-component digests
- Confirm replay and freshness are defined as fail-closed verification requirements rather than advisory diagnostics.
- Confirm coarse provisioning posture remains separate from detailed attestation posture.
- Confirm attestation verification-result caching is keyed by immutable identity and described as a performance optimization that does not weaken trust semantics.
- Confirm the same logical attestation architecture is preserved for constrained local devices and larger scaled deployments without introducing separate trust paths.
- Confirm attested runtime identity remains distinct from validated project-substrate snapshot identity even where later evidence may bind both.
- Confirm existing audit event families remain compatible and `attestation_evidence_digest` remains the additive linkage seam.

## Close Gate
Use the repository's standard verification flow before closing this change.
