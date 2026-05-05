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
- Treat baseline refresh as explicit reviewed change; do not hide threshold loosening in silent baseline updates
- Keep broader fixture ladders and cross-platform expansion in `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`
