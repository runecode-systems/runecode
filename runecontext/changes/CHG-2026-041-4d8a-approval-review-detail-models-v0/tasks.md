# Tasks

## Approval Detail Surface

- [x] Define richer approval detail/read-model expectations beyond list-facing summary fields.
- [x] Surface `policy_reason_code` directly.
- [x] Surface exact-action vs stage-sign-off binding kind explicitly.

## Structured Explanation

- [x] Define structured “what changes if approved” data.
- [x] Define blocked-work scope and bound-identity detail needed by the TUI.

## Lifecycle Detail

- [x] Define stale, superseded, expired, consumed, approved, and denied detail semantics through typed fields and reason codes.

## Acceptance Criteria

- [x] The alpha TUI can explain approval review from broker-projected typed data rather than payload scraping.
- [x] The model preserves exact-action vs stage-sign-off semantics cleanly.
