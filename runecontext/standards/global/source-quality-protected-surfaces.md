---
schema_version: 1
id: global/source-quality-protected-surfaces
title: Source-Quality Protected Surfaces
status: active
aliases:
    - agent-os/standards/global/source-quality-protected-surfaces
---

# Source-Quality Protected Surfaces

- Treat these as protected surfaces:
  - `tools/checksourcequality/**`
  - `.source-quality-baseline.json`
  - `.source-quality-config.json`
  - `.golangci.yml`
  - `runner/eslint.config.*` when present
  - `justfile`
  - `docs/source-quality.md`
  - `.github/instructions/**`
  - `.github/copilot-instructions.md`
- Why:
  - changes here can weaken enforcement
  - changes here can create reviewer-policy drift
  - changes here can bypass Tier 1 protections quietly
- Rules:
  - cover them in `CODEOWNERS`
  - review threshold and exclusion changes carefully
  - keep diffs narrow and easy to audit
