# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Standards inventory:** `runecontext/project/standards-inventory.md`
- **Trust boundaries:** `docs/trust-boundaries.md`

## UX And Implementation References

- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Bubbles: https://github.com/charmbracelet/bubbles
- Lip Gloss: https://github.com/charmbracelet/lipgloss
- NeoVim: https://neovim.io/
- Vim: https://www.vim.org/
- Which Key: https://github.com/folke/which-key.nvim

## Related Changes

- `runecontext/changes/CHG-2026-013-d2c9-minimal-tui-v0/`
- `runecontext/changes/CHG-2026-037-91be-tui-multi-session-power-workspace-v0/`
- `runecontext/changes/CHG-2026-038-5a1d-runecode-tui-experience-v0/`
- `runecontext/changes/CHG-2026-039-7c2e-interactive-control-plane-ux-contracts-v0/`
- `runecontext/changes/CHG-2026-040-2b7f-session-transcript-model-v0/`
- `runecontext/changes/CHG-2026-043-8e9b-live-activity-watch-streams-v0/`
- `runecontext/changes/CHG-2026-045-7f4c-direct-credential-model-providers-v0/`
- `runecontext/changes/CHG-2026-048-6b7a-session-execution-orchestration-v0/`

## Planning Notes

- Planning intentionally keeps the NeoVim-inspired leader and `:` concepts while rejecting NeoVim's beginner-hostile quitting and discoverability posture.
- The durable product rule is explicit power-user entry plus visible beginner affordances, not terminal cleverness for its own sake.
- The immediate trigger for this change was the temporary uppercase-global-shortcut stopgap that protected local route typing and secret entry but did not replace the underlying shell input model.
