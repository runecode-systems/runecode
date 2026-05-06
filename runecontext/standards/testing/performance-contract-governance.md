---
schema_version: 1
id: testing/performance-contract-governance
title: Performance Contract Governance
status: active
suggested_context_bundles:
    - ci-tooling
---

# Performance Contract Governance

Use `tools/perfcontracts/manifest.json` as the authoritative inventory for checked-in performance contracts and reviewed baselines.

- Keep performance contracts separate from `runecontext/assurance/baseline.yaml`
- Keep CI check-only: verification must never auto-rewrite performance baselines
- Require explicit `threshold_origin` per threshold: `product_budget | investigation_baseline | first_calibration | temporary_guardrail`
- Require explicit timing boundaries (`start_event`, `end_event`, `clock_source`, `evidence_source`, `included_phases`) for every metric
- Treat the manifest baseline entry for each metric as authoritative: required `regression-budget` and `hybrid-budget` metrics must point `baseline_ref` at the exact path registered in `tools/perfcontracts/manifest.json`
- Reject duplicate `metric_id` entries in manifest baselines; provenance must never depend on last-write-wins manifest ordering
- Treat required enforcement as the intersection of reviewed `lane_authority` and `activation_state: required`; defined, informational, and `contract_pending_dependency` metrics stay outside required numeric enforcement
- Keep the shared-Linux required lane truthful: it enforces only the current checked-in `required_shared_linux` subset, while broader surfaces may remain informational or `contract_pending_dependency`
- Keep perf-tool diagnostics sanitized: do not leak sensitive local paths, tokens, or raw startup output in check failures
- Keep measurement boundaries honest: validate fixture or path preconditions before timing, measure fresh-process startup or attach when startup cost is in scope, and preserve the authoritative timing source when a script or tool emits the measurement directly
- Treat baseline refresh as explicit reviewed change; do not hide threshold loosening in silent baseline updates
- Keep broader fixture ladders and cross-platform expansion in `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`
