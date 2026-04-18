# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change preserves canonical `runecontext/` project state and does not introduce a RuneCode-only mirror.
- Confirm adopt-existing, initialize-new, and upgrade flows all remain compatible with direct RuneContext usage.
- Confirm RuneCode is the hard compatibility gate for managed repos while RuneContext remains generic and machine-friendly.
- Confirm broker diagnostics can surface supported RuneContext ranges, active project posture, and blocked reasons.
- Confirm project-context binding reaches audit, attestation, and verification surfaces where relevant.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.6`.

## Close Gate
Use the repository's standard verification flow before closing this change.
