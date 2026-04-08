# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its v0.1.0-alpha.3 roadmap bucket and title after migration.
- Confirm the design freezes effective policy-context composition and `manifest_hash` semantics around one compiled policy context.
- Confirm the change records one canonical `ActionRequest` model, a closed `action_kind` registry, and a shared trusted evaluation boundary rather than leaving action identity implicit.
- Confirm role taxonomy, typed gateway destination/allowlist modeling, and hard-floor assurance taxonomy are explicit enough for later gateway and runner work to build on without inventing parallel policy vocabularies.
- Confirm approval semantics distinguish exact-action approval from stage sign-off, including stage-summary-hash supersession semantics.
- Confirm the change separates `policy_reason_code`, `approval_trigger_code`, and `error.code`, and that successful deny outcomes remain `PolicyDecision` artifacts rather than ad hoc failures.
- Confirm executor/dependency semantics distinguish offline workspace execution from gateway/network or system-modifying behavior.

## Close Gate
Use the repository's standard verification flow before closing this change.
