---
applyTo: "runecontext/project/**/*.md,runecontext/changes/**/*.md,runecontext/specs/**/*.md,runecontext/decisions/**/*.md,runecontext/standards/**/*.md"
---

Use these references for planning and roadmap review comments:

- `/runecontext/project/standards-inventory.md`
- `/runecontext/standards/product/roadmap-conventions.md`
- `/runecontext/project/roadmap.md`

When reviewing changes in this scope, focus on:

- Roadmap structure remains valid (`Upcoming Features`, `Unscheduled (Needs Specs)`, `Completed Features`).
- The roadmap remains a human-facing summary rather than the lifecycle source of truth.
- Upcoming and completed entries stay outcome-focused and do not reintroduce legacy `agent-os/specs/*` links as canonical roadmap links.
- Active lifecycle state stays in `runecontext/changes/*/status.yaml`, with durable completed outcomes in `runecontext/specs/*.md`.
- Standards inventory and bundle references stay accurate and concise.

Prefer comments that preserve traceability from roadmap items to RuneContext changes or specs when those canonical artifacts exist.
