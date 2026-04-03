---
name: copilot-review-instructions-sync
description: Update and validate GitHub Copilot review instruction files from RuneCode standards, trust-boundary docs, and CI conventions.
argument-hint: "[optional focus area such as runner, go-control-plane, or ci]"
disable-model-invocation: true
---

Use this workflow when standards or repository conventions change and Copilot review instructions must stay in sync.

## Standards and references to read first

- `runecontext/project/standards-inventory.md`
- Relevant changed standards under `runecontext/standards/**`
- Relevant bundle definitions under `runecontext/bundles/*.yaml`
- `docs/trust-boundaries.md`
- `CONTRIBUTING.md`
- `justfile`
- `.github/workflows/ci.yml`

## Procedure

1. Inspect current Copilot instruction files:
   - `.github/copilot-instructions.md`
   - `.github/instructions/**/*.instructions.md`
2. Map standards and conventions to instruction scope:
   - Repository-wide policy goes in `.github/copilot-instructions.md`.
   - Domain-specific policy goes in path-specific files with `applyTo` frontmatter.
3. Keep guidance concise and review-oriented:
   - Prioritize security, correctness, reliability, portability, and maintainability.
   - Avoid style-only directives that create noisy reviews.
4. Validate `applyTo` globs and remove conflicting directives.
5. Ensure practical review context is included:
   - CI parity command: `just ci`.
   - Supporting checks: `go test ./...`, `cd runner && npm run lint`, `cd runner && npm test`, `cd runner && npm run boundary-check`.
   - Trust boundary constraints from `docs/trust-boundaries.md`.
6. Enforce Copilot code review constraints:
   - Keep each instruction file under 4000 characters.
7. Report:
   - Which standards and conventions were reflected.
   - Which instruction files changed.
   - Any remaining conventions not yet represented.

## Guardrails

- Keep instructions declarative and broadly applicable.
- Do not include secrets, credentials, or environment-specific values.
- Prefer repository source-of-truth docs over ad-hoc process notes.
