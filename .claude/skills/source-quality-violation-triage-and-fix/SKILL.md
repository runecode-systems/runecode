---
name: source-quality-violation-triage-and-fix
description: Triage and fix RuneCode source-quality violations with the narrowest safe change, then rerun the gate.
argument-hint: "[optional paths, rule names, or failing command output]"
disable-model-invocation: true
---

Use this workflow when `tools/checksourcequality`, `golangci-lint`, or related source-quality review feedback must be fixed.

## Standards and references to read first

- `agent-os/standards/global/source-quality-enforcement-layering.md`
- `agent-os/standards/global/source-quality-reviewed-exceptions.md`
- `agent-os/standards/global/source-quality-protected-surfaces.md`
- `agent-os/standards/global/language-aware-source-docs.md`
- `docs/source-quality.md`
- `.source-quality-baseline.json`
- `.source-quality-config.json`
- `.golangci.yml`
- `justfile`

## Procedure

1. Identify the failing source-quality surface:
   - `go run ./tools/checksourcequality`
   - `go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run`
   - review comments on source-quality policy, checker code, or protected surfaces
2. Classify each finding before editing:
   - missing or wrong docs/module docs
   - comment-quality false positive or true positive
   - function/file budget violation
   - Tier 1 suppression or reviewed-exception issue
   - protected-surface policy/config drift
3. Prefer the narrowest safe fix:
   - refactor code before raising limits
   - add or improve docs before widening exceptions
   - tighten checker heuristics instead of weakening policy when false positives appear
   - use checked-in baseline/config changes only when the exception is justified and reviewable
4. Keep exceptions disciplined:
   - never rely on reviewer memory or PR comments as the active exception mechanism
   - baseline/config changes must stay narrow, justified, and easy to audit
   - Tier 1 suppressions must not become casual inline escapes
5. Re-run verification after fixes:
   - targeted command(s) first
   - `just lint`
   - `just test`
6. Report:
   - what failed
   - what changed
   - whether any baseline/config/protected-surface files changed
   - final command results

## Guardrails

- Do not weaken thresholds, exclusions, or protected-surface rules casually.
- Prefer fixing code, docs, or heuristics before adding reviewed exceptions.
- If a change touches `.source-quality-baseline.json`, `.source-quality-config.json`, `.golangci.yml`, `justfile`, or `tools/checksourcequality/**`, call out that a protected surface changed.
- Do not commit or push unless explicitly requested.
