# Tasks

## Approval Detail Surface

- [ ] Define richer approval detail/read-model expectations beyond list-facing summary fields.
- [ ] Surface `policy_reason_code` directly.
- [ ] Surface exact-action vs stage-sign-off binding kind explicitly.

## Structured Explanation

- [ ] Define structured “what changes if approved” data.
- [ ] Define blocked-work scope and bound-identity detail needed by the TUI.

## Lifecycle Detail

- [ ] Define stale, superseded, expired, consumed, approved, and denied detail semantics through typed fields and reason codes.

## Acceptance Criteria

- [ ] The alpha TUI can explain approval review from broker-projected typed data rather than payload scraping.
- [ ] The model preserves exact-action vs stage-sign-off semantics cleanly.
