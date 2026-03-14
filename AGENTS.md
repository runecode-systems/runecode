# AGENTS.md

Repo bootstrap for coding agents. Read this first, then follow the linked standards/docs for the area you are changing.

## Non-negotiables
- DO NOT edit `agent-os/doc-dump/project-idea.md`.
- Preserve trust boundaries: `runner/` is untrusted; `cmd/` and `internal/` are trusted.
- Never add runner imports/references into trusted paths (`cmd/`, `internal/`).
- Runner must not import/reference repo-root `tools/` or other cross-boundary roots outside `protocol/`.
- Runner cross-boundary file access is only allowed to `protocol/schemas/` and `protocol/fixtures/`.
- Do not add ad-hoc cross-boundary message formats outside `protocol/schemas/`.
- `protocol/schemas/manifest.json` is the authoritative inventory for checked-in protocol schemas and registries.
- `protocol/fixtures/manifest.json` is the authoritative inventory for shared protocol fixtures.
- Keep checks deterministic and CI/local parity centered on `just ci`.
- Never leak secrets, tokens, or sensitive local paths in logs, errors, fixtures, tests, or docs.

## Repo Map
- `cmd/` - trusted Go binaries
- `internal/` - trusted Go packages/helpers
- `runner/` - untrusted Node/TypeScript package
- `protocol/` - authoritative schema bundle, registries, and shared fixtures for trusted/untrusted contracts
- `tools/` - repo-local helper tools
- `agent-os/` - specs, standards, roadmap/product docs
- `docs/trust-boundaries.md` - boundary contract and prohibited bypasses
- `docs/source-quality.md` - source-quality policy and enforcement expectations

## Canonical Commands
- Show recipes: `just --list`
- Format: `just fmt`
- Lint: `just lint`
- Test: `just test`
- CI parity gate: `just ci`
- Protocol-focused checks:
  - `go test ./internal/protocolschema`
  - `cd runner && node --test scripts/protocol-fixtures.test.js`
  - `cd runner && npm run boundary-check`
- Use `justfile` as the exact command source of truth.

## If You Touch...

### `runner/`
- Read:
  - `docs/trust-boundaries.md`
  - `agent-os/standards/security/trust-boundary-interfaces.md`
  - `agent-os/standards/security/trust-boundary-layered-enforcement.md`
  - `agent-os/standards/security/trust-boundary-change-checklist.md`
  - `agent-os/standards/security/runner-boundary-check.md`
- Verify:
  - `cd runner && npm run boundary-check`
  - `cd runner && node --test scripts/boundary-check.test.js`
  - `cd runner && npm test`

### `protocol/`
- Read:
  - `agent-os/standards/global/protocol-bundle-manifest.md`
  - `agent-os/standards/global/protocol-schema-invariants.md`
  - `agent-os/standards/global/protocol-registry-discipline.md`
  - `agent-os/standards/global/protocol-canonicalization-profile.md`
  - `agent-os/standards/testing/protocol-fixture-manifest-parity.md`
  - `docs/trust-boundaries.md`
- Verify:
  - `go test ./internal/protocolschema`
  - `cd runner && node --test scripts/protocol-fixtures.test.js`
  - `cd runner && npm run boundary-check`

### `cmd/`, `internal/`, `tools/`
- Read:
  - `/.github/instructions/go-control-plane.instructions.md`
  - `agent-os/standards/global/source-quality-enforcement-layering.md`
  - `agent-os/standards/global/language-aware-source-docs.md`
- Verify:
  - relevant `go test ./...` target(s)
  - `just lint`

### `agent-os/` docs, specs, roadmap, or standards
- Read:
  - `/.github/instructions/agent-os-docs.instructions.md`
  - `agent-os/standards/product/roadmap-conventions.md` when touching `agent-os/product/roadmap.md`
- Rules:
  - Keep standards index entries concise and accurate
  - Maintain roadmap-to-spec traceability
  - Never modify `agent-os/doc-dump/project-idea.md`

## Verification Expectations
- `just ci` stays check-only; do not introduce silent writes or lockfile churn.
- Keep Linux/macOS and Windows portability.
- Avoid bash-only assumptions in core workflows.
- CI and local checks should leave the repo clean.
- If changing Node support, update both:
  - `runner/package.json` `engines.node`
  - `.github/workflows/ci.yml` Windows Node matrix

## Instructions And Standards
- Global review instructions: `/.github/copilot-instructions.md`
- Scoped instructions:
  - `/.github/instructions/go-control-plane.instructions.md`
  - `/.github/instructions/runner-boundary.instructions.md`
  - `/.github/instructions/ci-tooling.instructions.md`
  - `/.github/instructions/source-quality.instructions.md`
  - `/.github/instructions/agent-os-docs.instructions.md`
- Standards index: `agent-os/standards/index.yml`

Cursor rules status in this repo:
- No Cursor rules are currently present in `.cursorrules` or `.cursor/rules/`.
