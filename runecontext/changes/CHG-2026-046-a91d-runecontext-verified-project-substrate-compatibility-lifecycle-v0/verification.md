# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just lint`
- `just test`

## Verification Notes
- Confirm the change preserves canonical repo-root `runecontext.yaml` plus canonical `runecontext/` substrate and does not introduce a RuneCode-only mirror or daemon-private project store.
- Confirm RuneCode initialization remains aligned with the canonical `runectx init` substrate shape for the selected substrate version.
- Confirm discovery is deterministic at one authoritative repository root and does not rely on arbitrary upward search for the nearest `runecontext.yaml`.
- Confirm adoption is read-only and does not silently normalize or rewrite discovered project substrate.
- Confirm adopt-existing, initialize-new, and upgrade flows all remain compatible with direct RuneContext usage.
- Confirm RuneCode is the hard compatibility gate for managed repos while RuneContext remains generic and machine-friendly.
- Confirm compatibility is evaluated against the repository's declared and validated substrate contract rather than against each developer's installed RuneCode or `runectx` version.
- Confirm supported-current and supported-with-upgrade-available posture remain usable while missing, invalid, non-verified, unsupported-too-old, and unsupported-too-new posture remain blocked for normal RuneCode operation.
- Confirm upgrades are explicit, previewable, and auditable, and that normal startup or status paths never auto-upgrade project substrate.
- Confirm broker diagnostics can surface supported substrate ranges, active project posture, stable reason codes, blocked reasons, and recommended remediation.
- Confirm project-context binding reaches audit, attestation, and verification surfaces through validated snapshot identity where relevant.
- Confirm future dashboard/operator decisions are still modeled as broker-owned typed contracts rather than client-authoritative setup state.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.6`.

## Close Gate
Use the repository's standard verification flow before closing this change.
