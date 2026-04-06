# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change defines a transport-neutral logical broker API rather than only a Unix-socket implementation sketch.
- Confirm the change defines first-class typed request/response/read-model/stream families for runs, approvals, artifacts, audit, readiness, and version info.
- Confirm run list and run detail are explicitly first-class operator-facing reads and not TUI-only shortcuts.
- Confirm approval identity, approval lifecycle states, and bound-scope metadata are specified consistently with the signed approval artifact model.
- Confirm broker read models stay topology-neutral and do not require host-local paths, socket names, usernames, or daemon-private storage details.
- Confirm stream semantics are explicit (`stream_id`, monotonic `seq`, exactly one terminal event, typed terminal failure).
- Confirm pagination, ordering, and error-taxonomy expectations are explicit enough for later TUI and protobuf work to reuse without redefining semantics.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the change still matches its v0.1.0-alpha.3 roadmap bucket and title after migration.

## Close Gate
Use the repository's standard verification flow before closing this change.
