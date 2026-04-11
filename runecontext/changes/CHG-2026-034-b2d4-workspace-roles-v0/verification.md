# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `cd runner && npm run boundary-check`
- `just test`

## Verification Notes
- Confirm the split preserves workspace-role capability and offline-boundary requirements from the prior combined change.
- Confirm role execution remains aligned with launcher, broker, and policy constraints.
- Confirm the docs freeze one reviewed role-to-executor matrix rather than leaving executor authority implicit.
- Confirm the docs require one trusted authoritative executor registry with only a read-only runner projection.
- Confirm the docs define non-shell-passthrough as typed reviewed executors, not freeform shell strings or wrapper chains.
- Confirm the docs keep `role_kind` and `executor_class` as separate shared concepts.
- Confirm the docs bind workspace execution to `RunPlan` executor entries rather than workflow-local executor selection.

## Close Gate
Use the repository's standard verification flow before closing this change.
