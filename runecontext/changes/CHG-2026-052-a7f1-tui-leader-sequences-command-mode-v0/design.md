# Design

## Overview
Replace the shell's ambient plain-letter shortcut model with an explicit input-mode architecture centered on leader sequences, bottom-left command mode, and stronger route-local keyboard ownership.

## Key Decisions
- Plain-letter shell globals used as power-user entry points should be retired rather than expanded.
- Shell power actions should require explicit entry through one of three shell-owned surfaces:
  - leader sequence mode
  - `:` command mode
  - fuzzy command/palette mode
- The default leader key is `space`, but the leader key must be configurable by the user.
- Leader-key configuration is local-only convenience state and belongs in the existing local workbench preference layer.
- A which-key-style overlay should appear immediately when leader mode is entered and update after each sequence step.
- `:` command mode should render in the bottom-left command-entry area of the TUI rather than as a centered overlay.
- `Enter` executes command-mode input, and `Esc` aborts command mode without side effects.
- `ctrl+p` may remain a fuzzy command/object discovery entry point, but it should reuse the same underlying command/action graph as leader and `:`.
- Beginner-friendly quitting must remain visible and discoverable even after shell power actions move behind leader and `:`.
- `Quit RuneCode` should be exposed as a visible shell navigation/action entry, not as a fake route.
- `ctrl+c` remains an emergency terminal quit path, but it requires two consecutive presses to exit.
- The first `ctrl+c` should not quit. It should instead populate the bottom-left command-entry area with a clear message instructing the user to press `ctrl+c` once more to quit.
- Quit confirmation should appear only when there is active compose, secret entry, command entry, or comparable local entry state.
- `tab` and `shift+tab` remain the default shell focus traversal because they are intuitive and already aligned with standard terminal-app behavior.
- Local widgets may override `tab` and `shift+tab` when they enter a stronger keyboard-capture state.

## Input Ownership Model

### Problem With The Current Model
The current shell mostly knows whether a route is in text entry or not through a boolean `IsTextEntryActive()` shape. That is not expressive enough for the product RuneCode is becoming.

The shell now needs to distinguish between ordinary route interaction and stronger local keyboard claims so that leader mode, command mode, focus traversal, and emergency quit behavior can stay predictable.

### Keyboard Ownership States
Replace the narrow boolean guard with a richer route-to-shell keyboard-ownership contract that can express at least:

- `normal`
  - route-local keys and shell-global explicit-entry keys are both allowed
  - leader, `:`, palette entry, and ordinary shell focus traversal may still work
- `text_entry`
  - text typing belongs to the local widget
  - plain text and punctuation should not trigger shell power actions
  - shell may still allow explicitly reserved non-text escape hatches where appropriate
- `exclusive_local_capture`
  - the local widget owns all ordinary keyboard input, including keys such as `tab` if needed
  - shell-level power-entry keys should not preempt the local widget except for explicitly reserved emergency behavior such as the first `ctrl+c` warning and the second `ctrl+c` quit path

Examples that should map cleanly onto this model:
- chat composer text area
- provider secret entry
- future inline filter/query boxes
- future route-local wizards and editors
- future command-mode entry itself

### Shell Reserved Keys
After this change, the shell should reserve only a very small direct key set outside explicit power-entry surfaces. In `v0`, that reserved set should be narrow and reviewable:

- `ctrl+c` emergency quit sequence
- overlay dismissal keys when an overlay is already open
- `tab` and `shift+tab` focus traversal only when the local keyboard-ownership state allows it

The shell should not continue to reserve plain-letter globals such as `q`, `b`, `y`, `t`, `j`, `k`, or digits as ambient power-user entry points.

## Leader Sequence Model

### Entry And Lifecycle
- Pressing the configured leader key enters leader mode.
- The shell opens a which-key-style overlay immediately.
- The overlay shows the currently valid next keys, grouped by semantic family.
- Each next key either:
  - executes an action directly
  - narrows into another submap
  - aborts with a clear non-destructive message if the key is invalid in the current sequence
- `Esc` aborts leader mode.
- The first cut should avoid timeout-driven execution; leader mode remains active until a valid completion, explicit abort, or invalid-key abort.

### Configurable Leader Key
- Default: `space`
- Initial allowed configured values should be deliberately limited to a reviewed set of low-risk keys, for example:
  - `space`
  - `comma`
  - `backslash`
- Configuration must be:
  - easy to set from command mode
  - persisted in local-only workbench state
  - validated against unsafe choices such as `enter`, `esc`, or `ctrl+c`
- The command-mode surface should support at least:
  - `:set leader space`
  - `:set leader comma`
  - `:set leader backslash`
  - `:set leader default`

### Sequence Families
The first sequence families should be mnemonic, discoverable, and stable enough for help generation. A recommended initial shape:

- `s`: search and discovery
- `o`: open or jump
- `w`: workbench/window/layout actions
- `c`: copy actions
- `a`: approvals/action center/operator attention
- `q`: quit/back lifecycle actions

The specific leaf mappings may evolve during implementation, but the durable rule is that the leader tree must be generated from the real action graph rather than hand-maintained separately from commands or help text.

### Focus Movement Under Leader
Power-user focus movement may be added under leader without replacing `tab` or `shift+tab`. Recommended future-friendly examples:

- `<leader>w n`: next pane
- `<leader>w p`: previous pane
- `<leader>w h/j/k/l`: directional focus moves when layout permits

This gives power users mnemonic focus control without forcing Vim-style pane movement as the only default.

## Command Mode Model

### Presentation
- Pressing `:` enters command mode.
- The active command line should appear in the bottom-left command-entry area of the TUI, in the same shell-owned region that surfaces typed command text.
- While command mode is active, that command-entry area temporarily takes precedence over ordinary route hints or bottom-strip guidance.

### Behavior
- Typed text is appended to the active command line.
- `Backspace` deletes.
- `Enter` parses and executes the command.
- `Esc` aborts command mode and clears the in-progress command text.
- Parse errors and execution errors should remain visible in the same bottom-left command-entry area long enough for the user to understand what failed.

### Command Surface
The first cut should support both canonical names and short aliases. At minimum, the command graph should support patterns such as:

- `:q`
- `:quit`
- `:open chat`
- `:open runs`
- `:sidebar toggle`
- `:theme cycle`
- `:set leader space`

The command-line parser should remain shell-owned and command-registry-backed rather than allowing routes to invent unrelated parsing rules.

## Unified Action Graph

### One Authoritative Graph
Leader sequences, `:` command mode, fuzzy command search, help rendering, and discoverability overlays should all be generated from one authoritative shell action graph.

That graph should carry enough metadata to support:
- command IDs
- user-facing titles and descriptions
- command aliases for `:` mode
- leader key paths
- visibility in fuzzy search
- help and which-key display text
- whether the action is shell-wide, route-sensitive, or conditionally available

### Why This Matters
This avoids the current and future risk that:
- leader mappings drift from command aliases
- help text drifts from actual behavior
- discoverability overlays show options that cannot execute
- power-user flows and visible beginner flows diverge into multiple unreviewed command systems

## Beginner-Friendly Quit Model

### Visible Quit Action
The shell must expose a visible `Quit RuneCode` action for non-power users.

That action should be represented as a shell action in visible navigation or an equivalent always-discoverable shell action area, not as a route.

Durable rule:
- quitting is an action, not a place
- the visible action should remain available even if leader and command mode exist

### Quit Entry Paths
By the end of this change, quit should be available through:
- visible `Quit RuneCode` navigation/action entry
- leader flow under a quit family
- `:q` and `:quit`
- fuzzy command/palette search
- double-press `ctrl+c` emergency exit

### `ctrl+c` Double-Press Flow
To reduce accidental exits when users attempt text-copy flows:

- first `ctrl+c`
  - do not quit
  - surface a bottom-left command-entry message such as `Press Ctrl+C again to quit RuneCode.`
  - arm a short-lived pending emergency-quit state
- second `ctrl+c` while that pending state is active
  - quit immediately
- if the second `ctrl+c` does not arrive within the configured short window or the user continues normal interaction
  - clear the pending emergency-quit state

The exact timeout value may be implementation detail, but the presence of a short-lived two-step emergency quit flow is part of the product behavior this change should freeze.

### Quit Confirmation With Active Entry State
If there is active compose, secret entry, command entry, or comparable local entry state:

- visible quit actions, leader quit actions, and `:q` should prompt for confirmation
- the prompt should make it clear that in-progress local input may be discarded
- `ctrl+c` emergency quit remains an emergency escape hatch and may bypass the richer confirmation prompt after the second press

If there is no active local entry state, visible quit and command/leader quit paths may complete without an extra confirmation prompt.

## Focus Traversal Model

### Default Traversal
- `tab`: next focus area
- `shift+tab`: previous focus area

These remain the default because they are intuitive, already aligned with the current TUI, and more discoverable for general terminal users than directional Vim-style pane movement.

### Interaction With Local Widgets
- in `normal`, shell focus traversal may run normally
- in `text_entry`, local widgets may accept `tab` when they need it; otherwise, shell traversal may still work
- in `exclusive_local_capture`, local widgets fully own `tab` and `shift+tab`

This preserves intuitive shell traversal while leaving room for richer local editing widgets later.

## Persistence Model

### Local Preference Storage
Leader-key configuration should be persisted in the same local-only workbench state family used for other user-local TUI preferences such as theme and layout.

This means:
- scoped by logical broker target or equivalent local TUI target identity
- not broker-owned
- not repository-owned
- easy to change and restore

### Ease Of Configuration
The first cut must make leader configuration easy enough that users do not need to edit files manually. Command-mode configuration is sufficient for `v0` if it is:
- documented in help or discoverability surfaces
- validated
- persisted automatically
- reversible through an obvious reset path

## Help And Discoverability
- help output must be generated from the real action graph and real key definitions rather than from hand-maintained strings
- which-key overlays must reflect the real currently valid next keys
- command-mode help and fuzzy-search results must reflect the same canonical action metadata
- visible navigation and visible quit affordances must remain intact so the product does not regress into a power-user-only shell

## Foundation Shortcuts To Avoid
- do not preserve plain-letter global power actions as the long-term shell model
- do not solve collisions by adding more uppercase shell globals
- do not keep text-entry suppression as a narrow boolean forever
- do not make quit discoverable only through leader or `:`
- do not store leader configuration in broker state, repo config, or canonical product policy
- do not let leader mappings, command aliases, fuzzy-search actions, and help text drift into separate definitions
- do not make Vim-style pane movement the only intuitive focus traversal posture

## Main Workstreams
- keyboard-ownership contract upgrade
- leader sequence and which-key overlay foundation
- bottom-left command mode foundation
- unified command/action graph and help generation
- visible quit action and emergency quit redesign
- local preference persistence for leader configuration
