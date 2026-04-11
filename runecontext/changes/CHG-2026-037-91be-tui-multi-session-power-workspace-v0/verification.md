# Verification

## Planned Checks
- `runectx validate --json`
- `runectx status --json`
- `just test`

## Verification Notes
- Confirm the change is scheduled pre-MVP and after `CHG-2026-013-d2c9-minimal-tui-v0`, in `v0.1.0-beta.1`.
- Confirm the change extends the MVP TUI foundation rather than replacing or superseding its trust model.
- Confirm multi-session and workspace behavior is framed around canonical session identity rather than client-local tabs.
- Confirm advanced live activity depends on typed watch/event families rather than log scraping alone.
- Confirm richer inspection continues to use broker-owned read models and drill-down APIs rather than daemon-private files or storage layouts.
- Confirm question/pending-answer integration is conditional on a canonical broker object model and is not invented locally by the TUI.
- Confirm saved layouts, presets, and workspace UI state are treated as convenience state, not trusted control-plane state.
- Confirm theme presets are built on semantic tokens and preserve non-color cues.
- Confirm remote/scaled backend compatibility is preserved at the logical-contract level without introducing remote transport changes in this change.
- Confirm raw model chain-of-thought remains out of scope.

## Close Gate
Use the repository's standard verification flow before closing this change.
