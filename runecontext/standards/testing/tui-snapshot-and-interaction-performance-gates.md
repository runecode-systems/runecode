---
schema_version: 1
id: testing/tui-snapshot-and-interaction-performance-gates
title: TUI Snapshot And Interaction Performance Gates
status: active
suggested_context_bundles:
    - ci-tooling
    - product-planning
---

# TUI Snapshot And Interaction Performance Gates

Use this standard when changing deterministic TUI snapshots, TUI visual review tooling, or TUI interaction-performance gates.

## Snapshot Build Surface

- Keep snapshot-only flags and scenario catalogs behind the `runecode_tui_snapshot` build tag.
- Keep default/release `runecode-tui` builds fail-closed for snapshot-only flags; `just tui-release-safety` must catch accidental exposure.
- Keep snapshot artifact generation deterministic for a fixed scenario, viewport, theme, and terminal size.
- Keep checked scenarios and bundles named explicitly; do not infer review scope from filenames when a manifest can carry the scope.

## Snapshot Artifacts

- Treat `manifest.json` as the authoritative inventory for generated scenarios, bundle, viewport, dimensions, route, artifact paths, and hashes.
- Emit `.ansi`, `.txt`, and `.svg` artifacts for each scenario; emit `.png` only when local ImageMagick `magick` or `convert` is available.
- Canonicalize output paths and constrain local snapshot output to trusted local roots before writing artifacts.
- Clean stale artifacts before each deterministic run unless the caller explicitly preserves them for inspection.
- Keep generated artifacts out of normal tracked source unless a later reviewed baseline policy explicitly introduces checked-in visual baselines.

## Review Workflow

- Keep `just tui-snapshot-review` non-GUI by default and suitable for terminal-only CI/dev environments.
- Keep GUI opening explicit through an `open` review mode or dedicated recipe.
- Keep route, overlay, compact, and mobile review bundles available so visual dogfooding does not cover only the default desktop route set.
- Preserve snapshot coverage for command palette, session switcher, leader help, quit confirmation, sidebar/focus states, rendered/structured/raw route modes, degraded states, setup flows, and evidence routes.

## Interaction Performance Gates

- Keep `just tui-perf` focused on deterministic, fixed-dataset TUI hot paths rather than broad end-to-end workflows.
- Keep `just ci-fast` running the focused TUI interaction benchmark lane so obvious hot-path regressions fail before merge.
- Keep required shared-Linux TUI perf contracts in `just ci-required-shared-linux` and `tools/perfcontracts/`; do not silently loosen thresholds or rewrite baselines from CI.
- Keep timing boundaries explicit: distinguish fresh-process attach latency, key-response latency, update/filter latency, render cost, idle CPU, and benchmark parser outputs.
- Keep perf diagnostics sanitized; do not leak sensitive local paths, tokens, raw broker startup output, or secret-like values in failures.
- Distinguish local noisy perf runs from deterministic CI failures, but fix stale markers, stale fixtures, and broken harness assumptions immediately.

## Maintenance

- Update snapshot expected scenario inventories when adding, renaming, or removing scenario names.
- Update perf harness markers when stable shell chrome changes.
- Update README command summaries when adding or retiring public `just tui-*` recipes.
- Cross-reference `runecontext/standards/product/tui-bubbletea-lipgloss-performance.md` for Bubble Tea/Lip Gloss implementation guidance.
