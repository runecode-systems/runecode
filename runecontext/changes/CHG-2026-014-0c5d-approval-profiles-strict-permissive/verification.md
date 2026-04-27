# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the migrated change preserves the legacy task breakdown and acceptance criteria in `tasks.md`.
- Confirm canonical references remain on RuneContext project, spec, and change paths, with no active workflow depending on legacy planning paths.
- Confirm the migrated text assumes RuneContext is canonical, RuneCode owns the user-facing UX, and verified-mode project state remains the expected operating posture.
- Confirm the change still matches its v0.2 roadmap bucket and title after migration.
- Confirm `git_remote_ops` remains an exact-action hard-floor approval across all profiles and is not batchable into stage sign-off, milestone approval, or ambient acknowledgment.
- Confirm the minimum assurance floor for `git_remote_ops` remains at least `reauthenticated` across profiles.
- Confirm git remote-mutation approval bindings still require canonical repository identity, target refs, referenced patch artifact digests, expected result tree hash, and canonical action request hash.
- Confirm approval-profile expansion preserves the shared dependency-fetch checkpoint model rather than introducing per-cache-miss approval semantics for ordinary in-scope `fetch_dependency` work.
- Confirm any stricter or more permissive dependency-related approval behavior is expressed through canonical dependency-fetch scope and action semantics rather than ambiguous dependency-install or package-manager-local language.
- Confirm approval profiles do not weaken blocked project-substrate posture or override diagnostics/remediation-only repository substrate states.
- Confirm approval profiles govern formal approval timing only and do not replace shared `autonomy_posture` controls for operator-guidance cadence.
- Confirm profile semantics preserve the distinction between `waiting_approval` and `waiting_operator_input`.

## Close Gate
Use the repository's standard verification flow before closing this change.
