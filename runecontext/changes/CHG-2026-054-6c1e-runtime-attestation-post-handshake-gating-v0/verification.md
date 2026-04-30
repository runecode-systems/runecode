# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change keeps RuneContext change docs as the canonical planning surface.
- Confirm the roadmap entry is added under `v0.1.0-beta.1` and remains outcome-focused.
- Confirm the design preserves the reviewed trust ordering from isolate attestation instead of redefining the attestation model.
- Confirm `attested` is described as unavailable before secure-session validation and post-handshake trusted verification.
- Confirm the change does not redefine runtime identity away from the signed runtime-asset pipeline.
- Confirm the change does not move attestation authority into the runner or any untrusted component.
- Confirm the change preserves one backend-neutral and topology-neutral trust path across constrained and scaled deployments.
- Confirm the tasks require persistence, restart reconstruction, broker projection, and audit timing updates rather than only launcher-local checks.

## Close Gate
Use the repository's standard verification flow before closing this change.
