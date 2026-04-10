# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the split preserves runner and durable-state requirements from the prior combined change.
- Confirm workspace roles and deterministic gates remain tracked as separate child features.
- Confirm the docs explicitly separate broker-owned run truth from runner-owned resumable orchestration detail.
- Confirm the docs freeze one stable logical workflow identity model plus separate attempt identities for retries and recovery.
- Confirm the docs require typed runner->broker checkpoint/result reporting and deterministic broker-wins reconciliation after restart.
- Confirm the docs preserve the approval split between exact-action approval and stage sign-off, including stage-summary supersession semantics.

## Close Gate
Use the repository's standard verification flow before closing this change.
