# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `cd runner && npm run boundary-check`
- `cd runner && node --test scripts/boundary-check.test.js`
- `just test`

## Verification Notes
- Confirm the split preserves runner and durable-state requirements from the prior combined change.
- Confirm workspace roles and deterministic gates remain tracked as separate child features.
- Confirm the docs explicitly separate broker-owned run truth from runner-owned resumable orchestration detail.
- Confirm the docs require the runner to consume a broker-compiled immutable `RunPlan` rather than plan workflows locally.
- Confirm the docs freeze one stable logical workflow identity model plus separate attempt identities for retries and recovery.
- Confirm the docs require typed runner->broker checkpoint/result reporting and deterministic broker-wins reconciliation after restart.
- Confirm the docs preserve the approval split between exact-action approval and stage sign-off, including stage-summary supersession semantics.
- Confirm the docs keep the runner thin and explicitly reject runner-local policy, approval, and workflow-planning authority.

## Close Gate
Use the repository's standard verification flow before closing this change.
