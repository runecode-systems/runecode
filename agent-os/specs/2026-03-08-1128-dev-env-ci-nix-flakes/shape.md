# Dev Environment + CI Bootstrap (Nix Flakes) — Shaping Notes

## Scope

Standardize local development and CI via a Nix Flake dev shell, automatic `direnv` entry/exit, and a `just` command surface.

## Decisions

- Use Nix Flakes for the canonical local dev environment.
- Use `direnv` + `nix-direnv` so entering/leaving the repo automatically enters/exits the dev shell.
- Use `just` for stable, memorable developer commands across a multi-component repo (Go + TS/Node). This is useful here because it keeps commands consistent across OSes and reduces “tribal knowledge” about which subproject command to run.
- CI uses Nix on Linux/macOS; Windows CI remains native (portability guardrail) because Nix is not a first-class Windows runtime.
- Treat `direnv` and `nix-direnv` as host prerequisites. Keep `.envrc` intentionally thin (`use flake` only).
- Supply chain posture for tooling is lockfile-driven: `flake.lock` is committed, CI uses `--no-write-lock-file`, and the job fails if `flake.lock` changes.
- Nix binary caches are allowlisted (MVP: `https://cache.nixos.org` only). Any additional cache requires explicit substituter/key pinning.
- `nix flake check` must be meaningful (at minimum: dev shell build + Nix formatting check) to avoid “green but empty” CI.
- High-leverage files (`flake.nix`, `flake.lock`, `.envrc`, `justfile`, `.github/workflows/*`) are protected via CODEOWNERS.

## Context

- Visuals: None.
- References: external flake style reference (see `references.md`).
- Product alignment: Improves contributor ergonomics without weakening the security model.

## Standards Applied

- Spec-specific standards captured in `standards.md` (supply-chain hardening, execution safety, and portability constraints).
