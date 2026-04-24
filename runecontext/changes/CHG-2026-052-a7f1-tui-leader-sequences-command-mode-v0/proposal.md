## Summary
Replace RuneCode TUI's shell-first plain-letter global shortcut model with an explicit power-user input system built around a configurable leader key, which-key-style sequence overlays, and a real `:` command mode, while preserving beginner-friendly visible navigation and preventing global shortcut interference with local route interaction.

## Problem
The current TUI shell relies primarily on direct single-key global shortcuts, with a temporary mitigation that moved a few conflicting globals to uppercase and added narrow text-entry bypasses. That stopgap reduced some collisions, but it did not fix the underlying input-model problem:

- shell power actions are still entered implicitly rather than intentionally
- the shell still handles many plain-letter global shortcuts before route-local interaction
- the current `IsTextEntryActive()`-style guard is too narrow for future local widgets, secret-entry flows, inline searches, wizards, and richer editors
- power-user affordances are not yet modeled as one coherent system across discoverable overlays, exact commands, and fuzzy command search
- beginner-friendly quitting and expert-friendly quitting are not yet unified in one intentional design

If the TUI keeps layering exceptions on top of plain-letter shell globals, RuneCode will continue to accumulate route-specific suppression logic, surprising key collisions, and avoidable friction for both beginners and power users.

## Proposed Change
- Retire plain-letter shell globals as power-user entry points and move shell power actions behind explicit entry flows.
- Introduce a configurable shell leader key with `space` as the default.
- Add which-key-style shell overlays for leader-driven multi-key sequences so each intermediate key narrows the next valid choices.
- Upgrade `:` into a real command mode rendered in the bottom-left command-entry area of the TUI rather than as a centered palette overlay.
- Keep `ctrl+p` or equivalent fuzzy command-surface entry available for object and command discovery, but back it with the same command graph used by leader sequences and command mode.
- Add a richer route-to-shell keyboard-ownership contract so local route interaction can explicitly retain control during text entry or stronger local capture states.
- Preserve beginner accessibility by adding a visible `Quit RuneCode` action in the shell navigation/action surface instead of forcing users to know leader or `:` flows.
- Keep `ctrl+c` as an always-available emergency terminal quit path, but require two consecutive `ctrl+c` presses to exit so users do not accidentally quit when they intend to copy text.
- Confirm quit only when there is active compose, secret entry, command entry, or comparable local entry state; otherwise, quit actions may complete immediately.

## Why Now
This work belongs in `v0.1.0-alpha.7` alongside session-execution maturation because the TUI already has enough shell, command-surface, text-area, and persistence foundation that the shortcut stopgap should be replaced before more local interaction modes are added.

Landing this now avoids cementing the current uppercase-global-key workaround into the long-term product and gives the TUI one durable input model before more route-local flows, richer local widgets, and more advanced operator workflows arrive.

## Assumptions
- `runecontext/changes/*` is the canonical planning surface for this repository.
- RuneCode owns the user-facing TUI command surface while remaining a strict client of broker-owned contracts.
- TUI-local preferences such as leader-key selection remain local-only convenience state rather than broker-owned or repository-owned truth.
- `CHG-2026-013-d2c9-minimal-tui-v0` and `CHG-2026-037-91be-tui-multi-session-power-workspace-v0` remain the shell and workbench foundations this change builds on rather than replaces.
- The command registry, command palette, and object-aware navigation surface are durable shell infrastructure that should be reused rather than forked into independent shortcut systems.
- Keyboard behavior must preserve beginner usability alongside power-user speed; RuneCode should not copy NeoVim's friction around discoverability or quitting even if it borrows the leader and `:` concepts.
- `tab` and `shift+tab` remain the default focus-traversal posture unless a local widget explicitly owns them in a stronger local-capture state.

## Out of Scope
- Replacing Bubble Tea, Lip Gloss, the root shell architecture, or the shell-owned command registry.
- Making leader-key configuration broker-owned, repository-owned, or shared team state.
- Turning `Quit RuneCode` into a fake navigable route rather than a shell action.
- Removing fuzzy command/palette discovery in favor of leader-only or command-line-only navigation.
- Adding a full settings route in the first cut if command-mode configuration is sufficient for the leader-key preference.
- Reintroducing plain-letter global power shortcuts as the long-term interaction model.

## Impact
This change replaces the TUI shortcut stopgap with a durable shell input architecture that explicitly separates power-user command entry from ordinary local interaction.

It also freezes the following product-level clarifications for future TUI work:
- shell power actions require explicit entry through leader, `:`, or discoverable command surfaces rather than ambient plain-letter globals
- local route widgets may claim stronger keyboard ownership than the shell can express today
- the command graph behind leader sequences, `:` command mode, help text, and fuzzy command search should stay unified
- quitting remains easy for beginners, but accidental exit from `ctrl+c` should be harder than it is today
