# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `go test ./internal/policyengine`
- `go test ./internal/launcherbackend`
- `go test ./internal/launcherdaemon`
- `go test ./internal/brokerapi`
- `go test ./internal/protocolschema`
- `cd runner && npm run boundary-check`
- `just test`
- `just ci`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change matches its `v0.1.0-alpha.4` roadmap bucket and title after refinement.
- Confirm the design captures the decided instance-scoped selection model for the active running RuneCode instance.
- Confirm the design now requires a first-class launcher `instance_id` and binds backend-posture approvals to that runtime instance rather than only to `run_id`.
- Confirm the design scopes initial implementation to offline workspace-role launches only and does not blur role-family separation or gateway-role networking into the initial container slice.
- Confirm the design keeps the TUI/operator experience uniform across backends and treats backend choice as shared posture metadata rather than a second UX flow.
- Confirm the design requires generic exact-action approval issuance and consumption for reduced-assurance backend opt-in rather than a container-specific approval pathway.
- Confirm the design adds a dedicated broker instance-control policy path rather than pretending backend posture changes are ordinary run-scoped runtime actions.
- Confirm the design adds broker local API posture operations while preserving the broker local API as the only public/untrusted boundary.
- Confirm the design adds a private broker-to-launcher posture application/query contract rather than a direct TUI-to-launcher path.
- Confirm the design treats missing baseline container hardening controls as launch failures, not as an acceptable second layer of reduced assurance.
- Confirm the design preserves the no-host-mount invariant for container realization and keeps host-local paths out of schemas, audit payloads, and degraded-reason strings.
- Confirm the design records broker-owned backend posture application audit evidence distinct from later run launches.
- Confirm the task list captures replay-prevention on launcher restart, conformance coverage, and end-to-end checks for approval binding, posture reset, networking denial, and bind-mount absence.
- Confirm shared backend/runtime posture stays split across backend kind, runtime isolation assurance, provisioning/binding posture, audit posture, and backend-specific implementation evidence.
- Confirm standards and references cover trust-boundary, policy, approval-binding, broker-contract, runtime-evidence, and protocol-discipline concerns touched by this change.

## Close Gate
Use the repository's standard verification flow before closing this change.
