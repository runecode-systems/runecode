---
schema_version: 1
id: global/source-quality-enforcement-layering
title: Source-Quality Enforcement Layering
status: active
suggested_context_bundles:
    - go-control-plane
aliases:
    - agent-os/standards/global/source-quality-enforcement-layering
---

# Source-Quality Enforcement Layering

- Use layered enforcement:
  - `golangci-lint` is the broad Go floor
  - `tools/checksourcequality` enforces stricter repo-specific and path-tiered rules
  - `just lint` and `just ci` run both
- Keep rules in `golangci-lint` when they are:
  - language-native
  - deterministic repo-wide
  - easy to express without path-policy drift
- Keep rules in `checksourcequality` when they are:
  - cross-language
  - Tier 1 vs Tier 2 specific
  - tied to reviewed baseline/config files
  - about protected surfaces or repo-specific policy
- Do not duplicate policy logic in both layers unless the overlap is intentional and documented.
