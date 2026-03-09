# Monorepo Scaffold + Package Boundaries (v0) — Shaping Notes

## Scope

Set up the initial monorepo structure and baseline build/test/lint commands, with explicit trust boundaries between trusted Go components and the untrusted TS/Node workflow runner.

This spec assumes the dev shell + CI plumbing already exists (Nix flake, `just`, GitHub Actions) and focuses on turning the repo into a real multi-language workspace (Go module + Node package + lockfiles) with a meaningful `just ci`.

## Decisions

- Go components and the TS workflow runner live in separate top-level areas with an explicit trust boundary.
- Go module lives at repo root so `go test ./...` works from repo root and `just`/CI recipes stay simple.
- Go module path is explicit and stable: `github.com/runecode-ai/runecode`.
- Node workflow runner lives in `runner/` as a standalone npm package; dependency pinning uses `package-lock.json` for MVP.
- Runner supports Node 22 and 24 (CI matrix); declare `engines.node` as `>=22.22.1 <25`.
- Go TUI uses Bubble Tea (`github.com/charmbracelet/bubbletea`) as the UI framework.
- The trust boundary is documented explicitly in `docs/trust-boundaries.md`; cross-boundary artifacts are schemas/fixtures in `protocol/` (details are owned by the Protocol & Schema Bundle v0 spec).
- This spec creates the `protocol/` directory skeleton; the protocol/schema spec owns population/versioning.
- Trust boundary is not "doc-only": add a mechanical runner boundary guardrail (`npm run boundary-check`) and run it in `just ci`.
- Boundary guardrail scope covers runner JS/TS source across `runner/` (not only `runner/src/`), skips dependency/build directories, and fails closed if no source files are found.
- Boundary guardrail path handling is cross-platform: treat Unix absolute and Windows drive-letter/UNC path references as boundary-relevant.
- Boundary guardrail avoids false positives from third-party package names that contain segments like `/internal/`; enforcement is based on repo-root references and path resolution into trusted areas.
- Boundary guardrail violation messages use normalized runner-relative forward-slash paths for deterministic cross-platform test/log output.
- The guardrail is intentionally best-effort static analysis; authoritative enforcement remains broker auth/schema validation, policy, and runtime isolation backends.
- Runner TS config is hardened for boundary safety: `rootDir: src` and `noEmit: true` for MVP typecheck/lint.
- `just ci` is check-only and must not modify committed files; lockfile generation is an explicit developer action.
- For MVP command clarity, keeping an explicit `cd runner && npm run lint` in `just ci` is acceptable even if `npm test` also invokes lint.
- CI cleanliness verification checks both tracked diffs and untracked files after running `just ci`.
- Repository line endings are normalized with `.gitattributes` to keep portability checks stable across Linux/macOS/Windows.
- Dependency vulnerability scanning is desirable but non-gating for MVP parity (expose as an optional command/CI job later).
- Developer tooling and CI conventions (Nix flake + `direnv` + `just` + GitHub Actions) are implemented first by `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/` and should not be re-invented here.

## Context

- Visuals: None.
- Product alignment: Matches the intended split (Go security kernel + Go TUI + TS LangGraph runner treated as untrusted at runtime).
- Product context: `agent-os/product/mission.md`, `agent-os/product/tech-stack.md`
- Repo tooling context: `flake.nix`, `justfile`, `.github/workflows/ci.yml`, `.github/CODEOWNERS`
- Repo hygiene context: `.gitignore` (currently minimal; must expand as languages are scaffolded)
- Related specs: `agent-os/specs/2026-03-08-1128-dev-env-ci-nix-flakes/`, `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`

## Standards Applied

- `product/roadmap-conventions` (when updating roadmap state after this spec ships)
- Portability guardrail: `just ci` must be runnable on Windows without Nix and without bash-only dependencies.
