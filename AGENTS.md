# AGENTS.md

Guidance for agentic coding agents working in this repository.

## Non-negotiable rules
- DO NOT edit `agent-os/doc-dump/project-idea.md` (historical-only document).
- Preserve trust boundaries: `runner/` is untrusted; `cmd/` and `internal/` are trusted.
- Never add runner imports/references into trusted paths (`cmd/`, `internal/`).
- Runner must not import/reference repo-root `tools/` or other cross-boundary roots outside `protocol/`.
- Runner cross-boundary file access is only allowed to `protocol/schemas/` and `protocol/fixtures/`.
- Do not add ad-hoc cross-boundary message formats outside `protocol/schemas/`.
- Keep checks deterministic and CI/local parity centered on `just ci`.
- Never leak secrets/tokens/sensitive local paths in logs, errors, fixtures, tests, or docs.

## Repository map
- `cmd/` - trusted Go binaries
- `internal/` - trusted Go packages/helpers
- `runner/` - untrusted Node/TypeScript package
- `protocol/` - cross-boundary schemas and fixtures
- `tools/` - repo-local helper tools
- `agent-os/` - specs, standards, roadmap/product docs
- `docs/trust-boundaries.md` - boundary contract and prohibited bypasses
- `docs/source-quality.md` - source-quality policy and enforcement expectations

## Toolchain baseline
- Go: `1.25.x` (`go.mod`)
- Node: `>=22.22.1 <25` (`runner/package.json`)
- npm engine enforcement: `runner/.npmrc` has `engine-strict=true`
- Canonical shell entrypoint: `nix develop -c just ci`
- Non-Nix fallback is acceptable if versions and commands match CI behavior.

## Build, lint, and test commands
Run from repo root unless noted.

- Show recipes: `just --list`
- Format (writes): `just fmt`
- Lint: `just lint`
- Test: `just test`
- CI parity gate: `just ci`

`just fmt` runs:
- `go run ./tools/gofmtcheck --write`

`just lint` runs:
- `go run ./tools/gofmtcheck`
- `go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run`
- `go vet ./...`
- `go run ./tools/checksourcequality`
- `cd runner && npm run lint`
- `cd runner && npm run boundary-check`

`just test` runs:
- `go test ./...`
- `cd runner && npm test`

`just ci` runs:
- gofmt check, pinned `golangci-lint`, vet, source-quality checks, Go tests, and `go build ./cmd/...`
- `cd runner && npm ci && npm run lint && npm test && npm run boundary-check`

## Single-test recipes (important)
Go:
- One package: `go test ./internal/scaffold`
- One test: `go test ./internal/scaffold -run '^TestWriteHelp$'`
- Verbose one test: `go test ./internal/scaffold -run '^TestValidateArgs$' -v`

Runner tests (Node built-in runner):
- Full runner test file: `cd runner && node --test scripts/boundary-check.test.js`
- One test by name:
  - `cd runner && node --test scripts/boundary-check.test.js --test-name-pattern "rejects Unix absolute path references"`
- Boundary-only guardrail:
  - `cd runner && npm run boundary-check`

Notes:
- `cd runner && npm test` runs lint first, then tests.
- For tight loops, prefer direct `node --test ... --test-name-pattern ...`.

## Code style: global formatting
- Follow `.editorconfig` and `.gitattributes` exactly.
- LF endings, UTF-8, final newline, no trailing whitespace (Markdown exempt).
- Go files use tabs.
- TS/JS/JSON/MD/YAML use 2 spaces.
- Do not hand-format Go; use `go run ./tools/gofmtcheck --write`.

## Code style: Go (`cmd/`, `internal/`, `tools/`)
- Keep `cmd/<bin>/main.go` thin and wiring-only; place logic in `internal/...`.
- Imports: stdlib block first, blank line, then external/module imports.
- Reuse scaffold helpers when applicable:
  - `internal/scaffold.ValidateArgs`
  - `internal/scaffold.WriteHelp`
  - `internal/scaffold.WriteStubMessage`
- Exit-code contract for CLIs:
  - help => stdout, exit `0`
  - usage/arg errors => stderr, exit `2`
  - runtime/internal errors => stderr, exit `1`
- Avoid panics for expected/user input errors; return and handle errors explicitly.
- Keep stub behavior side-effect free (no listeners, no filesystem writes).

Go tests:
- Prefer table-driven tests for input matrices.
- Use `t.Run(...)` with descriptive names.
- Test both success and failure paths.
- Use explicit assertion messages (`t.Fatalf`, `t.Fatal`).

## Code style: Runner (`runner/`)
- TypeScript lint is typecheck-only: `tsc --noEmit`.
- Keep `runner/tsconfig.json` invariants:
  - `strict: true`
  - `noEmit: true`
  - `rootDir: "src"`
  - `module` and `moduleResolution`: `NodeNext`
- Prefer Node built-ins and low-dependency scripts/tests.
- Prefer Node's native test runner (`node --test`), not Jest/Vitest.
- Keep `npm test` as the combined gate (lint + test).

JS/TS naming/import conventions:
- Use `node:`-prefixed builtin imports (`node:fs`, `node:path`, etc.).
- Use camelCase for functions/locals.
- Use UPPER_SNAKE_CASE for module-level constants.
- Keep utilities deterministic and portable.
- In runner source, avoid absolute paths except explicitly allowed protocol paths.

## Trust-boundary change checklist
When touching `runner/`, `protocol/`, or boundary docs/scripts:
- Confirm no new runner references to `cmd/`, `internal/`, `tools/`, or other repo-root paths outside allowed `protocol/` roots.
- Confirm only `protocol/schemas/` and `protocol/fixtures/` remain allowed cross-boundary roots.
- Preserve fail-closed behavior in boundary checks:
  - no source files found => fail
  - any violation => fail
- Preserve Unix + Windows path-escape coverage (including drive-letter and UNC paths).
- Re-run:
  - `cd runner && npm run boundary-check`
  - `cd runner && node --test scripts/boundary-check.test.js`

## CI and portability expectations
- `just ci` stays check-only (no silent writes/lockfile churn).
- Keep Linux/macOS and Windows portability.
- Avoid introducing bash-only assumptions into core workflows.
- CI and local checks should leave repo clean (no tracked diffs, no untracked files).
- If changing Node support, update both:
  - `runner/package.json` `engines.node`
  - `.github/workflows/ci.yml` Windows Node matrix

## Copilot/Cursor and instruction files
Copilot instructions exist and must be followed:
- `/.github/copilot-instructions.md`
  - Prioritize: security, correctness, reliability, portability, maintainability.
  - Include severity + file path + impact + concrete fix in findings.
  - Keep parity with `just ci`, Go tests, runner lint/tests, and boundary check.

Additional scoped instruction files:
- `/.github/instructions/go-control-plane.instructions.md`
- `/.github/instructions/runner-boundary.instructions.md`
- `/.github/instructions/ci-tooling.instructions.md`
- `/.github/instructions/source-quality.instructions.md`
- `/.github/instructions/agent-os-docs.instructions.md`

Cursor rules status in this repo:
- No Cursor rules are currently present in `.cursorrules` or `.cursor/rules/`.

## Docs and roadmap edits
- Follow `agent-os/standards/product/roadmap-conventions.md` for roadmap changes.
- Keep standards index entries concise and accurate when adding standards.
- Maintain roadmap-to-spec traceability.
- Never modify `agent-os/doc-dump/project-idea.md`.
