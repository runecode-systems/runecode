# Monorepo Scaffold + Package Boundaries (v0)

User-visible outcome: the repo has a clear, security-aware monorepo layout (Go control plane + Go TUI + TS/Node scheduler) with explicit boundaries, and a consistent local build/test/lint loop.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Monorepo Layout + Package Boundaries

- Establish a monorepo layout for:
  - Go: local control plane/security kernel binaries (launcher, broker, secrets daemon, audit daemon) and Go TUI.
  - TypeScript/Node: workflow runner (LangGraph) treated as untrusted at runtime.
- Make the trust boundary explicit in code layout (no Go imports from Node packages; minimal shared artifacts only via schemas/specs).

## Task 3: Build/Test/Lint Baselines

- Go: standardize `go test ./...` and formatting checks.
- Node: pin dependencies via a lockfile and standardize `test`/`lint` scripts.
- Wire these commands into the shared developer command surface (`just`) defined in the dev environment spec.

## Task 4: Keep CI/Dev Tooling Centralized

- Defer Nix Flake / `direnv` / GitHub Actions wiring to:
  - `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/`
- This spec assumes those conventions exist and focuses on repo layout + package boundaries.

## Acceptance Criteria

- A new contributor can run the minimal build/test loop via `just` in the dev shell.
- The repo layout enforces the “untrusted scheduler” boundary by default.
