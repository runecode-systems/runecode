# Design

## Overview
Implement deterministic gate execution with explicit, auditable evidence outputs and fail-closed semantics.

## Key Decisions
- Gates are deterministic and produce typed evidence artifacts.
- Gate failures fail the run by default.
- Any override requires explicit approval and audit events.
- Gate overrides should be modeled as canonical policy actions with typed approval payloads and shared reason-code semantics rather than as feature-local override exceptions.
- Gates should be first-class typed workflow checks with stable identity and version rather than loose human-facing names or implicit shell steps.
- Gate execution should use declared and normalized inputs rather than ambient process state or ad hoc directory scraping.
- Retry and rerun semantics should mint separate gate-attempt identities instead of mutating prior gate evidence or silently overwriting old results.
- Gate evidence should be small, typed, reference-heavy, and content-addressed rather than relying on unstructured logs as the trust root.

## Gate Contract

- Every gate should have a stable contract including at least:
  - `gate_id`
  - `gate_kind`
  - `gate_version`
  - declared normalized input references and digests
  - deterministic execution contract
  - evidence schema
  - retry semantics
  - override policy semantics
- Gate ordering should be part of the workflow/process definition or compiled execution plan rather than implicit step-local convention.
- Gates should execute at explicit checkpoints and consume workspace-role outputs through declared artifact or digest inputs rather than ambient mutable local state wherever practical.

## Gate Lifecycle

- Gate lifecycle should be explicit and reusable across workflows:
  - `planned`
  - `running`
  - `passed`
  - `failed`
  - `overridden`
  - `superseded`
- A retry should create a new gate attempt.
- Earlier failed or superseded gate attempts should remain durably inspectable and referenced by audit/evidence surfaces rather than being overwritten.

## Gate Evidence Model

- Gate evidence should be represented by a dedicated typed evidence object rather than only generic logs.
- The typed evidence object should include at least:
  - `gate_id`
  - `gate_kind`
  - `gate_version`
  - `run_id`
  - `stage_id`
  - `step_id`
  - `role_instance_id`
  - `gate_attempt_id`
  - `started_at`
  - `finished_at`
  - normalized input digests / refs
  - tool/runtime identity and version
  - deterministic outcome
  - referenced output/log digests
  - failure reason code when applicable
  - related policy decision refs and override linkage when applicable
- Bulky stdout/stderr or other large outputs should remain separate referenced artifacts; the evidence object should be the canonical typed summary and binding layer.
- Gate evidence artifacts should use a dedicated data class so policy, retention, and audit linkage can reason about them explicitly.

## Override Model

- Gate overrides remain canonical policy actions, not local gate exceptions.
- Override requests should bind the specific gate identity, gate attempt or failed result identity as needed, and the current effective policy context.
- Override consumption should remain explicit, auditable, and time-bounded.
- An override should not mutate the original failed gate result; it should add a policy-approved continuation path linked to that failed result.

## Main Workstreams
- Gate framework and execution order.
- Evidence artifact schema and retention linkage.
- Retry and override policy integration.
- Gate identity, lifecycle, and attempt model.
