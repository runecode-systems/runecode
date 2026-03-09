---
name: verification-gate-runecode
description: Run the standard verification gate for this RuneCode repository and report pass or fail results clearly.
argument-hint: "[optional targeted tests or extra verification commands]"
disable-model-invocation: true
---

Run this verification workflow before handoff or merge.

## Procedure

1. Run baseline checks:
   - `just ci`
2. Run targeted checks for changed areas when useful:
   - Go-focused changes: `go test ./...`
   - Runner-focused changes: `cd runner && npm run lint && npm test && npm run boundary-check`
3. For trust-boundary changes, ensure boundary checks are explicitly covered:
   - `cd runner && npm run boundary-check`
   - `cd runner && npm test`
4. If a check fails:
   - Capture failure details.
   - Apply safe, scoped fixes.
   - Re-run the failed checks.
   - Re-run `just ci` before final handoff.
5. Report results in a command matrix:
   - Command
   - Purpose
   - Status (`pass` or `fail`)
   - Notes on retries, environment limits, or follow-up

## Guardrails

- Never claim a check passed unless it was executed in this session.
- If a check cannot run, report it explicitly and provide an exact local reproduce command.
- Do not skip baseline verification unless the user explicitly requests it.
