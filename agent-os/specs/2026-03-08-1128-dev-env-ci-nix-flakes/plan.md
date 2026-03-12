# Dev Environment + CI Bootstrap (Nix Flakes)

User-visible outcome: contributors get a consistent local dev environment via `nix develop`, with `direnv` auto-entry and a `just` command surface; CI runs the same logical checks on Linux/macOS with Nix, and preserves Windows portability via a native Windows job. Supply-chain posture is explicit: `flake.lock` is the source of truth, CI does not mutate it, and Nix binary caches are allowlisted.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: Add `flake.nix` Dev Shell (Project Standard)

- Add a `flake.nix` using `flake-utils` `eachDefaultSystem`, similar to the reference style.
- Minimum supported Nix for contributors using the dev shell: Nix >= 2.18.
- Commit `flake.lock` and treat it as the supply-chain source of truth for the dev environment.
- Supply-chain guardrails (MVP):
  - CI must run Nix with an explicit substituter + trusted-public-keys allowlist (start with `https://cache.nixos.org` only; avoid implicit third-party caches). MVP values:
    - `substituters = https://cache.nixos.org`
    - `trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=`
  - CI must use `--no-write-lock-file` (or equivalent) and must fail if `flake.lock` changes during the run.
  - Protect high-leverage files with `CODEOWNERS`: `flake.nix`, `flake.lock`, `.envrc`, `justfile`, `.github/CODEOWNERS`, and `.github/workflows/*`.
- Provide flake outputs:
  - `formatter = pkgs.nixfmt-rfc-style` (or the chosen formatter)
  - `devShells.default = pkgs.mkShell { ... }`
  - `checks` so `nix flake check` is meaningful (at minimum: build the dev shell closure + a Nix formatting check)
- Include baseline packages (MVP):
  - `go`, `gopls` (and common Go tooling)
  - `nodejs` (and minimal TS tooling)
  - `just`
  - `git`, `jq`, `ripgrep`, `fd`, `curl`
- Add a minimal `shellHook` that:
  - prints a short “Entering dev shell” banner (and optionally tool versions)
  - runs `just --list` to advertise common commands
  - prints banner/list output only in interactive shells (avoid noisy `nix develop -c ...` output)
  - does not make network calls and does not mutate files outside the repo
  - avoids prepending repo-writable directories to `PATH` unless they are intentionally committed (prefer explicit `./scripts/` usage over implicit `./bin/` shadowing)

Parallelization: touches high-leverage files (`flake.nix`, `flake.lock`); avoid parallel edits with other tasks that modify Nix/CI plumbing.

## Task 3: Add `direnv` Integration

- Treat `direnv` and `nix-direnv` as host prerequisites (they are required before the flake can auto-load).
- Add a repo `.envrc` that is as thin as possible and uses the flake dev shell (`use flake`).
- Document the setup: install `direnv` + `nix-direnv`, set up shell hook, then `direnv allow`.
- Update `.gitignore` to ignore `direnv` and common Nix build artifacts (at minimum: `.direnv/`, `result`, `result-*`).
- Document rollback/unblock steps: `direnv deny` to stop auto-loading; `nix develop` still works as a manual fallback.
- Ensure the experience is “enter shell on cd into repo; exit on leave”.

Parallelization: can be implemented in parallel with Task 4, but both may touch `.gitignore` and repo root docs.

## Task 4: Add `justfile` Command Surface

- Add a top-level `justfile` with a small set of stable commands:
  - `fmt`
  - `lint`
  - `test`
  - `ci` (runs the exact checks CI uses)
  - `dev` (optional convenience entrypoint)
- Define the MVP semantics explicitly so this spec can land before language packages exist:
  - `ci` is at minimum a smoke check that verifies the dev shell contains the expected baseline tools and prints versions.
  - `fmt`/`lint`/`test` may be placeholders initially, but the names and intent are stable and will be extended by follow-on specs.
- Keep commands cross-platform (avoid bashisms where possible) so Windows CI can run `just ci` without Nix.

Parallelization: can be implemented in parallel with CI workflow wiring, but avoid conflicts by agreeing on the canonical `just ci` contract early.

## Task 5: GitHub Actions CI Pipeline (Nix + Windows Portability)

- Linux/macOS jobs:
  - set workflow token permissions to least privilege (`permissions: contents: read`, unless broader scopes are required)
  - pin all workflow actions to full commit SHA (checkout/setup/Nix install/cache)
  - install Nix (installer action must be SHA-pinned)
  - add Nix cache acceleration via a SHA-pinned action (recommended)
  - configure Nix with an explicit substituter/key allowlist (do not rely only on flake `nixConfig.extra-*`). Example via `NIX_CONFIG` or `--option`:
    - `substituters = https://cache.nixos.org`
    - `trusted-public-keys = cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=`
  - validate `flake.lock` metadata before executing flake checks (`nix flake lock --no-update-lock-file`)
  - run `nix flake check --no-write-lock-file`
  - run `nix develop --no-write-lock-file -c just ci`
  - fail the job if `flake.lock` is modified by any step
  - log Nix + key tool versions early for debuggability
  - if additional caches are added for speed (e.g., Cachix), enumerating substituters/keys is required (no implicit caches)
- Windows job:
  - run the “portability guardrail” checks without Nix (use standard Go/Node setup actions)
  - pin installed tool versions (avoid `@latest` for CI-critical tools)
  - run `just ci` (or the equivalent underlying commands)

Parallelization: touches high-leverage workflow files; avoid parallel edits with other CI-related specs to reduce merge conflicts.

## Task 6: Short Dev Docs

- Add a brief “Dev environment” section explaining:
  - prerequisites (Nix; and for auto-entry: `direnv` + `nix-direnv`)
  - `nix develop` / `direnv` usage
  - the canonical command surface is `just`
  - how CI maps to `just ci`
  - `direnv`/flake trust model (why `flake.nix`, `flake.lock`, and `.envrc` changes are reviewed carefully)
  - how to get unstuck if the flake breaks (`direnv deny`, manual `nix develop`, clearing `.direnv/`)

Parallelization: can be implemented in parallel with the scaffold spec docs; coordinate on shared wording for CI/justfile conventions.

## Acceptance Criteria

- On Linux/macOS, `direnv` loads the flake dev shell and `just --list` shows commands.
- CI runs on Linux/macOS (Nix-based) and Windows (native toolchains) and executes the same logical checks.
- The workflow is consistent: “if `just ci` passes locally in the dev shell, it passes in CI”.
- CI uses `--no-write-lock-file`, validates `flake.lock` before flake execution, and fails if `flake.lock` changes.
- CI workflows set least-privilege permissions and use full-SHA-pinned actions.
- High-leverage files (`flake.nix`, `flake.lock`, `.envrc`, `justfile`, `.github/CODEOWNERS`, `.github/workflows/*`) are protected by CODEOWNERS.
