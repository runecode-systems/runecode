# Verification

## Executed Automated Checks
- `go test ./cmd/runecode-tui`
- `just ci`

## Planned Automated Checks
- `go test ./cmd/runecode-tui`
- `just ci`

## Interactive Workbench Checks
- Verify leader mode opens from the configured leader key and shows a which-key-style overlay immediately.
- Verify the overlay updates after each leader key step and shows only valid next actions.
- Verify invalid leader keys abort cleanly without executing unrelated actions.
- Verify `:` opens a bottom-left command line rather than a centered palette overlay.
- Verify typed command text appears in the bottom-left command-entry area, `enter` executes, and `esc` aborts.
- Verify `ctrl+p` or equivalent fuzzy command discovery remains available and draws from the same action graph as leader and `:`.
- Verify plain typing in the chat composer does not trigger shell power actions.
- Verify plain typing during provider secret entry does not trigger shell power actions.
- Verify route-local widgets can retain `tab` ownership when in stronger local-capture states.
- Verify `tab` and `shift+tab` continue to traverse focus normally in ordinary shell interaction.
- Verify a visible `Quit RuneCode` action exists for beginners outside leader and `:` flows.
- Verify visible quit, leader quit, and `:q` only ask for confirmation when active local entry state would be discarded.
- Verify the first `ctrl+c` does not quit and instead displays an explicit bottom-left warning telling the user to press `ctrl+c` again to quit.
- Verify the second `ctrl+c` within the emergency-quit window exits immediately.
- Verify leader-key configuration persists locally and invalid leader values are rejected clearly.

## Verification Notes
- Confirm the change replaces the uppercase-global-shortcut stopgap rather than adding more route-specific suppression rules.
- Confirm shell power actions require explicit entry through leader, `:`, or fuzzy command discovery rather than ambient plain-letter globals.
- Confirm the keyboard-ownership contract is richer than the current text-entry boolean.
- Confirm `space` is the default leader key and that leader-key configuration remains local-only convenience state.
- Confirm the command-entry surface is shell-owned and bottom-left aligned during `:` command mode.
- Confirm `Quit RuneCode` remains a visible shell action and not a fake route.
- Confirm emergency quit through `ctrl+c` now requires two presses.
- Confirm quit confirmation appears only when local-entry state would be discarded.
- Confirm help text and discoverability surfaces are generated from real action definitions.

## Results
- Verified the shell uses the richer keyboard-ownership contract to gate leader mode, command mode, focus traversal, and route-local typing.
- Verified leader mode opens immediately from the configured leader key, renders a which-key overlay, narrows valid next keys, aborts on `esc`, and aborts invalid sequences without executing unrelated actions.
- Verified `:` opens a shell-owned bottom-left command mode that supports typing, backspace, enter execution, `esc` abort, and inline parse/execution errors.
- Verified fuzzy discovery remains available through `ctrl+p` and is sourced from the same unified action graph used by leader mode and command aliases.
- Verified visible beginner quit discoverability remains available even when sidebar navigation is hidden, via the bottom strip quick-action hint tied to the real `shell.quit` action.
- Verified visible route discoverability no longer implies retired single-stroke route jumps; sidebar route hints now reflect the real leader-backed route-open bindings.
- Verified `ctrl+c` requires two presses to quit, the first press arms emergency quit and surfaces an explicit bottom-strip warning, and pending emergency state clears on timeout or normal interaction.
- Verified non-emergency quit confirmation appears only when active local entry state would be discarded.

## Close Gate
Use the repository's standard verification flow before closing this change.
