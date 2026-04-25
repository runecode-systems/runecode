# Tasks

## Phase 1: Keyboard Ownership Foundation

- [x] Replace the narrow route text-entry boolean guard with a richer route-to-shell keyboard-ownership contract.
- [x] Define at least `normal`, `text_entry`, and `exclusive_local_capture` keyboard-ownership states.
- [x] Ensure the shell checks keyboard ownership before opening leader mode, command mode, or running focus traversal.
- [x] Keep route-local widgets authoritative for ordinary typing in `text_entry` and broader key handling in `exclusive_local_capture`.
- [x] Update existing compose and secret-entry flows to use the richer ownership model rather than ad hoc suppression logic.

## Phase 2: Retire Plain-Letter Shell Globals

- [x] Remove plain-letter shell globals as power-user entry points.
- [x] Retire ambient shell power shortcuts such as direct plain-letter quit, back, copy, theme, and similar non-entry actions.
- [x] Keep only a narrow, reviewable set of directly reserved shell keys outside explicit power-entry surfaces.
- [x] Preserve compatibility for shell overlays and focus traversal only where the new ownership model allows it.

## Phase 3: Leader Sequence Foundation

- [x] Add a configurable shell leader key with `space` as the default.
- [x] Validate configured leader keys against a reviewed allowlist and reject unsafe options such as `enter`, `esc`, and `ctrl+c`.
- [x] Add immediate leader-mode entry without timeout-based auto-execution in the first cut.
- [x] Add a which-key-style overlay that appears as soon as leader mode starts.
- [x] Update the which-key overlay after each sequence step to show only the currently valid next choices.
- [x] Support abort via `esc` and clean invalid-key abort behavior with a clear user-visible message.

## Phase 4: Command Mode Foundation

- [x] Upgrade `:` from a palette opener into a real shell-owned command mode.
- [x] Render command-mode input in the bottom-left command-entry area of the TUI.
- [x] Support command editing, `enter` execution, and `esc` abort.
- [x] Surface parse and execution errors in the same command-entry area.
- [x] Keep command-mode rendering shell-owned rather than route-specific.

## Phase 5: Unified Action Graph

- [x] Refactor shell commands, leader mappings, command-mode aliases, fuzzy search, and help generation onto one authoritative action graph.
- [x] Ensure leader overlays, command-mode aliases, and fuzzy results are generated from the same real definitions.
- [x] Keep help and discoverability text generated from the real action metadata rather than hand-maintained strings.
- [x] Preserve the existing command registry where possible rather than replacing it with a second unrelated system.

## Phase 6: Initial Leader And Command Coverage

- [x] Add initial leader groups for search/discovery, open/jump, workbench/window actions, copy actions, approvals/action-center flows, and quit/back actions.
- [x] Add command-mode support for canonical commands and aliases such as `:q`, `:quit`, `:open ...`, `:sidebar toggle`, `:theme cycle`, and `:set leader ...`.
- [x] Keep `ctrl+p` or equivalent fuzzy command discovery available and backed by the same action graph.
- [x] Ensure route-sensitive actions remain available through the unified command system without letting shell-local power flows preempt local route typing.

## Phase 7: Focus Traversal Clarification

- [x] Preserve `tab` and `shift+tab` as default focus traversal in ordinary shell interaction.
- [x] Allow local widgets to own `tab` and `shift+tab` when their keyboard-ownership state requires it.
- [x] Add optional power-user focus movement under leader-managed workbench actions without replacing `tab` defaults.

## Phase 8: Beginner-Friendly Quit

- [x] Add a visible `Quit RuneCode` shell action in navigation or an equivalent always-discoverable shell action surface.
- [x] Ensure visible quit remains available even when users never use leader or command mode.
- [x] Keep quit modeled as an action rather than inventing a fake route.
- [x] Expose quit through leader, command mode, fuzzy command search, and visible navigation/action surfaces.

## Phase 9: Emergency Quit Redesign

- [x] Keep `ctrl+c` as an emergency terminal quit path.
- [x] Require a second consecutive `ctrl+c` press before quitting.
- [x] On the first `ctrl+c`, render a bottom-left command-entry message instructing the user to press `ctrl+c` once more to quit.
- [x] Keep the first `ctrl+c` from terminating the process.
- [x] Clear the pending emergency-quit state after a short-lived timeout or when equivalent normal interaction resumes.

## Phase 10: Quit Confirmation With Active Entry State

- [x] Detect active compose, secret entry, command entry, and comparable local-entry states before non-emergency quit flows complete.
- [x] Prompt for confirmation only when active local entry state would be discarded.
- [x] Allow normal quit flows to complete immediately when no active local entry state exists.
- [x] Keep the double-press `ctrl+c` path as the emergency escape hatch even when richer confirmation exists.

## Phase 11: Preference Persistence And Configuration

- [x] Persist the configured leader key in the local-only workbench preference store.
- [x] Scope the preference to the existing logical local TUI target or equivalent local scope.
- [x] Support command-mode configuration and reset flows for leader selection.
- [x] Make configuration easy enough that users do not need to edit files manually for the initial cut.

## Phase 12: Verification And Regression Coverage

- [x] Replace tests that encode the uppercase-global-shortcut stopgap with tests for the new input-mode rules.
- [x] Add tests proving local text entry does not trigger shell power actions.
- [x] Add tests proving provider secret entry and similar exclusive local-capture flows suppress shell power-entry handling.
- [x] Add tests covering leader-mode entry, overlay progression, abort behavior, and invalid-key handling.
- [x] Add tests covering bottom-left command-mode rendering, parsing, execution, and abort.
- [x] Add tests covering visible `Quit RuneCode` behavior, quit confirmation with active local entry state, and double-press `ctrl+c` emergency quit.
- [x] Add tests proving leader configuration persists locally and rejects invalid values.

## Acceptance Criteria

- [x] Plain-letter shell globals no longer act as the default power-user command model.
- [x] The default leader key is `space`, and users can reconfigure it through an easy local preference flow.
- [x] Leader mode shows a which-key-style overlay and narrows choices after each key press.
- [x] `:` opens a real bottom-left command mode, `enter` executes, and `esc` aborts.
- [x] Help, leader mappings, command aliases, and fuzzy search are generated from one authoritative action graph.
- [x] Local route typing and secret entry no longer collide with shell power-user actions.
- [x] `tab` and `shift+tab` remain intuitive default focus traversal except when stronger local capture states claim them.
- [x] Beginners can quit RuneCode through a visible shell action without learning leader or `:`.
- [x] `ctrl+c` requires two presses to quit, and the first press warns in the command-entry area.
- [x] Non-emergency quit confirmation appears only when active local entry state would be discarded.
