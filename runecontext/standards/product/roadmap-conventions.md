---
schema_version: 1
id: product/roadmap-conventions
title: Product Roadmap Conventions
status: active
suggested_context_bundles:
    - project-core
    - product-planning
aliases:
    - agent-os/standards/product/roadmap-conventions
---

# Product Roadmap Conventions

Applies to: `runecontext/project/roadmap.md`

This roadmap is a human-facing product summary. Lifecycle state for active work lives in `runecontext/changes/*/status.yaml`, and durable completed outcomes live in `runecontext/specs/*.md`.

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

- Each roadmap entry should include:
  - Feature title
  - A short, user-visible description (1-2 lines)
  - Optional canonical RuneContext artifact references when they already exist

Template:

```md
- Feature Title
  - Short description of the user-visible outcome.
```

Rules:

- Do not use checkboxes as lifecycle state; active status belongs in `runecontext/changes/*/status.yaml`.
- Reference canonical RuneContext change or spec artifacts when they exist.
- Do not use `agent-os/specs/*` as canonical roadmap links.
- Keep descriptions outcome-focused (what changes for the user), not implementation notes.

## Moving Items On Release

- When a version is released:
  - Move the entire version block from `## Upcoming Features` to `## Completed Features`.
  - Update any durable spec references if they now exist.
  - Keep Completed ordered newest-first.

## Converting Unscheduled Items Into Specs

- If an item exists under `## Unscheduled (Needs Specs)` and a spec is created for it:
  - Replace the unscheduled item with a proper roadmap entry under the target version group.
  - Remove the duplicate unscheduled item.
