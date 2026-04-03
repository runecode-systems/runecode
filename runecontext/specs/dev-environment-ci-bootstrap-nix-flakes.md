---
schema_version: 1
id: dev-environment-ci-bootstrap-nix-flakes
title: Dev Environment + CI Bootstrap (Nix Flakes)
originating_changes:
  - CHG-2026-001-57d6-agent-os-to-runecontext-migration-umbrella
revised_by_changes: []
---

# Dev Environment + CI Bootstrap (Nix Flakes)

## Summary

RuneCode uses a Nix-flake-based developer environment with a stable `just` command surface and CI parity built around `just ci`.

## Durable Current-State Outcomes

- `flake.nix` and `flake.lock` define the canonical local dev environment.
- `.envrc` supports `direnv` + `nix-direnv` auto-entry for the dev shell.
- `justfile` provides stable `fmt`, `lint`, `test`, and `ci` entrypoints.
- CI uses Nix-based checks on Linux/macOS and native toolchains on Windows while preserving the same logical check contract.
- CI and local workflows are check-only by default and should not silently mutate lockfiles or tracked files.

## Security And Supply-Chain Posture

- `flake.lock` is committed and treated as the reproducibility source of truth for the dev environment.
- CI uses pinned actions, explicit Nix cache allowlisting, and lockfile immutability checks.
- High-leverage workflow and environment files are expected to remain under explicit review ownership.

## Operator Guidance

- Canonical local parity command: `nix develop -c just ci`.
- Windows/non-Nix parity command: `just ci` with required toolchain versions installed.

## Related Standards

- `runecontext/standards/ci/just-ci.md`
- `runecontext/standards/ci/worktree-cleanliness.md`
- `runecontext/standards/ci/github-actions-supply-chain.md`
- `runecontext/standards/global/deterministic-check-write-tools.md`
