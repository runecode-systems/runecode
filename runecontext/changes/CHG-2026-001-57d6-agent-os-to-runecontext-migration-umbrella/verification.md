# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`

## Verification Notes
- Confirm decision frontmatter validates against the current decision schema.
- Confirm umbrella change status links to all Phase 1 decision records.
- Confirm migration governance assumptions are now captured in durable RuneContext artifacts.
- Confirm `runecontext/project/mission.md`, `runecontext/project/roadmap.md`, and `runecontext/project/tech-stack.md` replace the legacy product-doc role.
- Confirm repo-level references now treat `runecontext/project/*` as canonical product-doc paths, with no active workflow depending on legacy product-doc paths.

## Close Gate
Use the repository's standard verification flow before closing this change.
