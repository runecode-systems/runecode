# Dev Environment + CI Bootstrap (Nix Flakes)

User-visible outcome: contributors get a consistent local dev environment via `nix develop`, with `direnv` auto-entry and a `just` command surface; CI runs the same checks on Linux/macOS with Nix and preserves Windows portability via a native Windows job.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Add `flake.nix` Dev Shell (Project Standard)

- Add a `flake.nix` using `flake-utils` `eachDefaultSystem`, similar to the reference style.
- Commit `flake.lock` and treat it as the supply-chain source of truth for the dev environment.
- Add minimal Nix supply-chain guardrails (MVP):
  - explicitly scope substituters and trusted public keys (avoid implicit third-party caches)
  - ensure CI does not update `flake.lock` implicitly
- Provide:
  - `formatter = pkgs.nixfmt`
  - `devShells.default = pkgs.mkShell { ... }`
- Include baseline packages (MVP):
  - `go`, `gopls` (and common Go tooling)
  - `nodejs` (and minimal TS tooling)
  - `just`, `direnv`, `nix-direnv`
  - `git`, `jq`, `ripgrep`, `fd`, `curl`
- Add a `shellHook` that:
  - prints a short “Entering dev shell” banner
  - runs `just --list` to advertise common commands
  - exports any repo-local dev PATH overrides (e.g. `./bin/`)

## Task 3: Add `direnv` Integration

- Add a repo `.envrc` that uses the flake dev shell (`use flake`).
- Document the setup: `direnv allow`.
- Ensure the experience is “enter shell on cd into repo; exit on leave”.

## Task 4: Add `justfile` Command Surface

- Add a top-level `justfile` with a small set of stable commands:
  - `fmt`
  - `lint`
  - `test`
  - `ci` (runs the exact checks CI uses)
  - `dev` (optional convenience entrypoint)
- Keep commands cross-platform (avoid bashisms where possible).

## Task 5: GitHub Actions CI Pipeline (Nix + Windows Portability)

- Linux/macOS jobs:
  - install Nix
  - run `nix fmt` (or an equivalent flake formatter check)
  - run `nix flake check` (without writing/updating the lockfile)
  - run `nix develop -c just ci`
- Windows job:
  - run the “portability guardrail” checks without Nix (use standard Go/Node setup actions)
  - run `just ci` (or the equivalent underlying commands)

## Task 6: Short Dev Docs

- Add a brief “Dev environment” section explaining:
  - `nix develop` / `direnv` usage
  - the canonical command surface is `just`
  - how CI maps to `just ci`

## Acceptance Criteria

- On Linux/macOS, `direnv` loads the flake dev shell and `just --list` shows commands.
- CI runs on Linux/macOS (Nix-based) and Windows (native toolchains) and executes the same logical checks.
- The workflow is consistent: “if `just ci` passes locally in the dev shell, it passes in CI”.
