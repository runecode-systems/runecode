# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm first-party workflows use the shared workflow substrate rather than a hard-coded product-only execution path.
- Confirm drafting workflows operate on canonical RuneContext project state and keep outputs reviewable.
- Confirm approved-change implementation stays on the shared isolate-backed workflow path and reuses approval, audit, verification, and git semantics.
- Confirm live chat and autonomous entry surfaces trigger the same workflow pack.
- Confirm the roadmap and change text both place this feature in `v0.1.0-alpha.8`.

## Close Gate
Use the repository's standard verification flow before closing this change.
