# Scoped Review Instruction Files

Applies to: `.github/instructions/*.instructions.md`

- These files guide humans + automated reviewers; keep them concise and path-cited
- Include YAML frontmatter with `applyTo: "..."`
- Use a consistent structure:
  - "Use these references" (source-of-truth paths)
  - "When reviewing... focus on" bullets
  - Escalation note (what to raise, when)

```md
---
applyTo: "runner/**/*.{ts,js,json}"
---

Use these references:
- `/docs/trust-boundaries.md`

When reviewing changes in this scope, focus on:
- ...
```
