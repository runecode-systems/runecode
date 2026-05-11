---
schema_version: 1
id: product/tui-bubbletea-lipgloss-performance
title: TUI Bubble Tea And Lip Gloss Performance
status: active
suggested_context_bundles:
    - product-planning
    - go-control-plane
---

# TUI Bubble Tea And Lip Gloss Performance

Use this standard when changing `cmd/runecode-tui` shell, route, overlay, input, rendering, or style code.

## Bubble Tea Event Loop

- Treat `Update` and `View` as serialized hot paths: any synchronous filtering, route-surface generation, layout, render, broker call, file write, or network call can directly increase key-to-render latency.
- Keep all blocking broker, file, and network work in `tea.Cmd`; route `Update` methods should update state and return commands, not perform blocking work inline.
- Keep `View` deterministic and side-effect free so snapshots, benchmarks, and no-dropped-key tests remain meaningful.
- Keep route activation, palette actions, reference jumps, and mutation actions typed and non-destructive until the user executes an explicit action.
- Keep async watch/polling honest: ticks, spinners, and reload commands may reflect real in-flight commands or broker-owned watch state, but must not imply client-local optimistic completion.
- Keep keyboard ownership explicit for text entry, secret entry, overlays, command mode, normal route navigation, selection mode, quit confirmation, and emergency quit.
- Keep resize handling centralized so route viewport sizing stays predictable across desktop, compact, and mobile terminal widths.
- Keep mouse capture optional and reversible so terminal text selection remains possible.

## Input-Path Performance

- Precompute normalized searchable text for command palette, session switcher, and object indexes before hot typing paths.
- Reuse match buffers and avoid repeated lowercase/concatenate/allocate scans on every keypress.
- Version asynchronous search/filter requests and reject stale results when query, entry version, or normalized search needle no longer matches current state.
- Render only the visible window of overlay rows plus gap markers; do not format every match before bounded-window trimming.
- Avoid repeated route `ShellSurface`, layout, overlay-height, and final `View` work for one keypress unless the underlying state changed.
- Reuse cached overlay background frames while typing in command palette, session switcher, or leader help, and explicitly invalidate the cache when route state, watch/session/object-index state, theme, focus, overlay identity, or viewport size changes.
- Keep local convenience persistence off critical input paths; debounce or asynchronously write layout, theme, recent-session, and leader-key preferences.
- Add regression coverage for fast scripted overlay input to prove characters are consumed in order without dropped input.

## Bubbles Usage

- Use `textarea` or equivalent maintained primitives for multiline composer behavior instead of ad-hoc line editing.
- Use `viewport` or bounded long-form primitives for transcripts, diffs, logs, artifact content, and audit records rather than clipping raw strings manually.
- Use `help`/`key` or one generated action graph for concise help where practical; keep exhaustive discovery in command palette, leader help, or dedicated help surfaces.
- Use `spinner` or small activity indicators only for real in-flight commands or watch activity.

## Lip Gloss Rendering

- Build from shared design tokens for base surface, elevated panel surface, overlay surface, foreground, dimmed text, links/evidence, code/digest text, selected rows, borders, and state colors.
- Prefer composable card, rail, row, pill, command-palette, modal-panel, detail-viewport, setup-stepper, and evidence-trail components over per-route string dumps.
- Budget widths and heights before rendering; prevent panes and overlays from overflowing or collapsing unpredictably.
- Use `lipgloss.Width`, ANSI-aware truncation, and display-cell-aware clipping for styled, wide-character, and colored content; do not use raw rune-count clipping where alignment matters.
- Prefer padding, accent rails, muted dividers, and selected-row backgrounds over excessive full borders.
- Make modal surfaces visibly modal within terminal constraints: centered panel, bounded height, clear escape hint, strong selected row, and no duplicate footer noise.
- Verify dark, dusk, and high-contrast themes; selected rows, warnings, destructive actions, and reduced-assurance states must remain legible without relying on hue alone.

## Verification

- Keep focused benchmarks for palette typing, session-switcher typing, leader open/step flows, overlay rendering, shell focus traversal, watch projection, and shell `View` costs.
- Keep no-dropped-key tests for fast scripted overlay entry.
- Keep perf harness startup and key-response markers aligned with stable current shell chrome.
- Treat interaction responsiveness as product correctness, not optional cleanup.
