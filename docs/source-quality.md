# Source Quality Policy

## Purpose

RuneCode is a security-focused, trust-boundary-heavy codebase. Source quality in this repo means code stays understandable under review, audit, and future hardening work without relying on boilerplate comments or oversized files.

This policy defines what RuneCode means by source quality, what is in scope, what is intentionally out of scope, which paths receive stricter enforcement, and which expectations must be explicit enough for deterministic tooling.

## Related Documents

- `docs/trust-boundaries.md`
- `AGENTS.md`
- `agent-os/specs/2026-03-13-1415-source-quality-guardrails-v0/`

## What Source Quality Means In RuneCode

RuneCode source quality focuses on:

- file and module size staying small enough to review safely
- function size and cognitive complexity staying understandable
- language-appropriate documentation in the places where maintainers expect it
- comments explaining why, invariants, trust-boundary assumptions, or subtle behavior
- suppression comments carrying a specific reason rather than acting as silent escapes
- ratcheting legacy exceptions downward over time instead of normalizing them

RuneCode does not treat comment volume as a quality goal. The goal is maintainability, auditability, and safe change velocity.

## In Scope

This policy applies to hand-maintained repo code and policy surfaces that define or explain enforcement behavior, including:

- `cmd/`
- `internal/`
- `runner/` (the untrusted domain; see `docs/trust-boundaries.md`)
- `tools/`
- hand-maintained protocol definitions in `protocol/`
- policy and enforcement documents that shape review or CI behavior, including:
  - `docs/source-quality.md`
  - `docs/trust-boundaries.md`
  - `.github/copilot-instructions.md`
  - `.github/instructions/**`
  - `runecontext/standards/**`

The following concern areas are in scope:

- file and module decomposition pressure
- function complexity and size
- documentation placement and minimum expectations
- comment quality and anti-boilerplate rules
- suppression hygiene
- ratcheted handling for legacy exceptions

## Out Of Scope

This policy does not apply the same way to every artifact.

Excluded or reduced-scope surfaces:

- generated files identified by a standard generated-file marker such as `Code generated ... DO NOT EDIT.` or another language-appropriate generated header
- vendored dependencies
- build output directories
- lockfiles
- large protocol fixtures and other non-source test data
- planning-only docs under `agent-os/specs/**` and `runecontext/project/**`, unless a later checker explicitly adds doc-quality rules for them

If an artifact format does not support meaningful inline comments, the rationale should live in an adjacent maintained document or spec instead of a forced pseudo-comment convention.

## Language-Aware Baseline

RuneCode does not use a single documentation model for every language.

### Go

- Prefer package comments and exported declaration comments as the baseline.
- Do not require a top-of-file doc block on every large Go file.
- Require extra file- or package-level rationale when code contains trust-boundary rules, path normalization, policy decisions, cryptographic handling, schema enforcement, secrets handling, or similarly subtle behavior.

### JS/TS

- Prefer top-of-file module docs for runner entrypoints, trust-boundary modules, policy-sensitive modules, protocol adapters, and non-obvious modules.
- Do not require blanket JSDoc or module docs on every helper.

### Tests

- Prefer descriptive test names and table cases before explanatory comments.
- Reserve comments for surprising fixtures, security coverage, and cross-platform edge cases.
- Test-file classification is deterministic:
  - Go tests use `_test.go`.
  - JS/TS tests use `*.test.*`, `*.spec.*`, `__tests__/`, or a dedicated `tests/` directory.

### Protocol Files

- Hand-maintained schemas and boundary-critical protocol definitions are in scope for maintainability guardrails.
- Generated protocol artifacts and large fixtures are excluded.
- Hand-maintained protocol definitions should document trust-boundary constraints, validation rules, and non-obvious assumptions either inline or in an adjacent maintained doc when the format is comment-hostile.

## Comment Quality Baseline

Good comments explain one or more of:

- why a choice exists
- invariant or trust-boundary assumptions
- security-sensitive or fail-closed behavior
- path normalization or validation rules
- platform-specific behavior
- reasons a simpler-looking implementation would be unsafe or incorrect

Comments should not merely narrate obvious syntax or repeat the next line in prose.

Commented-out code should be removed rather than preserved inline, unless a generator or intentionally disabled snippet requires it and that exception is documented.

## Architecture-Rationale Expectations

Subsystem-level rationale belongs in maintained docs, specs, or ADRs when it spans multiple functions, files, or components.

This especially applies to:

- trust-boundary rules
- policy decisions
- protocol and schema validation
- secrets or credential handling
- cryptographic behavior
- audit-critical flows

Inline comments should capture local invariants and hazards, not attempt to become the full design document.

Reviewers should ask for a maintained doc, spec update, or ADR-style note when:

- a trust-boundary rule only becomes understandable after reading multiple files
- a policy decision or security tradeoff needs paragraph-level justification
- a protocol or validation flow depends on cross-component invariants
- a change adds complexity that cannot be justified by local inline comments alone

## Path-Tier Map

RuneCode uses two source-quality policy tiers plus a small set of documentation surfaces that are protected for review purposes.

### Tier 1: Strictest Enforcement

Treat the following as Tier 1 by default:

- `internal/**`
- repo guardrail and policy-enforcing tools in `tools/**`
- hand-maintained `protocol/schemas/**`
- boundary/security-sensitive scripts

Treat the following as Tier 1 protected policy surfaces:

- `docs/source-quality.md`
- `docs/trust-boundaries.md`
- `.github/copilot-instructions.md`
- `.github/instructions/**`
- `runecontext/standards/**`

Runner classification is deterministic:

- `runner/**` defaults to Tier 2.
- Only runner files explicitly listed in `.source-quality-config.json` as trust-boundary, policy, or guardrail enforcement code are Tier 1.
- Current examples include the runner boundary guardrail under `runner/scripts/`.

Tools classification is also deterministic:

- Any tool that enforces guardrails, touches `protocol/schemas/**`, or generates code consumed by trusted paths is Tier 1 regardless of its location under `tools/`.

Tier 1 protected policy surfaces are held to stricter documentation, review, and suppression expectations. Numeric source-code budgets apply only where a source-like file format and checker meaningfully support them.

### Tier 2: Standard Enforcement

Treat the following as Tier 2 by default:

- `cmd/**`
- lower-risk helpers in `tools/**` that do not enforce guardrails, touch `protocol/schemas/**`, or generate code for trusted paths
- routine runner modules that do not implement trust-boundary or policy enforcement

Planning-oriented docs under `agent-os/specs/**` and `runecontext/project/**` remain outside code-size and function-complexity enforcement unless a future checker adds separate doc-quality rules.

## Initial Policy Defaults

These are the current provisional defaults for implementation work:

| Category | Tier 1 | Tier 2 |
|----------|--------|--------|
| Source file target | 250 SLOC | 400 SLOC |
| Test file target | 500 SLOC | 800 SLOC |
| Function cognitive complexity max | 10 | 15 |
| Function length target | 40 lines | 60 lines |

These defaults apply to new code unless a reviewed exception is recorded.

The Tier 1 test-file target is intentionally provisional. If real table-driven security tests show that `500` SLOC is too tight, the threshold may be adjusted with explicit review and documented rationale rather than silently drifting upward.

## Ratchet Baseline And Reviewed Exceptions

RuneCode uses a checked-in repo-root file named `.source-quality-baseline.json` for reviewed legacy exceptions.

Each baseline entry should be explicit about what is temporarily permitted, including:

- file path
- source or test classification
- allowed SLOC cap
- allowed cognitive-complexity cap if applicable
- allowed function-length cap if applicable
- rationale
- follow-up reference when one exists

Baseline entries may be reduced or removed over time, but should not be increased without explicit review and written justification.

Any deterministic Tier 1 runner override or other checker-owned exception should also live in `.source-quality-config.json` rather than relying on reviewer memory.

## Suppression Expectations

- Ordinary source-quality suppressions may exist only with a specific reason.
- Bare suppressions or vague reasons like `legacy` are not sufficient.
- Tier 1 security- or boundary-sensitive suppressions are prohibited inline by default.
- Any reviewed Tier 1 suppression exception must live in `.source-quality-config.json` with a rationale and review reference; it must not be a casual inline escape.

## Enforcement Expectations

- `just lint` and `just ci` are the canonical enforcement entrypoints.
- Go source-quality enforcement should use a small, version-pinned `golangci-lint` binary and a checked-in, reviewed configuration.
- JS/TS may use a minimal, version-pinned ESLint layer if it materially improves enforcement.
- A trusted repo-specific checker remains required for cross-language, path-tiered, and ratcheted rules.
- In the current v0 rollout, `golangci-lint` acts as a repo-wide floor while the repo-specific checker carries stricter Tier 1 differentiation where path-specific policy is tighter than the global Go linter settings.
- Source-quality checks are check-only and fail closed on missing files, parse/read failures, or policy violations.
- Exit semantics should be consistent:
  - pass => `0`
  - check failure or policy violation => `1`
  - usage or configuration error => `2`
- Violation output should include:
  - rule name
  - file path
  - line, function, or module context when available
  - observed value versus expected value or threshold
  - suggested remediation category

### Hard Failures vs Review Guidance

Hard-fail enforcement should stay narrow, deterministic, and tool-friendly. In v0 that means the automated gate is appropriate for:

- file-size and function-size budgets
- cognitive-complexity limits where a deterministic checker exists
- required Tier 1 module docs
- suppression-comment and Tier 1 exception handling
- generated-file, parse, and scan-root fail-closed behavior

Review guidance should remain responsible for higher-judgment questions such as:

- whether a comment truly explains the right invariant or tradeoff
- whether a subsystem needs a spec/doc/ADR update rather than more inline commentary
- whether code should be decomposed further even when it technically fits under current thresholds
- whether a new ESLint layer would add enough value to justify more dependency surface

## Protected Enforcement Surfaces

The following files and paths are protected source-quality surfaces because changing them can weaken enforcement or reviewer expectations:

- `tools/checksourcequality/**`
- `.source-quality-baseline.json`
- `.source-quality-config.json`
- `.golangci.yml`
- `runner/eslint.config.*` (if added)
- `justfile`
- `docs/source-quality.md`
- `.github/copilot-instructions.md`
- `.github/instructions/**`

Changes to these surfaces should receive explicit review and remain easy to audit in diffs.

## Review Guidance

When reviewing code against this policy, prioritize:

- oversized files or functions in Tier 1 paths
- missing documentation where the language-aware baseline requires it
- noisy comments that restate the code
- complex logic with no explanation of invariants or constraints
- suppression comments without a concrete reason
- rationale that belongs in a maintained doc rather than being hidden in one file

Style-only nitpicks are not the goal. Review should focus on maintainability, auditability, trust-boundary clarity, and future-safe change velocity.
