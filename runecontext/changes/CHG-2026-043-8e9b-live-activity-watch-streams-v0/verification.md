# Verification

## Planned Checks
- `runectx validate --json`
- `go test ./internal/protocolschema`
- `cd runner && node --test scripts/protocol-fixtures.test.js`

## Verification Notes
- Confirm the feature defines typed watch families for runs, approvals, and sessions.
- Confirm the feature keeps shared stream semantics aligned rather than inventing a second live model.
- Confirm all three watch families have positive stream fixtures that pass shared stream-sequence runtime rules.
- Confirm invalid double-terminal watch sequences fail under the same shared stream-sequence runtime rules.
