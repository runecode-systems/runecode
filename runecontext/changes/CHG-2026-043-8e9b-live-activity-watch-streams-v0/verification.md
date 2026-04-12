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
- Implemented verification now includes:
  - `go test ./internal/brokerapi -run 'TestRunWatchStreamIncludesSnapshotUpsertAndTerminal|TestApprovalWatchStreamIncludesSnapshotAndTerminal|TestSessionWatchStreamIncludesSnapshotAndTerminal' -v`
  - `go test ./internal/protocolschema`
  - `cd runner && node --test scripts/protocol-fixtures.test.js`
  - `go test ./cmd/runecode-tui`
  - `just ci`
- The TUI foundation consumes typed watch families through the dashboard route and keeps logs as supplementary rather than authoritative live control-plane state.
