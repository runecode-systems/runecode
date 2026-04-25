---
schema_version: 1
id: product/tui-shell-input-and-command-surfaces
title: TUI Shell Input And Command Surfaces
status: active
suggested_context_bundles:
    - project-core
    - product-planning
---

# TUI Shell Input And Command Surfaces

When extending `runecode-tui` shell interaction behavior:

- Keep route keyboard-ownership state authoritative for whether the shell may intercept typing, power actions, and focus traversal.
- Use explicit shell entry surfaces for power actions: leader sequences, `:` command mode, fuzzy discovery, or visible shell actions.
- Do not reintroduce ambient plain-letter shell globals that can collide with route-local typing, compose buffers, or secret entry.
- Keep help, leader bindings, command aliases, visible discoverability, and fuzzy command discovery generated from one authoritative action definition graph rather than hand-maintained parallel lists.
- Keep command-mode rendering shell-owned and visibly distinct from centered overlay discovery surfaces; `:` is a command-entry surface, not a second way to open the palette.
- Keep quit modeled as a shell action rather than inventing a fake route or route-local escape path.
- Preserve a visible beginner-friendly quit path outside leader and command fluency so first-time users do not need to learn hidden power shortcuts before they can leave the product.
- Treat emergency quit as distinct from normal quit: `ctrl+c` may remain an escape hatch, but normal quit confirmation and beginner discoverability should flow through the real shell action surfaces.
- Clear pending emergency-quit state as soon as ordinary interaction resumes; emergency arming must not linger across unrelated normal use.
- Keep client-local leader-key preferences and similar shell convenience state explicitly local-only and non-authoritative.
