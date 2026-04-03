---
schema_version: 1
id: global/language-aware-source-docs
title: Language-Aware Source Docs
status: active
suggested_context_bundles:
    - go-control-plane
aliases:
    - agent-os/standards/global/language-aware-source-docs
---

# Language-Aware Source Docs

- Do not use one doc or comment style everywhere.

Go:
- prefer package comments and exported declaration comments
- do not require top-of-file docs on every large file

JS/TS:
- require top-of-file module docs for Tier 1, trust-boundary, or policy-sensitive modules
- do not require blanket JSDoc on helpers

Comments:
- explain why, invariants, trust-boundary assumptions, or non-obvious tradeoffs
- do not restate syntax
- prefer maintained docs, specs, or ADRs when rationale spans multiple files or components
