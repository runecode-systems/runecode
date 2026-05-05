# Performance Contracts

This directory under `tools/perfcontracts/` is the reviewed, machine-readable contract surface for performance verification.

It is intentionally outside `runecontext/` and separate from `runecontext/assurance/baseline.yaml`.

## Artifact Format

- `manifest.json` is the authoritative inventory for checked-in performance contracts and optional repeated-sample baselines.
- `fixtures/*.json` contains reviewed fixture inventory and stable fixture IDs.
- `contracts/*.json` contains per-surface metric contracts.
- `baselines/*.json` contains optional repeated-sample baseline artifacts for regression and hybrid budgets.

Each metric contract declares:

- metric identity and fixture identity
- measurement kind and unit
- budget class (`exact`, `absolute-budget`, `regression-budget`, `hybrid-budget`)
- lane authority and activation state
- threshold origin
- timing boundary (`start_event`, `end_event`, `clock_source`, `evidence_source`, `included_phases`)

## Baseline Governance

### Threshold review process

Threshold changes are contract changes, not harness-only edits.

- Tightening a threshold requires explicit rationale, evidence, and expected operator or product impact.
- Deliberate threshold loosening (accepted regression) requires explicit justification in review notes and must explain why the regression is acceptable now.
- Every threshold keeps a reviewed `threshold_origin` (`product_budget`, `investigation_baseline`, `first_calibration`, `temporary_guardrail`) so provenance remains inspectable.
- `threshold_origin` values are validated by `internal/perfcontracts` and must not be free-form.

### Baseline refresh policy for major architecture shifts

When a major reviewed architectural shift lands, refresh baselines with an explicit review path:

1. Keep metric identity stable (`metric_id`, fixture, timing boundary, budget class) unless semantics truly changed.
2. If semantics changed, add a new metric or fixture identity rather than silently reusing old identities.
3. Collect repeated samples using the reviewed defaults below in the authoritative environment.
4. Commit refreshed baseline artifacts and any threshold changes together with rationale.
5. Keep normal CI check-only; baseline refresh is intentional and reviewed, never auto-mutated.

## Statistical defaults (reviewed v1)

These defaults are the initial contract constants for CHG-053 and are tuned only through explicit follow-up review.

- **Microbenchmarks**
  - repeated samples: `10` for required PR comparisons
  - repeated samples: `20` preferred for baseline refresh or recalibration
  - comparison: robust repeated-sample regression check with practical noise-floor gate
- **Latency metrics**
  - trials: `30` when `p95` is authoritative
  - p95 eligibility: require fixed repeated trials sufficient for meaningful p95; otherwise use median+max while informational
  - comparison: explicit reviewed ceilings (`p95` or median+max per metric contract)
- **CPU/process-behavior metrics**
  - warmup window: `3000ms`
  - observation window: `20000ms`
  - repeated windows: `3`
  - comparison: sustained average/median signal plus max guardrail
- **Exact metrics**
  - comparison: exact value or hard bound only (no inferential statistics)

### Practical noise-floor policy

Regression checks that use repeated-sample comparisons must require both:

- regression threshold exceeded (for example, `max_regression_percent`)
- practical noise floor exceeded (`practical_noise_floor`)

This avoids gate churn from statistically detectable but operationally irrelevant movement.

## CI Contract

The trusted compare/enforce tool is `go run ./tools/perfcontracts`.

Normal CI runs are check-only:

- contract validation and compare are read-only
- no baseline rewrite behavior is allowed in verification flows

## Scope

This initial inventory intentionally covers one reviewed MVP fixture set per major surface.
Broader fixture ladders and cross-platform expansion are deferred to:

- `CHG-2026-061-45fe-performance-program-expansion-cross-platform-gates-v0`

Deferred broader performance surfaces include larger fixture ladders, wider workflow-pack and git-gateway coverage, and tuned macOS/Windows numeric-gate programs.
