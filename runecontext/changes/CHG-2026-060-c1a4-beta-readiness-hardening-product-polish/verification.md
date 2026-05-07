# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./...`
- `cd runner && npm run lint`
- `cd runner && npm test`
- `cd runner && npm run boundary-check`
- `just test`
- `just ci`

## Required Product Smokes
- Project-substrate lifecycle smoke: inspect/posture, adopt compatible existing substrate, init preview/apply for missing substrate, upgrade preview/apply for supported older substrate, and validate/status after apply.
- Workflow smoke: run `change_draft` and `spec_draft` through the real trusted `RunPlan` and runner path.
- Promote/apply smoke: promote a reviewed change draft into `runecontext/changes/` and a reviewed spec draft into `runecontext/specs/` through the shared audited mutation path.
- Implementation smoke: run `approved_change_implementation` from one reviewed implementation input set and verify resulting local workspace mutation plus required RuneContext lifecycle metadata updates when included in the approved input.
- Evidence smoke: inspect run/session/artifact/approval/audit surfaces, capture an evidence snapshot, verify at least one record-inclusion result, export an evidence bundle, and verify the bundle offline.
- External anchoring smoke: exercise external audit anchoring on the real workflow path where environment and policy allow.
- Git publication posture: confirm beta messaging does not claim push, pull request, prompt-to-PR, or team-collaboration publishing unless a separate reviewed git remote smoke path is added.

## Verification Notes
- Confirm `runecontext/project/roadmap.md` places this change under `v0.1.0-alpha.11` and keeps `v0.1.0-beta.1` as the milestone framing.
- Confirm `CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0` is reflected consistently wherever beta assurance wording depends on its verified integration.
- Confirm the proposal treats this lane as integration and dogfooding hardening, not a replacement architecture.
- Confirm the design requires canonical project-substrate lifecycle proof through broker-owned product surfaces.
- Confirm the design requires `change_draft`, `spec_draft`, `draft_promote_apply`, and `approved_change_implementation` through the real trusted and untrusted execution path.
- Confirm the design requires production adoption of authoritative trusted `RunPlan` compilation and persistence.
- Confirm the design calls out runner transport and reporting integration rather than leaving noop/default runner transport ambiguous for the real path.
- Confirm run/session/TUI projections are plan-authoritative for the supported path rather than artifact-inferred.
- Confirm approved implementation can carry required RuneContext lifecycle metadata updates without creating a separate fifth workflow operation for this lane.
- Confirm the tasks explicitly capture TUI and operator polish discovered while testing.
- Confirm the change requires exercising evidence snapshot, record inclusion, bundle export, and offline verification on the real workflow path.
- Confirm the design keeps product messaging and assurance wording aligned with actual implementation state.
- Confirm git remote publication remains out of required beta messaging unless a separate reviewed publishing smoke path is added.

## Close Gate
Use the repository's standard verification flow before closing this change, with `just ci` as the parity gate after targeted workflow and product smokes pass.
