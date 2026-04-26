# Verification

## Executed Automated Checks
- `go test ./cmd/runecode-tui`
- `just ci`

## Automated Results
- `go test ./cmd/runecode-tui`: pass after foundation refactor and follow-up review fixes.
- `just ci`: pass, including gofmt/lint/vet/source-quality/model-checking/full Go test suite/runner checks/boundary-check.

## Review Lanes
- `review-code-correctness`: found input/history and command-side-effect issues; fixes applied and reverified.
- `review-code-integration`: no findings.
- `review-code-security`: found UI/clipboard redaction and defensive validation gaps; fixes applied and reverified.

## Interactive Workbench Checks
- Verify the TUI launches in full-screen alt-screen mode and exits cleanly.
- Verify the sidebar is visible by default, can be toggled off and back on, and palette-only navigation remains fully capable when it is hidden.
- Verify wide, medium, and narrow terminal behaviors match the shell-level breakpoint model.
- Verify `open`, `inspect`, `jump`, and `back` behave consistently across sessions, runs, approvals, artifacts, and audit records.
- Verify Live Activity and Action Center surfaces are driven by typed watch-backed updates rather than by log scraping.
- Verify selection mode allows ordinary drag-to-select copying from transcripts, diffs, logs, and inspectors.
- Verify explicit copy actions work for canonical IDs, digests, raw blocks, transcript excerpts, artifact previews, and linked references.
- Verify compose input supports multiline paste and bracketed paste without losing text.
- Verify the shell-level running indicator appears when canonical work is active and that `loading`, `running`, and `degraded sync` are distinguishable.
- Verify persisted local workbench state restores sidebar visibility, pane state, presentation mode, theme preset, recents, pinned sessions, and last active session per workspace without affecting canonical system truth.
- Capture repeatable screenshots or VHS tapes for the key shell, palette, selection-mode, Action Center, and inspector flows.

Interactive coverage note:
- The branch includes route and shell tests for the major shell semantics above, plus capture assets under `cmd/runecode-tui/capture/`. Manual terminal-flow revalidation should still be performed during release promotion or dogfooding when a live broker is available.

## Verification Notes
- Confirm the change is scheduled pre-MVP and after `CHG-2026-013-d2c9-minimal-tui-v0`, in `v0.1.0-alpha.5`.
- Confirm the change extends the MVP TUI foundation rather than replacing or superseding its trust model.
- Confirm the shell, not the routes, owns the top status bar, optional sidebar, main pane, optional inspector, bottom strip, overlays, and breakpoint behavior.
- Confirm `workspace` means canonical broker workspace identity and `workbench layout` means local saved UI arrangement.
- Confirm multi-session behavior is framed around canonical session identity rather than client-local tabs.
- Confirm the minimum session directory metadata is available for quick switching and background awareness.
- Confirm long-lived watch streams and object-summary caches are shell-owned workbench infrastructure rather than duplicated route state.
- Confirm advanced live activity depends on typed watch/event families rather than log scraping alone.
- Confirm richer inspection continues to use broker-owned read models and drill-down APIs rather than daemon-private files or storage layouts.
- Confirm Action Center v0 families are approvals, operational attention, and blocked work impact, and that future question queues remain reserved until a canonical broker model exists.
- Confirm question or pending-answer integration is conditional on a canonical broker object model and is not invented locally by the TUI.
- Confirm saved layouts, presets, recents, pinned sessions, and workspace UI state are treated as convenience state, not trusted control-plane state.
- Confirm copy and paste support both terminal selection and explicit in-app actions without sacrificing one for the other.
- Confirm theme presets are built on semantic tokens and preserve non-color cues.
- Confirm remote or scaled backend compatibility is preserved at the logical-contract level without introducing remote transport changes in this change.
- Confirm raw model chain-of-thought remains out of scope.

## Implemented Foundation Areas
- shell-owned pane composition, overlays, responsive breakpoints, breadcrumbs/history, and status surfaces
- canonical multi-session workspace and shell-owned object index for palette discovery
- shell-owned navigation semantics for `open`, `inspect`, `jump`, and `back`
- shared inspector and persistent long-form document/viewport model across session/run/approval/artifact/audit surfaces
- shell-owned watch manager with typed family reduction, health projection, and live activity semantics, including explicit waiting-state rendering that avoids the fast running animation path
- copy/selection/OSC52-aware clipboard support with defensive UI and clipboard redaction paths
- local-only persisted workbench layout/theme/session convenience state keyed by logical broker target

## Close Gate
Use the repository's standard verification flow before closing this change.
