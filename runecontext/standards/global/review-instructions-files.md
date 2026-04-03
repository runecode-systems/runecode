---
schema_version: 1
id: global/review-instructions-files
title: Scoped Review Instruction Files
status: active
aliases:
    - agent-os/standards/global/review-instructions-files
---

# Scoped Review Instruction Files

Applies to: `.github/instructions/*.instructions.md`

- These files guide humans + automated reviewers; keep them concise and path-cited
- Include YAML frontmatter with `applyTo: "..."`
- Use a consistent structure:
  - "Use these references" (source-of-truth paths)
  - "When reviewing... focus on" bullets
  - Escalation note (what to raise, when)
- When scopes overlap, keep detailed policy in one dedicated instruction file.
- Nearby scoped instruction files should point to that file instead of restating the same policy.
- Keep overlap intentional; avoid competing `applyTo` ownership for the same detailed guidance.

```md
---
applyTo: "runner/**/*.{ts,js,json}"
---

Use these references:
- `/docs/trust-boundaries.md`

When reviewing changes in this scope, focus on:
- ...
```

Example:
- `source-quality.instructions.md` holds detailed source-quality review logic
- Go, runner, and CI instruction files point to it instead of duplicating the full policy
