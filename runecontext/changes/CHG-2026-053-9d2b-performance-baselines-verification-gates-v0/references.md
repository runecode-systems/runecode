# References

## Product Context

- **Mission:** `runecontext/project/mission.md`
- **Tech stack:** `runecontext/project/tech-stack.md`
- **Standards inventory:** `runecontext/project/standards-inventory.md`
- **Trust boundaries:** `docs/trust-boundaries.md`
- **Source quality policy:** `docs/source-quality.md`
- **Roadmap:** `runecontext/project/roadmap.md`
- **Canonical command source:** `justfile`
- **CI workflow:** `.github/workflows/ci.yml`

## Investigation Source Files

- `cmd/runecode-tui/main.go`
- `cmd/runecode-tui/shell_model.go`
- `cmd/runecode-tui/shell_update.go`
- `cmd/runecode-tui/shell_view.go`
- `cmd/runecode-tui/shell_watch_transport.go`
- `cmd/runecode-tui/shell_watch_reduction.go`
- `cmd/runecode-tui/shell_watch_projection.go`
- `cmd/runecode-tui/shell_render_helpers.go`
- `cmd/runecode-tui/shell_object_index.go`
- `cmd/runecode-tui/command_surface.go`
- `cmd/runecode-tui/broker_client.go`
- `cmd/runecode-tui/broker_client_helpers.go`
- `cmd/runecode-tui/broker_client_ipc_config.go`
- `cmd/runecode-broker/main_base.go`
- `internal/brokerapi/local_api_watch_streams.go`
- `internal/brokerapi/local_api_watch_event_builders.go`
- `internal/brokerapi/local_api_types_session.go`
- `internal/brokerapi/local_ipc_linux.go`

## External Implementation References

- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Bubbles: https://github.com/charmbracelet/bubbles
- Lip Gloss: https://github.com/charmbracelet/lipgloss
- Go `pprof`: https://pkg.go.dev/runtime/pprof
- Go benchmarking: https://pkg.go.dev/testing

## Related Changes

- `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`
- `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/`
- `runecontext/changes/CHG-2026-010-54b7-container-backend-v0-explicit-opt-in/`
- `runecontext/changes/CHG-2026-011-7240-secretsd-model-gateway-v0/`
- `runecontext/changes/CHG-2026-012-f1ef-workflow-runner-workspace-roles-deterministic-gates-v0/`
- `runecontext/changes/CHG-2026-013-d2c9-minimal-tui-v0/`
- `runecontext/changes/CHG-2026-037-91be-tui-multi-session-power-workspace-v0/`
- `runecontext/changes/CHG-2026-043-8e9b-live-activity-watch-streams-v0/`
- `runecontext/changes/CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0/`
- `runecontext/changes/CHG-2026-048-6b7a-session-execution-orchestration-v0/`
- `runecontext/changes/CHG-2026-049-1d4e-first-party-runecontext-workflow-pack-v0/`
- `runecontext/changes/CHG-2026-050-e3f8-workflow-definition-contract-binding-v0/`

## Planning Notes

- The first live TUI CPU sample was corrected after the investigation confirmed that `--runtime-dir` and `--socket-name` isolate only the local IPC listener, not the broker store or audit ledger.
- The corrected empty-state measurement stayed around `0.5-1.0%` CPU for the real `runecode-tui` child, while the earlier non-empty-state sample reflected active or waiting repo-scoped broker state.
- The durable performance concern after correction is long-lived active or waiting-state cost rather than empty-idle cost alone.
