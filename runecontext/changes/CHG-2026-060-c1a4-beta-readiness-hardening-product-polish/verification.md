# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm `runecontext/project/roadmap.md` places this change under `v0.1.0-alpha.11` and keeps `v0.1.0-beta.1` as the milestone framing.
- Confirm `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` is also reflected under `v0.1.0-alpha.11`.
- Confirm the proposal treats this lane as integration and dogfooding hardening, not a replacement architecture.
- Confirm the design requires one honest useful workflow path through the real trusted and untrusted execution path.
- Confirm the design requires production adoption of authoritative trusted `RunPlan` compilation and persistence.
- Confirm the design calls out runner transport and reporting integration rather than leaving noop/default runner transport ambiguous for the real path.
- Confirm the tasks explicitly capture TUI and operator polish discovered while testing.
- Confirm the change requires exercising evidence snapshot, record inclusion, bundle export, and offline verification on the real workflow path.
- Confirm the design keeps product messaging and assurance wording aligned with actual implementation state.

## Close Gate
Use the repository's standard verification flow before closing this change.
