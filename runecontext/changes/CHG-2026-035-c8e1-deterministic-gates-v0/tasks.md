# Tasks

## Gate Framework

- [ ] Implement build, test, lint, format, secret-scan, and policy gates.
- [ ] Keep execution deterministic and reproducible.

## Evidence Artifacts

- [ ] Emit hash-addressed evidence artifacts for every gate run.
- [ ] Link evidence into audit and workflow records.

## Failure + Override Semantics

- [ ] Fail closed on gate failures by default.
- [ ] Record retries and require explicit approvals for overrides.
- [ ] Keep override semantics aligned with canonical policy `ActionRequest` / `PolicyDecision` identity and shared approval trigger semantics.

## Acceptance Criteria

- [ ] Gate outcomes are deterministic and auditable.
- [ ] Overrides are explicit, policy-controlled, and evidence-backed.

## Executor Classification Hardening (Pre-MVP Follow-up)

- [ ] Harden system-modifying command detection used by policy hard-floor classification:
  - extend launcher/wrapper normalization beyond the current minimal set,
  - or adopt conservative full-argv classification that cannot be bypassed through wrapper indirection.
- [ ] Add deterministic regression fixtures for representative wrapper-chaining forms to ensure fail-closed behavior remains stable.
