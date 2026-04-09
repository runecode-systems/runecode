# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, change, and standards paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its v0.1.0-alpha.3 roadmap bucket and title after refinement.
- Confirm launcher/broker ownership is explicit and keeps the logical trust-boundary contract aligned with the broker local API rather than inventing a second runtime API.
- Confirm backend-neutral launch/session/attachment contract seams are explicit enough for container, macOS, Windows, durable-state, and attestation follow-on changes.
- Confirm `backend_kind`, runtime isolation assurance, provisioning/binding posture, and audit posture are separated rather than overloaded.
- Confirm QEMU, host paths, device numbering, and transport allocation details do not leak into boundary-visible logical contracts.
- Confirm guest image identity is descriptor-pinned and attachment planning stays topology-neutral and host-mount-free.
- Confirm isolate-session audit payloads and backend error taxonomy are explicit enough for later implementation and verification work to build on without inventing parallel vocabularies.

## Close Gate
Use the repository's standard verification flow before closing this change.
