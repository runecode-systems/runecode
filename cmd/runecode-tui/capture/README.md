# runecode-tui visual capture

Repeatable visual capture for CHG-2026-037 Phase 12 workbench flows.

## Prereqs

- local broker in another terminal (recommended):
  - `runecode-broker serve-local`
- `runecode-tui` binary on `PATH` (or run from repo root with `go run ./cmd/runecode-tui` after editing the tape command)
- VHS installed (`vhs`)

## Capture command

From repository root:

```bash
vhs "cmd/runecode-tui/capture/workbench-flows.tape"
```

Output file:

- `workbench-flows.gif` in current working directory.

## Covered flow snapshots

- shell pane framing + focus affordance
- route/object quick transitions from palette
- inspector summary-to-detail mode changes (`rendered/raw/structured`)
- selection mode status visibility
- sidebar toggle + copy-action loop
