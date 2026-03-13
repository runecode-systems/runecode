# Standards for Source Quality Guardrails v0

These standards apply to implementation work produced from this spec.

## Existing Repo Standards

- `agent-os/standards/global/deterministic-check-write-tools.md`
- `agent-os/standards/ci/worktree-cleanliness.md`
- `agent-os/standards/security/trust-boundary-change-checklist.md`
- `agent-os/standards/security/trust-boundary-layered-enforcement.md`

## Documentation Placement Standard

- Do not require top-of-file docs on every source file.
- Documentation requirements must be language-aware:
  - Go: prefer package comments and exported declaration comments as the baseline.
  - JS/TS: prefer module-level docs for entrypoints, trust-boundary modules, policy-sensitive modules, and protocol adapters.
- `protocol/` artifacts that are hand-maintained and boundary-critical are in scope for maintainability rules, but generated artifacts and large fixtures are excluded.
- Generated-file exclusions must be deterministic and rely on standard generated-file markers.
- When a file format does not support meaningful inline comments, rationale must live in an adjacent maintained doc or spec rather than a forced pseudo-comment convention.
- Require module/package/file-level rationale docs when the code is security-sensitive, boundary-sensitive, or non-obvious enough that names and signatures are insufficient.
- Do not use doc comments as filler. If a doc block does not explain purpose, constraints, or usage beyond the signature/name, it is not sufficient.

## Comment Quality Standard

- Comments must explain one or more of:
  - purpose not obvious from code structure,
  - why a specific approach is required,
  - invariant or trust-boundary assumptions,
  - edge-case handling,
  - platform-specific behavior,
  - fail-closed or security-sensitive choices,
  - intentionally surprising behavior.
- Comments should not merely narrate syntax or restate variable/function names.
- Commented-out code should be removed rather than preserved inline, unless a generator or intentionally disabled snippet is required and documented.
- Commented-out-code detection should avoid false positives on valid suppression directives, annotation prefixes such as `NOTE:` / `TODO:` / `SECURITY:`, and ordinary explanatory prose.
- Tests should prefer descriptive names over explanatory comments; add comments only when fixtures or cross-platform/security behavior are not obvious.
- Test/source classification must be deterministic:
  - Go tests use `_test.go`.
  - JS/TS tests use `*.test.*`, `*.spec.*`, `__tests__/`, or a dedicated `tests/` directory.

## Complexity and Size Standard

- Prefer decomposition and clarity over writing larger files/functions and then documenting around the complexity.
- Use separate guardrails for source files and test files.
- Use explicit policy tiers:
  - Tier 1: trust-boundary, policy, secrets, audit, schema-validation, path-normalization, and similar enforcement logic.
  - Tier 2: routine commands, helpers, and lower-risk support code.
- Tiering must be deterministic:
  - `runner/**` defaults to Tier 2 unless checked-in checker configuration explicitly marks files as Tier 1.
  - tools that enforce guardrails, touch `protocol/schemas/**`, or generate code for trusted paths are Tier 1 regardless of location.
  - policy and enforcement documents may be Tier 1 protected surfaces even when numeric code budgets do not apply.
- Initial default budgets are:
  - Tier 1 source files: 250 SLOC
  - Tier 2 source files: 400 SLOC
  - Tier 1 test files: 500 SLOC
  - Tier 2 test files: 800 SLOC
- Initial function-complexity targets are:
  - metric: cognitive complexity
  - Tier 1: max 10
  - Tier 2: max 15
- Initial function-length targets are:
  - Tier 1: 40 lines
  - Tier 2: 60 lines
- Go function-length measurement should use the function body span rather than the full declaration span.
- Use ratcheted exception mechanisms for legacy oversized files instead of indefinite blanket exemptions.
- The ratchet mechanism uses a checked-in repo-root file named `.source-quality-baseline.json` with per-file caps and rationales.
- New code and high-risk code should be held to stricter defaults first.
- Function complexity checks are preferred over raw file-size checks when the concern is hard-to-reason-about control flow.

## Architecture-Rationale Standard

- System or subsystem rationale belongs in maintained docs/specs/ADRs when it spans multiple functions, files, or components.
- Inline comments should capture local invariants and hazards, not become substitutes for design documentation.
- If a reviewer needs paragraphs to understand a trust-boundary, policy, or security decision, the implementation should add or update a maintained doc in addition to any local comments.

## Suppression Standard

- `//nolint`, `eslint-disable`, and similar suppressions must include a specific reason.
- Acceptable reasons explain why the rule does not fit, what invariant is being preserved, or what follow-up debt remains.
- Bare suppressions or vague suppressions such as "legacy" without context are non-compliant.
- Suppressions are tiered:
  - ordinary source-quality suppressions may be allowed inline with a specific reason,
  - Tier 1 security- or boundary-sensitive suppressions should be prohibited inline by default and replaced with checked-in, reviewed checker-owned configuration.

## Enforcement Standard

- Prefer deterministic, low-subjectivity checks in CI.
- Reuse language-native tooling where it cleanly fits.
- Use a repo-specific source-quality gate only for policies that cannot be expressed well in existing tools or that must span both Go and JS/TS.
- The repo-wide checker should live in a trusted location such as `tools/`, not under `runner/`.
- V0 should use a version-pinned `golangci-lint` binary for Go source-quality enforcement with a deliberately small, checked-in, reviewed configuration.
- V0 may add a minimal ESLint layer for JS/TS when it materially improves enforcement, but it should stay intentionally small and pinned.
- The repo-specific checker remains required for cross-language, path-tiered, and ratcheted policies.
- Prefer built-in/core lint rules before adding third-party plugin surfaces.
- `just lint` and `just ci` remain the canonical enforcement entrypoints.
- Source-quality checks must be check-only and must not modify tracked files.
- Failure behavior is fail-closed:
  - no eligible files found => fail,
  - eligible-file read/parse failure => fail,
  - policy violation => fail.
- Requested scan roots must remain within the current repo/workspace boundary.
- Exit semantics should be consistent and predictable:
  - pass => `0`
  - check failure / policy violation => `1`
  - usage or configuration error => `2`
- Violation output should be actionable, including rule name, file path, local context when available, observed versus expected value, and suggested remediation category.
- The checker itself must have automated tests and must pass on its own implementation before merge.

## Trust-Boundary Emphasis

- Trusted and untrusted boundary code is held to a higher documentation standard than ordinary helpers.
- Path normalization, access-control logic, schema/policy enforcement, secrets handling, audit decisions, and similar logic must be easy to review and must not rely on implicit behavior alone.
- When such code remains complex after reasonable refactoring, comments should explain the invariant or boundary rule directly adjacent to the logic.

## Guardrail Surface Protection Standard

- Source-quality tooling, baseline files, threshold configuration, and `just` integration are protected surfaces because changing them can weaken enforcement.
- Linter configuration and version-pinning files are also protected surfaces because changing them can silently widen or weaken enforcement.
- These surfaces should be covered by explicit review ownership and must be easy to audit in code review.
- Reviewer and agent instruction files should be updated together with source-quality policy changes so human guidance and mechanical enforcement stay aligned.
