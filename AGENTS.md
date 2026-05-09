# AGENTS.md

Repo bootstrap for coding agents. Read this first, then follow the linked standards/docs for the area you are changing.

## Non-negotiables
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
- `runecontext/` - canonical project context, standards, changes, specs, decisions, and bundles
- `docs/trust-boundaries.md` - boundary contract and prohibited bypasses
- `docs/source-quality.md` - source-quality policy and enforcement expectations

## Canonical Commands
- Show recipes: `just --list`
- Format: `just fmt`
- Lint: `just lint`
- Test: `just test`
- CI parity gate: `just ci`
- TUI snapshot review, when iterating on `cmd/runecode-tui` UI/polish:
  - Generate isolated local snapshots: `just tui-snapshot-all`
  - Preserve artifacts for inspection: `TUI_SNAPSHOT_KEEP=1 just tui-snapshot-all`
  - Non-GUI review summary: `just tui-snapshot-review`
  - Print selected artifact paths: `just tui-snapshot-review-list`
  - Explicit GUI open for manual review: `just tui-snapshot-review-open`
  - Override local snapshot temp dir: `TUI_SNAPSHOT_DIR=/tmp/my-snapshots ...`
  - CI/dev-only validation: `just tui-snapshot-ci`
  - Release safety check: `just tui-release-safety`
- Protocol-focused checks:
  - `go test ./internal/protocolschema`
  - `cd runner && node --test scripts/protocol-fixtures.test.js`
  - `cd runner && npm run boundary-check`
- Use `justfile` as the exact command source of truth.

### TUI Snapshot Usage
- Use the snapshot workflow when changing TUI layout, wording, spacing, color treatment, or other visual polish in `cmd/runecode-tui`.
- Prefer the default non-GUI review flow first so local runs stay uncluttered; only use `tui-snapshot-review-open` when a human explicitly wants desktop-opened images.
- Local snapshot runs clean before and after by default; use `TUI_SNAPSHOT_KEEP=1` only when artifacts need to persist for inspection.
- Snapshot tooling is dev/CI-only and must not be added to release artifacts or release-only verification paths.

## If You Touch...

### `runner/`
- Read:
  - `docs/trust-boundaries.md`
  - `runecontext/standards/security/trust-boundary-interfaces.md`
  - `runecontext/standards/security/trust-boundary-layered-enforcement.md`
  - `runecontext/standards/security/trust-boundary-change-checklist.md`
  - `runecontext/standards/security/runner-boundary-check.md`
- Verify:
  - `cd runner && npm run boundary-check`
  - `cd runner && node --test scripts/boundary-check.test.js`
  - `cd runner && npm test`

### `protocol/`
- Read:
  - `runecontext/standards/global/protocol-bundle-manifest.md`
  - `runecontext/standards/global/protocol-schema-invariants.md`
  - `runecontext/standards/global/protocol-registry-discipline.md`
  - `runecontext/standards/global/protocol-canonicalization-profile.md`
  - `runecontext/standards/testing/protocol-fixture-manifest-parity.md`
  - `docs/trust-boundaries.md`
- Verify:
  - `go test ./internal/protocolschema`
  - `cd runner && node --test scripts/protocol-fixtures.test.js`
  - `cd runner && npm run boundary-check`

### `cmd/`, `internal/`, `tools/`
- Read:
  - `/.github/instructions/go-control-plane.instructions.md`
  - `runecontext/standards/global/source-quality-enforcement-layering.md`
  - `runecontext/standards/global/language-aware-source-docs.md`
- Verify:
  - relevant `go test ./...` target(s)
  - `just lint`

### Planning docs, specs, roadmap, or standards
- Read:
  - `/.github/instructions/runecontext-docs.instructions.md`
  - `runecontext/standards/product/roadmap-conventions.md` when touching `runecontext/project/roadmap.md`
- Rules:
  - Keep the standards inventory doc and bundle set concise and accurate
  - Maintain roadmap-to-change/spec traceability

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
  - `/.github/instructions/runecontext-docs.instructions.md`
- Standards inventory: `runecontext/project/standards-inventory.md`

Cursor rules status in this repo:
- No Cursor rules are currently present in `.cursorrules` or `.cursor/rules/`.
