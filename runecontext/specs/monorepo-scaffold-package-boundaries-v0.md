---
schema_version: 1
id: monorepo-scaffold-package-boundaries-v0
title: Monorepo Scaffold + Package Boundaries (v0)
originating_changes:
  - CHG-2026-001-57d6-agent-os-to-runecontext-migration-umbrella
revised_by_changes: []
---

# Monorepo Scaffold + Package Boundaries (v0)

## Summary

RuneCode establishes a security-aware monorepo layout with explicit trusted/untrusted boundaries and a cross-platform `just ci` contract.

## Durable Current-State Outcomes

- Trusted Go components are organized under `cmd/` and `internal/`.
- Untrusted workflow execution is isolated under `runner/`.
- Cross-boundary contract artifacts are centralized under `protocol/`.
- Trust-boundary rules and prohibited bypasses are documented in `docs/trust-boundaries.md`.
- Runner boundary guardrails are mechanically enforced via `runner` boundary-check tooling and CI.
- Root Go module and runner package scaffolding provide consistent build/test/lint entrypoints.

## Boundary And Portability Invariants

- The runner must not import or reference trusted `cmd/` or `internal/` paths.
- Shared trust-boundary artifacts are limited to approved protocol schema/fixture surfaces.
- `just ci` remains the canonical check gate and must be runnable in Windows CI without Nix-only assumptions.

## Related Standards

- `runecontext/standards/security/trust-boundary-interfaces.md`
- `runecontext/standards/security/trust-boundary-layered-enforcement.md`
- `runecontext/standards/security/runner-boundary-check.md`
- `runecontext/standards/ci/windows-portability-matrix.md`
- `runecontext/standards/ci/worktree-cleanliness.md`
