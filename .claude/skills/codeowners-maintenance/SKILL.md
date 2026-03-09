---
name: codeowners-maintenance
description: Keep .github/CODEOWNERS aligned with new high-sensitivity directories and policy files.
argument-hint: "[optional paths to evaluate for ownership coverage]"
disable-model-invocation: true
---

Use this workflow when new directories or important files are added and CODEOWNERS coverage may be missing.

## References to read first

- `.github/CODEOWNERS`
- `.github/workflows/ci.yml`
- `CONTRIBUTING.md`
- `docs/trust-boundaries.md`

## Procedure

1. Identify candidate paths:
   - Paths provided by the user.
   - New or changed high-impact paths in the current branch.
2. Prioritize coverage for policy and trust-sensitive areas:
   - `.github/**` (workflows, templates, Copilot instruction files)
   - `.claude/**` (skills and commands that shape agent behavior)
   - `agent-os/standards/**` and `agent-os/product/**`
   - Security, trust-boundary, and bootstrap tooling files
3. Compare candidates against existing CODEOWNERS patterns.
4. Build a proposed update set:
   - Prefer root-anchored paths (leading `/`).
   - Prefer directory-level entries when ownership should apply to all descendants.
   - Use specific file entries when scope should stay narrow.
5. Confirm changes with the user before writing.
6. Update `.github/CODEOWNERS` with minimal, non-duplicative entries.
7. Re-check final matching behavior:
   - No accidental owner removal due to later overlapping patterns.
   - No duplicate patterns with conflicting owners unless intentional.
8. Report what was added and why.

## Guardrails

- Do not remove or change existing owners unless explicitly requested.
- Do not widen ownership scope unexpectedly with broad wildcards.
- Keep ownership rules stable and easy to scan.
- Do not commit or push unless explicitly requested.
