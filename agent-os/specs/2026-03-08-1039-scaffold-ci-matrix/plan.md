# Monorepo Scaffold + Package Boundaries (v0)

User-visible outcome: the repo has a clear, security-aware monorepo layout (Go control plane + Go TUI + TS/Node workflow runner) with explicit boundaries, and a consistent local build/test/lint loop via `just` that also runs in CI.

This spec is intentionally "scaffold-first": the repo currently has CI/dev-shell plumbing but no Go/Node packages yet. The tasks below make the language workspaces real (lockfiles included) and turn placeholder `just` commands into meaningful checks.

## Task 1: Save Spec Documentation

Ensure `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/` contains:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty unless visuals are added later)

## Task 2: Define Repo Layout + Trust Boundary

- Adopt a top-level layout that makes the trusted/untrusted split obvious and hard to accidentally bypass.
- Proposed skeleton (MVP; exact names can be adjusted but the boundary must remain clear):

```text
.
├── cmd/                      # Trusted Go binaries (control plane + TUI)
│   ├── runecode-launcher/
│   ├── runecode-broker/
│   ├── runecode-secretsd/
│   ├── runecode-auditd/
│   └── runecode-tui/
├── internal/                 # Trusted Go libraries (not imported cross-boundary)
├── runner/                   # Untrusted TS/Node workflow runner package
│   └── src/
├── protocol/                 # Cross-boundary schemas + fixtures (source of truth)
│   ├── schemas/
│   └── fixtures/
└── docs/
    └── trust-boundaries.md
```

- Boundary rules (MVP):
  - Go control plane + TUI are "trusted"; the Node runner is "untrusted" at runtime.
  - Cross-boundary communication goes through the broker/local API using schema-validated messages (no ad-hoc JSON).
  - Shared artifacts across the boundary are limited to:
    - schemas/fixtures in `protocol/` (source of truth), and
    - generated validators/types in each language (generated outputs live with their language code).
  - Avoid "easy bypass" paths (examples: Node reading host secrets directly; Node reaching into trusted internal state via filesystem mounts).

- Ownership clarification (avoid spec chicken-and-egg):
  - This scaffold spec creates the `protocol/` directory skeleton (`protocol/schemas/` and `protocol/fixtures/`).
  - The schema content, canonicalization fixtures, and versioning rules are owned by `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`.

- Scaffold-level boundary guardrails (mechanical, not just documentation):
  - Add a runner-side boundary check (see Task 3) that fails CI if the TS runner imports from or references trusted Go areas (`cmd/`, `internal/`) or otherwise escapes its package boundary (except allowed reads of `protocol/`).
  - Rationale: future specs rely on the runner being untrusted by default (policy/broker/launcher are enforcement points). This guardrail prevents accidental coupling and "escape hatches" while the real runtime isolation is still being built.

Deliverables:
- `docs/trust-boundaries.md` documenting:
  - the boundary in one diagram or table,
  - allowed interfaces/artifacts,
  - explicit "must never happen" bypasses, and
  - how we keep it enforced by default.
  - an "Enforcement" section that names the real enforcement points owned by follow-on specs:
    - broker local API auth + schema validation (`agent-os/specs/2026-03-08-1039-broker-local-api-v0/`)
    - deterministic policy decisions (`agent-os/specs/2026-03-08-1039-policy-engine-v0/`)
    - runtime isolation backends with no host filesystem mounts (`agent-os/specs/2026-03-08-1039-launcher-microvm-backend-v0/`, `agent-os/specs/2026-03-08-1039-container-backend-opt-in-v0/`)

## Task 3: Scaffold Language Workspaces (Go + Node)

- Go (trusted components):
  - Create a Go module at repo root so `go test ./...` works from the repo root.
    - Module path (required): `github.com/runecode-ai/runecode`
    - Commit `go.mod` and `go.sum`.
  - Add minimal stub binaries in `cmd/` (one `main.go` per binary) so builds/tests are non-empty:
    - `cmd/runecode-launcher/`
    - `cmd/runecode-broker/`
    - `cmd/runecode-secretsd/`
    - `cmd/runecode-auditd/`
    - `cmd/runecode-tui/`
  - Stub safety posture (MVP):
    - Stubs must not listen on TCP/UDP ports by default.
    - Stubs must not accept secrets via CLI args or environment variables (stdin/file-descriptor only, when implemented later).
    - Prefer "help + safe exit" behavior until the owning specs implement real daemons.
  - TUI framework choice (MVP): implement `cmd/runecode-tui/` using Bubble Tea (`github.com/charmbracelet/bubbletea`).
    - Keep the initial program minimal (one screen) but "real" (model/update/view loop, key handling, graceful exit).
    - Avoid OS-specific terminal assumptions; it must compile and run on Linux/macOS/Windows in CI.
  - Add at least one small package and test under `internal/` so `go test ./...` exercises something.

- Node runner (untrusted package):
  - Create `runner/package.json` and pin dependencies via `runner/package-lock.json` (use `npm` for MVP).
    - Declare supported Node versions via `engines` (CI exercises Node 22 and 24):
      - `node: >=22.22.1 <25`
  - Add `runner/tsconfig.json` and a minimal `runner/src/index.ts` entrypoint.
    - `tsconfig.json` should be check-first for MVP: `noEmit: true`.
    - Set `rootDir: "src"` to make out-of-package imports fail fast.
  - Provide baseline scripts that work on Windows CI:
    - `npm run lint` (may start as `tsc --noEmit` until eslint is introduced)
    - `npm test` (must fail on type errors; can initially be equivalent to typecheck)

  - Boundary guardrail (required): add `npm run boundary-check` that scans runner JS/TS source files across `runner/` (not only `runner/src/`; excluding dependency/build directories and boundary-check tooling/test files) and fails if:
    - import from `../../internal/*` or `../../cmd/*` (or any other path that escapes `runner/`),
    - read or reference trusted code paths except for allowed reads of `protocol/` schemas/fixtures.
    - absolute path references (including Unix absolute paths and Windows drive-letter/UNC paths) escape `runner/` except for allowed protocol reads.
    - no runner source files are found (fail closed).
    - NOTE: the guardrail should avoid false positives on unrelated package names that merely contain path segments like `/internal/`.
    - Violation output should use runner-relative paths with forward slashes so test expectations are stable across Linux/macOS/Windows.
    Implementation guidance (MVP): keep it dependency-free and cross-platform (Node stdlib only).
  - Add baseline tests for the boundary-check guardrail and run them from `npm test`.

Deliverables:
- `go.mod` (module `github.com/runecode-ai/runecode`) and `go.sum` at repo root.
- `runner/package.json` and `runner/package-lock.json`.
- `runner/scripts/boundary-check.js` plus baseline guardrail tests.
- Minimal Go/TS entrypoints so `just` targets can run real checks.

## Task 4: Make `just` Real (fmt/lint/test/ci)

- Update `justfile` so the stable command names from the dev-env spec become meaningful:
  - `just fmt` applies formatting.
  - `just lint` runs check-only analysis (no mutation).
  - `just test` runs unit tests.
  - `just ci` runs the same checks as CI and is check-only; it must leave a clean `git diff`.

- Define concrete MVP semantics (cross-platform; avoid shell pipelines that break on Windows):
  - Go:
    - `fmt`: gofmt Go sources.
    - `lint`: `go vet ./...` + a gofmt check (fail if formatting would change).
    - `test`: `go test ./...`
    - `ci`: `go test ./...` + `go vet ./...` + `go build ./cmd/...` + gofmt check.
  - Node (`runner/`):
    - `lint`: `cd runner && npm run lint` (typecheck is acceptable for MVP).
    - `test`: `cd runner && npm test`
    - `ci`: `cd runner && npm ci` + `npm run lint` + `npm test` + `npm run boundary-check`
    - For MVP, `npm test` may include `npm run lint`; keeping an explicit lint step in `just ci` is acceptable for clear sequencing.

  - Bootstrap vs check-only contract:
  - Lockfile generation/updates (`go mod tidy`, `npm install`) are explicit developer actions; the resulting lockfile changes are committed.
  - `just ci` must never run dependency-updating commands and must not modify committed files.

- Implementation guidance (cross-platform):
  - If a check requires inspecting output (e.g., "fail if gofmt would change files"), prefer a small helper program/script over shell piping so Windows CI remains first-class.

- Optional (non-gating) hardening command surface:
  - Add a `just vuln` (or similar) target that runs best-effort dependency vulnerability scans (e.g., `govulncheck`, `npm audit`).
  - Keep it out of `just ci` for MVP so offline/local parity remains strong; consider making it a separate CI job later.

- Cross-platform requirements (Windows job runs this without Nix):
  - Avoid bashisms and unix-only tools in recipes (no `find`, `xargs`, process substitution, etc.).
  - Use explicit `cd runner` where needed and only invoke tools that exist in the dev shell + Windows CI toolchain.

## Task 5: CI Alignment (Keep Centralized)

- CI and dev-shell wiring already exists (Nix + Windows portability) and should remain centralized.
- Keep CI changes minimal and mechanical:
  - CI continues to call `just ci`.
  - If `just ci` grows new prerequisites, update `flake.nix` dev shell packages and the Windows job tool installs to match.
  - Preserve existing supply-chain guardrails (pinned actions, explicit Nix cache allowlist, and lockfile immutability).

- Version alignment guidance (avoid "works on my machine" drift as future specs land):
  - Go: keep Go version compatibility aligned with CI (Windows pins `go-version: 1.25.7`).
  - Node: keep the runner compatible with Node 22 and 24 (CI matrix); Nix dev shell may use Node 24 only.

## Task 6: Repo Hygiene + Ownership Guardrails

- Update `.gitignore` for new language/tooling artifacts (minimum):
  - `node_modules/`
  - `runner/dist/`
  - `*.tsbuildinfo`
  - common editor/OS junk (if not already covered elsewhere)

- Extend `.github/CODEOWNERS` coverage for security-critical areas introduced by this scaffold:
  - `/protocol/`
  - `/docs/trust-boundaries.md`
  - `/cmd/runecode-secretsd/`
  - `/cmd/runecode-auditd/`
  (Owner policy: match the existing "high-leverage" owner/team until a dedicated security team exists.)

- Add a minimal `.editorconfig` to reduce cross-editor churn across Go/TS/Markdown/JSON.

## Acceptance Criteria

- Linux/macOS: `nix develop -c just ci` passes and leaves a clean `git diff`.
- Windows CI: `just ci` runs successfully (no Nix, no bash dependency).
- Go: `go test ./...` runs from repo root (Go module present).
- Node: `cd runner && npm ci && npm test` runs (lockfile present).
- `docs/trust-boundaries.md` exists and documents allowed cross-boundary interfaces, prohibited bypasses, and the named enforcement points.
- Runner boundary guardrail is active: `cd runner && npm run boundary-check` fails on boundary escapes.
- `.gitignore`, `.github/CODEOWNERS`, and `.editorconfig` reflect the new repo surface area.
- `protocol/` skeleton exists (`protocol/schemas/` and `protocol/fixtures/`).
