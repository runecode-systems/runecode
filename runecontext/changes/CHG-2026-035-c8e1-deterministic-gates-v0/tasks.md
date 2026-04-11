# Tasks

## Gate Framework

- [ ] Implement build, test, lint, format, secret-scan, and policy gates.
- [ ] Keep execution deterministic and reproducible.
- [ ] Define one reusable typed gate contract with stable `gate_id`, `gate_kind`, `gate_version`, normalized inputs, and explicit retry/override semantics.
- [ ] Define explicit gate lifecycle states (`planned`, `running`, `passed`, `failed`, `overridden`, `superseded`) for runner/broker/audit alignment.
- [ ] Keep gate ordering and checkpoint placement explicit in workflow/process planning rather than implicit in executor-local scripts.
- [ ] Compile gate placements into immutable `RunPlan` entries and require runner execution to consume only those planned entries.

## Evidence Artifacts

- [ ] Emit hash-addressed evidence artifacts for every gate run.
- [ ] Link evidence into audit and workflow records.
- [ ] Introduce a dedicated typed gate-evidence object rather than relying only on generic logs.
- [ ] Introduce a dedicated gate-evidence artifact data class so policy, retention, and audit linkage can reason about gate evidence explicitly.
- [ ] Keep gate evidence reference-heavy:
  - typed evidence object as the canonical summary/binding layer
  - bulky stdout/stderr and related outputs as referenced artifacts
- [ ] Bind gate evidence to stable workflow scope identity plus a separate `gate_attempt_id` for retries and reruns.
- [ ] Bind gate evidence to the plan-derived gate placement identity so replay and audit can distinguish planned execution from stale or out-of-band attempts.

## Failure + Override Semantics

- [ ] Fail closed on gate failures by default.
- [ ] Record retries and require explicit approvals for overrides.
- [ ] Keep override semantics aligned with canonical policy `ActionRequest` / `PolicyDecision` identity and shared approval trigger semantics.
- [ ] Ensure retries create new gate attempts rather than mutating prior failed results.
- [ ] Ensure overrides reference the failed gate result and preserve the original failure evidence rather than overwriting history.
- [ ] Ensure override approvals are time-bounded, auditable, and bound to the exact gate identity and current policy context.

## Acceptance Criteria

- [ ] Gate outcomes are deterministic and auditable.
- [ ] Overrides are explicit, policy-controlled, and evidence-backed.
- [ ] Gate identity, lifecycle, and evidence semantics are reusable by later workflow families without inventing feature-local gate models.
- [ ] Gate ordering and execution remain plan-driven rather than runner-local or executor-local conventions.

## Executor Classification Hardening (Pre-MVP Follow-up)

- [ ] Harden system-modifying command detection used by policy hard-floor classification:
  - extend launcher/wrapper normalization beyond the current minimal set,
  - or adopt conservative full-argv classification that cannot be bypassed through wrapper indirection.
- [ ] Add deterministic regression fixtures for representative wrapper-chaining forms to ensure fail-closed behavior remains stable.
