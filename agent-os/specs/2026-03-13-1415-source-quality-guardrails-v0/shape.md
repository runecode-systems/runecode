# Source Quality Guardrails v0 — Shaping Notes

## Scope

Define a language-aware source-quality policy for RuneCode that keeps critical code understandable as the project grows.

This spec covers file/module documentation expectations, comment quality, function and file size guardrails, complexity checks, suppression hygiene, and where architectural rationale should live.

This spec does not require blanket "comment everything" rules, does not require top-of-file docs on every file, and does not attempt to solve maintainability with a single raw line-count threshold.

## Decisions

- RuneCode adopts language-aware documentation rules instead of one universal comment rule:
  - Go: require package comments and exported API docs where idiomatic; reserve file-level/module-level docs for security-critical or non-obvious packages/files.
  - JS/TS: require top-of-file module docs for runner entrypoints, trust-boundary code, policy/protocol adapters, and other security-sensitive or subtle modules.
  - Tests may use lighter documentation requirements when names and structure are already clear.
- `protocol/` is partially in scope:
  - hand-maintained schema and boundary-critical protocol files are in scope for size/maintainability guardrails,
  - generated protocol artifacts and large fixtures are excluded,
  - when a protocol artifact format does not support useful inline comments, rationale must live in an adjacent maintained doc rather than forced inline.
- RuneCode explicitly rejects the idea that maintainability is improved by blanket boilerplate comments.
  - Comments must explain why, invariants, trust-boundary constraints, threat-model assumptions, edge-case handling, or non-obvious tradeoffs.
  - Comments that merely restate syntax or narrate obvious code flow are noise and should not be required.
- Complex logic should first be simplified, split, or isolated before comments are added.
  - Inline comments are for remaining non-obvious logic after reasonable decomposition.
  - Architecture/subsystem rationale belongs in maintained docs/specs/ADRs rather than long explanatory prose buried inside one file.
- Source-quality enforcement has at least two policy tiers:
  - Tier 1 (strictest): trust-boundary, policy, secrets, audit, schema-validation, path-normalization, and other security-critical enforcement logic.
  - Tier 2 (standard): ordinary commands, helpers, utilities, and lower-risk support code.
- Tier classification must be deterministic rather than reviewer-implied.
  - `runner/**` defaults to Tier 2 unless checked-in checker configuration explicitly marks a file as Tier 1.
  - tools that enforce guardrails, touch `protocol/schemas/**`, or generate code for trusted paths are Tier 1 regardless of directory depth.
  - policy/enforcement documents such as `docs/source-quality.md`, `docs/trust-boundaries.md`, `.github/instructions/**`, and `agent-os/standards/**` are Tier 1 protected surfaces for documentation/review rigor even when numeric code budgets do not apply.
- Initial thresholds are explicit and intentionally provisional so implementation does not invent policy ad hoc.
  - Tier 1 source files target about 250 SLOC; Tier 2 source files target about 400 SLOC.
  - Tier 1 test files target about 500 SLOC; Tier 2 test files target about 800 SLOC.
  - Cognitive complexity is the primary function-complexity policy metric.
  - Tier 1 functions target a maximum cognitive complexity of `10`; Tier 2 functions target a maximum of `15`, with tool mappings defined during implementation.
  - Provisional function-length targets are stricter in Tier 1 than Tier 2.
- File-size and function-size enforcement should be selective and ratcheted.
  - Prefer separate source/test budgets.
  - Prefer a small, checked-in ratchet baseline file over repo-wide hard limits that instantly fail on existing debt.
  - Apply stricter rules first to new code and high-risk paths.
- Complexity checks are a better first-class signal than comment volume.
  - V0 should add `golangci-lint` for Go because it gives mature, low-friction coverage for complexity, size, and documentation-adjacent rules in trusted code.
  - JS/TS may add a minimal ESLint setup in v0 when it materially improves runner enforcement, but it should stay intentionally small and avoid plugin sprawl.
  - The repo-specific checker remains required because several RuneCode policies are cross-language, path-tiered, and repo-specific.
- Suppression handling is tiered:
  - source-quality suppressions (`//nolint`, `eslint-disable`, etc.) must carry a specific reason,
  - security- or boundary-critical suppressions are stricter and should be prohibited inline in Tier 1 paths unless implemented through checked-in, reviewed checker-owned configuration.
- Source-quality tooling is itself a protected surface.
  - The repo-wide checker should live in a trusted location such as `tools/`, not in `runner/`.
  - Tooling, thresholds, baseline files, and `just` wiring must receive explicit review because changing the checker can weaken enforcement.
  - Third-party lint dependencies should be pinned and kept intentionally small so source-quality enforcement does not create unnecessary supply-chain or maintenance risk.
- Source-quality enforcement must stay deterministic, CI-friendly, and cross-platform.
  - `just lint` and `just ci` remain the canonical gates.
  - Checks must avoid unstable heuristics that create churn or subjective failures.
  - Failure behavior is fail-closed: missing file discovery, parse/read failures, and violations all fail the check.

## Context

- Product alignment: RuneCode is security-focused and trust-boundary-heavy; code needs to stay understandable under audit and future hardening work.
- Repo context:
  - `justfile`
  - `runner/package.json`
  - `docs/trust-boundaries.md`
  - `AGENTS.md`
- Current pain/risk this spec addresses:
  - files may grow without decomposition pressure,
  - complex security-relevant logic may remain under-documented,
  - comment quality is currently mostly social convention rather than mechanically reinforced.
- Current implementation-reality assumptions this spec now resolves:
  - V0 should intentionally add `golangci-lint` for Go rather than waiting for a later phase.
  - V0 may add only a minimal ESLint layer for JS/TS if it provides clear value; complex plugin stacks are not required.
  - The initial ratchet mechanism should stay simple because repo-wide legacy debt is still small.
  - The Task 2 policy doc should be specific enough that implementers do not need to infer critical enforcement details from multiple later tasks.
- Research direction informing this spec:
  - mature projects usually combine targeted docs, complexity limits, and architecture docs,
  - they rarely require comments everywhere,
  - they often avoid blanket file caps except through repo-specific or ratcheted guardrails.
- Related specs:
  - `agent-os/specs/2026-03-08-1039-scaffold-ci-matrix/`
  - `agent-os/specs/2026-03-08-1039-protocol-schemas-v0/`
  - `agent-os/specs/2026-03-08-1039-policy-engine-v0/`
  - `agent-os/specs/2026-03-08-1039-broker-local-api-v0/`

## Standards Applied

- `agent-os/standards/global/deterministic-check-write-tools.md`
- `agent-os/standards/ci/worktree-cleanliness.md`
- `agent-os/standards/security/trust-boundary-change-checklist.md`
- `agent-os/standards/security/trust-boundary-layered-enforcement.md`
