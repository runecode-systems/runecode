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
- Go `benchstat`: https://pkg.go.dev/golang.org/x/perf/cmd/benchstat
- Criterion analysis guide: https://bheisler.github.io/criterion.rs/book/analysis.html
- LLVM benchmarking tips: https://llvm.org/docs/Benchmarking.html

## Related Changes

- `runecontext/changes/CHG-2026-008-62e1-broker-local-api-v0/`
- `runecontext/changes/CHG-2026-009-1672-launcher-microvm-backend-v0/`
- `runecontext/changes/CHG-2026-010-54b7-container-backend-v0-explicit-opt-in/`
- `runecontext/changes/CHG-2026-011-7240-secretsd-model-gateway-v0/`
- `runecontext/changes/CHG-2026-012-f1ef-workflow-runner-workspace-roles-deterministic-gates-v0/`
- `runecontext/changes/CHG-2026-013-d2c9-minimal-tui-v0/`
- `runecontext/changes/CHG-2026-025-5679-external-audit-anchoring-v0/`
- `runecontext/changes/CHG-2026-037-91be-tui-multi-session-power-workspace-v0/`
- `runecontext/changes/CHG-2026-043-8e9b-live-activity-watch-streams-v0/`
- `runecontext/changes/CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0/`
- `runecontext/changes/CHG-2026-048-6b7a-session-execution-orchestration-v0/`
- `runecontext/changes/CHG-2026-049-1d4e-first-party-runecontext-workflow-pack-v0/`
- `runecontext/changes/CHG-2026-050-e3f8-workflow-definition-contract-binding-v0/`
- `runecontext/changes/CHG-2026-054-6c1e-runtime-attestation-post-handshake-gating-v0/`

## Planning Notes

- The first live TUI CPU sample was corrected after the investigation confirmed that `--runtime-dir` and `--socket-name` isolate only the local IPC listener, not the broker store or audit ledger.
- The corrected empty-state measurement stayed around `0.5-1.0%` CPU for the real `runecode-tui` child, while the earlier non-empty-state sample reflected active or waiting repo-scoped broker state.
- The durable performance concern after correction is long-lived active or waiting-state cost rather than empty-idle cost alone.
- After the alpha.7 waiting-state split landed, a fresh isolated rerun measured empty-state CPU at `0.20-0.80%` and waiting-state CPU at `0.00-1.00%` for the real `runecode-tui` child.
- The strongest before/after comparison is the waiting-state path: the earlier sample climbed through `22.81%` and `61.92%` CPU, while the post-fix isolated waiting sample stayed at `1.00%` mid and aged CPU.
- The post-fix waiting transcript still rendered `WAITING session=sess-manual-multiwait`, confirming the improvement came from removing the fast repaint loop for waiting states rather than from hiding the state cue.
- The first durable gate set should use a dedicated reviewed performance-contract artifact family rather than overloading `runecontext/assurance/baseline.yaml`, which remains part of project-substrate assurance posture.
- The first implementation slice should use reviewed statistical defaults per metric class: repeated-sample robust comparison for microbenchmarks, median plus `p95` plus explicit ceilings for latency metrics, fixed-window repeated sampling with average or median plus max guardrails for CPU/process-behavior metrics, and exact comparison for deterministic invariant counts.
- Performance timing boundaries should terminate on reviewed broker-owned or persisted milestones whenever those authoritative surfaces exist downstream in the product contract.
- Follow-up review refined the implementation foundation further: performance contracts should live under `runecontext/assurance/performance/`, use stable fixture IDs, declare lane authority and activation state, declare threshold provenance, and include timing-boundary metadata before required enforcement.
- Shared hosted Linux remains acceptable for stable required checks, while high-noise checks should start informational, remain pending dependency, or move to a tighter Linux authority without changing RuneCode's product architecture.
