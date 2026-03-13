# Source Quality Guardrails v0

User-visible outcome: RuneCode gains clear, enforceable source-quality guardrails that keep trusted and untrusted code understandable as the codebase grows, without requiring noisy boilerplate comments.

This spec is intentionally policy-first. It defines what should be enforced, where rules differ by language and risk level, and how rollout should avoid freezing development under legacy debt.

## Task 1: Save Spec Documentation

Ensure `agent-os/specs/2026-03-13-1415-source-quality-guardrails-v0/` contains:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty placeholder until visuals are added later)

Parallelization: docs-only; safe to do anytime.

## Task 2: Define the Source-Quality Policy Surface

- Document the categories of source-quality concerns RuneCode will enforce:
  - file/module size and decomposition pressure,
  - function size and logic complexity,
  - documentation placement and minimum expectations,
  - comment quality and anti-boilerplate rules,
  - suppression hygiene,
  - rollout/ratchet policy for legacy code.
- Make the policy explicitly language-aware.
  - Go and JS/TS must not share a fake "one size fits all" doc model.
  - High-risk paths may have stricter requirements than low-risk paths.
- Define which paths count as high-risk or boundary-sensitive for this policy.
  - Required tiering for v0:
    - Tier 1 (strictest): security- and boundary-sensitive enforcement logic, including policy evaluation, schema validation, secrets handling, audit enforcement, path normalization, trust-boundary guardrails, and similar code under `internal/`, selected `tools/`, and selected `runner/` modules.
    - Tier 2 (standard): ordinary command wiring in `cmd/`, lower-risk helpers in `tools/`, and routine runner modules not implementing trust-boundary or policy enforcement.
  - The tier map must include protected policy/enforcement documentation surfaces.
    - Minimum Tier 1 protected surfaces:
      - `docs/source-quality.md`
      - `docs/trust-boundaries.md`
      - `.github/copilot-instructions.md`
      - `.github/instructions/**`
      - `agent-os/standards/**`
    - Planning-oriented docs under `agent-os/specs/**` and `agent-os/product/**` may remain outside code-size and function-complexity enforcement unless a future checker adds separate doc-quality rules.
  - Runner classification must be deterministic.
    - `runner/**` defaults to Tier 2.
    - Tier 1 runner modules must be explicitly listed by checked-in checker configuration rather than inferred ad hoc during review.
  - Tools classification must also be deterministic.
    - Any tool that enforces guardrails, touches `protocol/schemas/**`, or generates code consumed by trusted paths is Tier 1 regardless of location.
  - `protocol/` scope must be explicit:
    - hand-maintained schemas and boundary-critical protocol definitions are in scope for size/maintainability guardrails,
    - generated schemas, generated artifacts, and large fixtures are excluded,
    - trust-boundary constraints, validation rules, and non-obvious assumptions must be documented inline or in an adjacent maintained doc when the format is comment-hostile.
  - Generated files, vendored code, build outputs, and lockfiles are excluded.
    - Generated files must be identifiable through standard generated-file markers so checker exclusions stay deterministic.

Deliverables:
- A written policy section or repo doc that explains what RuneCode means by "source quality" and what is intentionally out of scope.
- A path-tier map that distinguishes Tier 1 and Tier 2 enforcement.
- The Task 2 policy doc must also surface core implementation-facing decisions needed for deterministic rollout:
  - cognitive complexity is the primary policy metric for function complexity,
  - provisional function-length defaults are documented,
  - `.source-quality-baseline.json` is named as the ratchet baseline file,
  - exit-code semantics and violation-output expectations are summarized,
  - architecture-rationale expectations are documented,
  - Tier 1 reviewed suppressions and runner overrides require checked-in checker-owned configuration rather than casual inline comments.

Parallelization: can be drafted in parallel with implementation planning, but must be stable before check thresholds are finalized.

## Task 3: Add Language-Aware Documentation Rules

- Go documentation policy:
  - Require package comments for packages that are public, reused, boundary-sensitive, or security-relevant.
  - Require exported declaration comments where Go tooling and review norms expect them.
  - Do not require a top-of-file doc block on every large Go file.
  - Require a file-level or package-level rationale doc only when a package/file contains subtle invariants, trust-boundary enforcement, path normalization, policy decisions, cryptographic handling, or similarly non-obvious behavior.
- JS/TS documentation policy:
  - Require a top-of-file module doc block for:
    - runner entrypoints,
    - trust-boundary guardrails,
    - protocol/schema adapters,
    - policy-sensitive modules,
    - files whose purpose is not obvious from exports and names alone.
  - Do not require blanket JSDoc or top-of-file docs on every small helper.
- Test documentation policy:
  - Prefer descriptive test names and table cases first.
  - Allow sparse comments in tests when names and fixtures are already clear.
  - Require comments only for surprising fixtures, non-obvious security coverage, or cross-platform edge cases.
  - Use deterministic test-file classification:
    - Go tests are files matching `_test.go`.
    - JS/TS tests are files matching `*.test.*`, `*.spec.*`, files under `__tests__/`, or files under a dedicated `tests/` directory.

Deliverables:
- Written documentation rules that explain where top-of-file docs are mandatory, optional, or discouraged.
- A path-based classification for stricter docs in security-sensitive code.

Parallelization: can be designed in parallel with comment-quality rules; both should land together to avoid contradictory guidance.

## Task 4: Define Comment Quality Rules (No Boilerplate)

- Explicitly prohibit comment patterns that add little or no value, such as:
  - comments that restate the next line in plain English,
  - placeholder docs that only repeat names or signatures,
  - large commented-out code blocks,
  - module headers that add no purpose/risk/invariant information.
- Explicitly encourage comments that explain:
  - why a choice exists,
  - security/trust-boundary assumptions,
  - normalization/invariant rules,
  - subtle ordering constraints,
  - platform quirks,
  - intentionally fail-closed behavior,
  - reasons a simpler-looking implementation would be unsafe or incorrect.
- Require reasons on lint suppression comments when they waive source-quality checks.
  - "legacy" alone is not sufficient; the reason should say what would break or why the rule does not fit.
- Separate ordinary source-quality suppressions from security-sensitive suppressions.
  - Inline suppressions for ordinary source-quality rules may be allowed with a specific reason.
  - Inline suppressions for security-, trust-boundary-, policy-, schema-, or secrets-related checks in Tier 1 code should be prohibited by default and replaced with an explicit reviewed exception mechanism.

Implementation guidance:
- Prefer mechanical checks for low-subjectivity cases (commented-out code, bare suppression comments without reasons, obvious placeholder docs).
- Leave nuanced prose-quality judgment to review when automation would be too brittle.
- Commented-out-code detection must favor low false-positive behavior over broad pattern matching.
  - Valid suppression directives must not also be flagged as commented-out code.
  - Annotation prefixes such as `NOTE:`, `TODO:`, and `SECURITY:` must remain allowed when they contain prose guidance rather than preserved code.
  - Ordinary explanatory prose that begins with words like `if`, `for`, or `return` must not be treated as code unless the comment also has clear code-like structure.

Parallelization: can be implemented in parallel with complexity checks, but suppression-comment requirements should be shared across both.

## Task 5: Add File and Function Guardrails

- Introduce source-quality checks that enforce decomposition pressure without causing immediate repo-wide churn.
- File-size policy:
  - Use source lines of code (or similarly meaningful counting) rather than raw total lines where practical.
  - Use separate budgets for source and test files.
  - Use the following initial defaults unless later implementation evidence justifies a documented change:
    - Tier 1 source files: 250 SLOC target
    - Tier 2 source files: 400 SLOC target
    - Tier 1 test files: 500 SLOC target
    - Tier 2 test files: 800 SLOC target
  - Prefer new-file defaults plus a simple ratcheted baseline for oversized existing files.
- Function-size / complexity policy:
  - V0 should add `golangci-lint` for Go enforcement.
  - Initial Go enforcement should use a deliberately small rule set focused on source quality rather than broad style churn.
  - Candidate Go checks include `funlen`, `gocyclo`, `cyclop`, `gocognit`, exported/package comment enforcement where applicable, commented-out-code detection, and suppression hygiene where supported.
  - `golangci-lint` version/configuration must be pinned and kept deterministic across local and CI execution.
  - JS/TS implementation may start with repo-local checks and may optionally add a minimal ESLint setup in v0 if it provides clear enforcement value.
  - If ESLint is added in v0, keep it intentionally small:
    - prefer core ESLint rules first,
    - use only a minimal rule set related to complexity and maintainability,
    - avoid broad plugin sprawl unless there is a concrete policy need.
  - The repo-specific checker remains required even when `golangci-lint` and/or ESLint are present.
  - Initial function-level targets:
    - policy metric: cognitive complexity
    - Tier 1 functions: max `10`
    - Tier 2 functions: max `15`
    - provisional function-length defaults: Tier 1 `40` lines, Tier 2 `60` lines
    - Go function-length measurement should use the function body span rather than the full declaration span.
  - Start with high-risk directories first if repo-wide rollout would create too much immediate debt.
- Rollout policy:
  - New code should meet current defaults.
  - Existing exceptions should be explicit and ratcheted downward over time.
  - No silent grandfathering; all exceptions must be named and reviewable.

Deliverables:
- A concrete policy for size/complexity thresholds and where they apply.
- A checked-in ratchet baseline format for legacy oversized files.

Required ratchet format:
- Use a checked-in repo-root file named `.source-quality-baseline.json`.
- Each entry must be per-file and explicit about what is being temporarily permitted, for example:
  - file path,
  - source/test classification,
  - current allowed SLOC cap,
  - current allowed complexity cap if applicable,
  - short rationale,
  - issue or follow-up reference if one exists.
- Baseline entries may be removed or reduced over time, but should not be increased without explicit review and written justification.
- Any deterministic Tier 1 runner override or other reviewed checker-owned exception must live in checked-in checker configuration rather than reviewer memory.
- Checker implementation details decided during v0 rollout:
  - Generated-file detection should scan multiple leading non-blank lines so standard generated markers still work when a copyright header comes first.
  - Root-path validation should fail closed when a requested scan root escapes the current repo/workspace boundary.
  - File discovery should avoid unnecessary double-reads of source content so the same bytes are used for generated-file classification and rule evaluation.

Parallelization: threshold selection can be done in parallel with tooling prototyping, but must converge before `just lint` integration.

## Task 6: Add Architecture-Rationale Expectations

- Define when rationale belongs outside the source file.
  - If a subsystem needs paragraphs to explain policy, threat boundaries, or cross-component behavior, that belongs in docs/specs/ADRs rather than only in inline comments.
- Require maintained design context for high-risk subsystems such as:
  - trust-boundary checks,
  - policy decisions,
  - protocol/schema validation,
  - secrets/credential handling,
  - cryptographic or audit-critical flows.
- Inline source comments should then point to the local invariant or hazard, not attempt to become the full design doc.

Deliverables:
- A policy that distinguishes inline comments from subsystem rationale docs.
- Clear guidance for reviewers on when to ask for docs/spec updates instead of more inline commentary.

Parallelization: can be written in parallel with source-quality scripting because it is primarily a documentation/process decision.

## Task 7: Choose Enforcement Mechanisms and Rollout Path

- Decide which guardrails belong in existing linters versus a repo-specific `check-source-quality` script.
- Expected split:
  - `golangci-lint` as the required Go enforcement layer for language-native checks,
  - minimal ESLint as an optional JS/TS enforcement layer in v0 when the added value justifies the dependency,
  - a repo-specific script for cross-language policies such as:
    - file/module budgets,
    - module doc requirements by path/category,
    - ratcheted legacy caps,
    - suppression-comment reason enforcement,
    - policy/report formatting aligned with RuneCode.
- The repo-wide checker should live in a trusted location such as `tools/`, not under `runner/`.
- The repo-specific checker is still mandatory; linters supplement it but do not replace it.
- Keep third-party linting surface area intentionally constrained:
  - pin versions,
  - prefer built-in/core rules before plugins,
  - justify any added plugin or linter beyond the baseline set.
- Integrate the chosen checks into the canonical flow:
  - `just lint`
  - `just ci`
  - any runner/package-level lint command updates required for JS/TS enforcement.
- Preserve deterministic, check-only behavior.
  - The source-quality gate must not rewrite files.
  - Output should make it obvious whether the fix is "split code", "add rationale doc", "replace boilerplate comment", or "justify suppression".

Required failure behavior:
- No eligible files found for a declared scan scope => fail closed.
- Read or parse failures for an eligible file => fail closed and report the file path.
- Policy violations => fail and report all discovered violations.
- Usage/configuration error => exit `2`.
- Check failure or violation => exit `1`.
- Successful check => exit `0`.

Required output shape:
- Each violation should include:
  - rule name,
  - file path,
  - line/function/module context when available,
  - observed value versus threshold or expectation,
  - suggested remediation category.
- Machine-readable summary output is desirable but non-blocking for v0.

Required verification:
- The checker must have automated tests covering:
  - pass/fail behavior,
  - threshold boundary conditions,
  - suppression parsing and reason enforcement,
  - ratchet baseline behavior,
  - commented-out-code false-positive regressions for prose comments, suppression comments, and allowed annotation prefixes,
  - cognitive-complexity violations,
  - fail-closed behavior,
  - cross-platform path handling if the checker resolves paths.
- The checker must pass on its own implementation files before merge.

Deliverables:
- Final enforcement plan showing which rules are hard failures, which are rollout warnings, and which remain review guidance only.
- A trusted implementation path for the checker and its tests.
- A pinned `golangci-lint` integration plan for Go.
- If ESLint is adopted in v0, a minimal pinned ESLint integration plan for JS/TS.

Parallelization: implementation planning can happen in parallel with policy drafting, but integration into canonical CI commands should land only after the policy is settled.

## Task 8: Protect Guardrail Surfaces and Sync Instructions

- Protect source-quality enforcement surfaces with explicit review ownership.
  - Minimum protected files/paths:
    - `tools/` implementation for the repo-wide checker,
    - `.source-quality-baseline.json`,
    - `justfile` entries that invoke the checker,
    - any source-quality config files,
    - reviewer/agent instruction files updated by this spec.
- Ensure changes that weaken thresholds, broaden exclusions, or suppress Tier 1 rules are easy to detect during review.
- Update ownership/docs so enforcement tooling cannot be silently weakened by routine edits.

Deliverables:
- Ownership and review guidance for source-quality enforcement surfaces.

Parallelization: can be planned in parallel with tooling work, but should land before the checker becomes authoritative in CI.

## Task 9: Document Reviewer Expectations

- Update review guidance so reviewers consistently look for:
  - oversized files/functions in risky paths,
  - missing module/package docs where the policy requires them,
  - noisy comments that restate code,
  - complex logic with no explanation of invariants or constraints,
  - suppression comments without a reason,
  - rationale that belongs in subsystem docs rather than one file.
- Make it explicit that style-only nitpicks are not the goal.
  - The goal is maintainability, auditability, and future-safe change velocity.
- Sync reviewer and agent instruction files with the final policy so repo guidance does not contradict the checker.
  - Minimum sync targets:
    - `.github/copilot-instructions.md`
    - `.github/instructions/go-control-plane.instructions.md`
    - `.github/instructions/runner-boundary.instructions.md`
    - any future source-quality-specific instruction file if one is added

Deliverables:
- Reviewer guidance aligned with the source-quality policy.
- Updated instruction-file expectations aligned with the same policy.

Parallelization: can be implemented in parallel with tooling work, but should be finalized before the rules become mandatory.

## Acceptance Criteria

- RuneCode has a documented, language-aware source-quality policy rather than a blanket "comment more" rule.
- The policy distinguishes:
  - Go package/exported docs from JS/TS module docs,
  - inline comments from architecture rationale docs,
  - source/test budgets,
  - new-code requirements from ratcheted legacy exceptions.
- Boilerplate comments are explicitly discouraged, while non-obvious invariants and trust-boundary logic are explicitly called out as comment-worthy.
- Source-quality suppressions require reasons, and Tier 1 security-sensitive suppressions are handled through a stricter reviewed exception path.
- A concrete enforcement path exists for `just lint` / `just ci`.
- Rollout avoids immediate repo-wide breakage while still applying real pressure to new and high-risk code.
- Initial thresholds, tiering, and the ratchet-baseline format are specified well enough that implementation does not have to invent policy.
- The checker has explicit fail-closed behavior, deterministic exit semantics, and automated tests.
- Source-quality enforcement surfaces and reviewer instruction files are covered by the rollout plan.
