# Verification

## Checks Run
- ✅ `runectx validate --json`
  - Result: `ok` (exit 0), diagnostics: `0`.
- ✅ `runectx status --json`
  - Result: `ok` (exit 0); change metadata parsed successfully.
- ✅ `go test ./internal/brokerapi`
  - Result: `ok` (cached), including local API typed operation/stream/auth tests.
- ✅ `go test ./internal/protocolschema`
  - Result: `ok` (cached), covering schema bundle/manifest checks relevant to new broker API object families.

## Verification Notes
- Confirm the change defines a transport-neutral logical broker API rather than only a Unix-socket implementation sketch.
- Confirm the change defines first-class typed request/response/read-model/stream families for runs, approvals, artifacts, audit, readiness, and version info.
- Confirm run list and run detail are explicitly first-class operator-facing reads and not TUI-only shortcuts.
- Confirm approval identity, approval lifecycle states, and bound-scope metadata are specified consistently with the signed approval artifact model.
- Confirm broker approval semantics preserve the distinction between exact-action approvals and stage sign-off approvals, and do not treat `ApprovalBoundScope` as the trust root.
- Confirm broker read models stay topology-neutral and do not require host-local paths, socket names, usernames, or daemon-private storage details.
- Confirm `RunSummary` / `RunDetail` semantics keep `backend_kind`, runtime isolation assurance, provisioning/binding posture, and audit posture as distinct operator-facing concepts.
- Confirm `backend_kind` stays topology-neutral and does not drift into hypervisor/runtime implementation names.
- Confirm `assurance_level` is scoped to runtime isolation assurance rather than being overloaded with audit verification or provisioning posture meanings.
- Confirm authoritative backend/runtime facts are expected to come from trusted launcher/broker state rather than runner-local inference or audit-only derivation.
- Confirm stream semantics are explicit (`stream_id`, monotonic `seq`, exactly one terminal event, typed terminal failure).
- Confirm pagination, ordering, and error-taxonomy expectations are explicit enough for later TUI and protobuf work to reuse without redefining semantics.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the change still matches its v0.1.0-alpha.3 roadmap bucket and title after migration.

## Verification Outcome
- This branch contains substantial broker/local API implementation progress and matching protocol schema inventory in-tree.
- Change metadata remains `status: planned` and `verification_status: pending` because acceptance checklist items remain open in `tasks.md`.
- Full repository gate (`just ci`) was not rerun as part of this metadata/doc update; use the standard CI gate for final branch-wide validation.

## Close Gate
Use the repository's standard verification flow before closing this change.
