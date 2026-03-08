# Product Roadmap Conventions

Applies to: `agent-os/product/roadmap.md`

This roadmap is the canonical view of what is planned next (as specs) and what has shipped (as releases).

## Structure

- Keep a very short intro at the top (2-5 lines) explaining how to read/maintain the roadmap.
- Required sections (in this order):
  - `## Upcoming Features`
  - `## Unscheduled (Needs Specs)`
  - `## Completed Features`

## Version Grouping

- Group roadmap items under version headings one level below the section heading (H3).
- Use `### vNext (Planned)` for work that is planned but not yet assigned a concrete version.

## Spec Entry Format

- Each spec entry is a checkbox with:
  - Spec title
  - Spec folder path (in backticks)
  - A short, user-visible description (1-2 lines)

Template:

```md
- [ ] Spec Title (`agent-os/specs/YYYY-MM-DD-HHMM-spec-slug/`)
  - Short description of the user-visible outcome.
```

Rules:

- Upcoming work uses `- [ ]`.
- Completed work uses `- [x]`.
- Reference specs by title + spec folder path (do not use numeric spec IDs).
- Keep descriptions outcome-focused (what changes for the user), not implementation notes.

## Moving Items On Release

- When a version is released:
  - Mark all items in that version block as `- [x]`.
  - Move the entire version block from `## Upcoming Features` to `## Completed Features`.
  - Keep Completed ordered newest-first.

## Converting Unscheduled Items Into Specs

- If an item exists under `## Unscheduled (Needs Specs)` and a spec is created for it:
  - Replace the unscheduled item with a proper spec entry under the target version group.
  - Remove the duplicate unscheduled checkbox.
