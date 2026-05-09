---
name: tui-snapshot-review
description: Use deterministic TUI snapshots for cmd/runecode-tui layout, wording, spacing, color, and visual polish audits.
argument-hint: "[full audit | focused area, scenario, route, or viewport]"
disable-model-invocation: true
---

Use this workflow when changing or reviewing `cmd/runecode-tui` UI/UX, layout, route wording, spacing, color treatment, shell chrome, overlays, or other visual polish.

## Core Principles

- Default to non-GUI review. Do not open desktop windows unless the human explicitly asks for GUI review.
- Snapshot runs clean their output directory before and after by default. Use `TUI_SNAPSHOT_KEEP=1` when artifacts must persist for inspection.
- Snapshot artifacts must stay under trusted temp roots. Use `TUI_SNAPSHOT_DIR` only for a temp-backed local output directory.
- Snapshot tooling is dev/CI-only. Do not add snapshot-enabled binaries, flags, or helpers to release artifacts.
- Use `justfile` recipes as the command source of truth.

## Available Commands

- Full local snapshot set: `just tui-snapshot-all`
- Dashboard-focused snapshots: `just tui-snapshot-dashboard`
- Dashboard desktop and mobile snapshots: `just tui-snapshot-dashboard-multi`
- Action Center-focused snapshots: `just tui-snapshot-action-center`
- Non-GUI summary review: `just tui-snapshot-review`
- Non-GUI artifact path listing: `just tui-snapshot-review-list`
- Explicit GUI review: `just tui-snapshot-review-open`
- Deterministic CI/dev validation: `just tui-snapshot-ci`
- Release safety check: `just tui-release-safety`

## Environment Controls

- Preserve artifacts: `TUI_SNAPSHOT_KEEP=1`
- Override output directory: `TUI_SNAPSHOT_DIR=/tmp/my-snapshots`
- Combine controls when an agent must inspect files directly: `TUI_SNAPSHOT_KEEP=1 TUI_SNAPSHOT_DIR=/tmp/runecode-tui-agent-review just tui-snapshot-all`

## Snapshot Scenarios

Use public recipes first. When a focused audit needs a specific scenario, build the snapshot binary and invoke the helper directly.

Public scenario groups:

- `all`: all deterministic scenarios
- `dashboard`: dashboard scenarios
- `action-center`: Action Center triage scenario

Specific deterministic scenarios:

- `dashboard-healthy-empty`
- `dashboard-approval-waiting`
- `dashboard-blocked`
- `dashboard-degraded`
- `action-center-triage`

Viewport presets:

- `desktop`: 160x48
- `compact`: 120x36
- `mobile`: 80x32

Focused direct helper pattern:

```sh
just tui-dev-snapshot-build
TUI_SNAPSHOT_KEEP=1 sh ./tools/tui_snapshot_local.sh \
  --binary /tmp/runecode-current/bin/runecode-tui \
  --scenario dashboard-degraded \
  --viewport desktop \
  --review \
  --review-mode summary
```

## Full TUI Audit Workflow

1. Confirm the request is a full visual audit or broad polish review for `cmd/runecode-tui`.
2. Generate and preserve all snapshots for inspection:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-all
```

3. Produce a non-GUI summary:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-review
```

4. Print artifact paths for direct file inspection:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-review-list
```

5. Inspect the selected `.png` or `.svg` files directly with file-reading tools. Do not open GUI windows unless asked.
6. Review each scenario for route hierarchy, text density, sidebar behavior, footer/chrome weight, inspector layout, modal/overlay polish, color contrast, and raw debug-token leakage.
7. Compare desktop and mobile/compact behavior when the audit involves layout responsiveness:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-dashboard-multi
```

8. Record findings with scenario name, viewport, artifact path, and concrete issue.
9. If code changes are made, regenerate the relevant snapshots and compare the updated artifacts.
10. Run validation before handoff:

```sh
just tui-snapshot-ci
```

11. If snapshot plumbing, build tags, release safety, or dev/CI boundaries changed, also run:

```sh
just tui-release-safety
```

## Focused TUI Audit Workflow

Use this workflow when the human asks to audit one route, state, viewport, or a small polish change.

1. Identify the target surface.
2. Prefer the nearest public recipe:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-dashboard
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-action-center
```

3. For a specific scenario or viewport, use the direct helper pattern from the Snapshot Scenarios section.
4. Use non-GUI review first:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-review-list
```

5. Inspect only the relevant artifact paths. Ignore unrelated scenarios.
6. Analyze the focused area for the requested concern, such as wording, spacing, overflow, truncation, color, responsive layout, or debug-token exposure.
7. If a fix is implemented, regenerate only the focused route/scenario where possible.
8. Run at least the relevant targeted verification:

```sh
go test ./cmd/runecode-tui
just tui-snapshot-ci
```

## GUI Review

Only open GUI windows when the human explicitly asks for visual desktop review.

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-review-open
```

If GUI opening fails, report the selected artifact paths from `just tui-snapshot-review-list` instead of retrying with different openers.

## Review Checklist

- Does the primary route communicate product state before implementation detail?
- Are degraded, blocked, or approval-required states clear without raw token spam?
- Are raw proof and debug details routed to inspectors, Status, Runs, Audit, or Action Center detail surfaces?
- Does the sidebar stay readable at the selected viewport?
- Does the footer stay calm and avoid consuming too much vertical space?
- Are modals, overlays, palette rows, and inspectors aligned and clipped correctly?
- Are colors, emphasis, and accents useful without becoming noisy?
- Does mobile or compact layout preserve the core task path?
- Do generated artifacts match the manifest and remain deterministic across reruns?

## Reporting

When reporting an audit, include:

- Command(s) run
- Scenario(s) and viewport(s) reviewed
- Artifact paths inspected
- Findings ordered by severity
- Any fixes made
- Verification results

## Guardrails

- Do not use `tui-snapshot-review-open` unless the human explicitly requests GUI-opened artifacts.
- Do not preserve artifacts unless inspection is needed.
- Do not commit generated snapshot artifacts.
- Do not add snapshot tooling or snapshot-enabled binaries to release artifacts.
- Do not weaken temp-root, manifest path, or release-safety checks.
- If snapshot workflow behavior changes, run both `just tui-snapshot-ci` and `just tui-release-safety`.
