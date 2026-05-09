---
name: tui-snapshot-review
description: Use deterministic TUI snapshots for cmd/runecode-tui layout, wording, spacing, color, and visual polish audits.
argument-hint: "[full audit | focused area, scenario, route, or viewport]"
disable-model-invocation: true
---

Use this workflow when changing or reviewing `cmd/runecode-tui` UI/UX, layout, route wording, spacing, color treatment, shell chrome, overlays, or other visual polish.

The snapshot tool now supports agent-facing audit bundles. Use bundles for audit scope selection. Use raw scenarios only when the request is narrowly scoped to one exact deterministic state.

## Core Principles

- Default to non-GUI review. Do not open desktop windows unless the human explicitly asks for GUI review.
- Snapshot runs clean their output directory before and after by default. Use `TUI_SNAPSHOT_KEEP=1` when artifacts must persist for inspection.
- Snapshot artifacts default to repo-local `.tui-snapshots/`. `TUI_SNAPSHOT_DIR` may override that to another allowed snapshot output directory.
- Snapshot tooling is dev/CI-only. Do not add snapshot-enabled binaries, flags, or helpers to release artifacts.
- In plan/read-only mode, do not claim a TUI audit unless a preserved snapshot manifest already exists. Missing preserved artifacts are a workflow blocker, not an audit finding.
- `python3 ./tools/tui_snapshot_review.py ...` is read-only. `just tui-snapshot-review*` recipes rebuild and regenerate snapshots first, so they are not read-only inspection commands.
- Use `justfile` recipes as the command source of truth.

## Available Commands

- Full local snapshot set: `just tui-snapshot-all`
- Full audit bundle: `just tui-snapshot-audit-full`
- Dashboard-focused snapshots: `just tui-snapshot-dashboard`
- Dashboard desktop and mobile snapshots: `just tui-snapshot-dashboard-multi`
- Action Center-focused snapshots: `just tui-snapshot-action-center`
- Focused audit bundles:
  - `just tui-snapshot-audit-dashboard`
  - `just tui-snapshot-audit-action-center`
  - `just tui-snapshot-audit-runs`
  - `just tui-snapshot-audit-approvals`
  - `just tui-snapshot-audit-audit`
  - `just tui-snapshot-audit-status`
  - `just tui-snapshot-audit-setup`
  - `just tui-snapshot-audit-chat`
- Fresh full-audit plus non-GUI summary: `just tui-snapshot-review`
- Fresh full-audit plus non-GUI artifact path listing: `just tui-snapshot-review-list`
- Explicit GUI review: `just tui-snapshot-review-open`
- Deterministic CI/dev validation: `just tui-snapshot-ci`
- Release safety check: `just tui-release-safety`
- Read-only preserved-artifact summary: `python3 ./tools/tui_snapshot_review.py --mode summary ./.tui-snapshots`
- Read-only preserved-artifact path listing: `python3 ./tools/tui_snapshot_review.py --mode list ./.tui-snapshots`

## Environment Controls

- Preserve artifacts: `TUI_SNAPSHOT_KEEP=1`
- Override output directory: `TUI_SNAPSHOT_DIR=/tmp/my-snapshots`
- Combine controls when an agent must inspect files directly: `TUI_SNAPSHOT_KEEP=1 TUI_SNAPSHOT_DIR=/tmp/runecode-tui-agent-review just tui-snapshot-audit-full`

## Mode Detection Preflight

1. Check the current OpenCode/system context before running any snapshot command.
2. If the context says plan/read-only mode is active, treat all write-producing snapshot commands as forbidden:
   - `just tui-dev-snapshot-build`
   - `just tui-snapshot-audit-*`
   - `just tui-snapshot-review*`
3. Do not perform a write probe to discover mode. If the context says read-only, believe it.
4. In plan/read-only mode, only inspect an already-preserved temp directory with explicit scope validation:

```sh
python3 ./tools/tui_snapshot_review.py \
  --mode summary \
  --require-bundle full-audit \
  --require-viewport desktop \
  ./.tui-snapshots

python3 ./tools/tui_snapshot_review.py \
  --mode list \
  --require-bundle full-audit \
  --require-viewport desktop \
  ./.tui-snapshots
```

5. If the preserved bundle is missing or incomplete, stop and tell the user exactly how to continue:

```text
I’m in OpenCode plan/read-only mode, so I can’t generate the TUI snapshot bundle because that writes a snapshot binary and snapshot artifacts.

To continue, either switch me to build mode, or run this command yourself:

TUI_SNAPSHOT_KEEP=1 TUI_SNAPSHOT_DIR=./.tui-snapshots just tui-snapshot-audit-full

Then tell me to continue, and I’ll audit the preserved bundle with:

python3 ./tools/tui_snapshot_review.py --mode summary --require-bundle full-audit --require-viewport desktop ./.tui-snapshots
python3 ./tools/tui_snapshot_review.py --mode list --require-bundle full-audit --require-viewport desktop ./.tui-snapshots
```

6. In write-capable mode, proceed with bundle generation and preserved-artifact review normally.

## Read-Only Or Plan Mode

1. Do not run `just tui-dev-snapshot-build`, `just tui-snapshot-audit-*`, or `just tui-snapshot-review*` in read-only/plan mode. Those commands rebuild binaries or write snapshot artifacts.
2. Only inspect an already-preserved temp directory:

```sh
python3 ./tools/tui_snapshot_review.py --mode summary --require-bundle full-audit ./.tui-snapshots
python3 ./tools/tui_snapshot_review.py --mode list --require-bundle full-audit ./.tui-snapshots
```

3. If `manifest.json` is missing or the listed PNG/SVG artifacts are absent, report a workflow blocker. Do not convert missing artifacts into a TUI audit finding.
4. State clearly that no audit coverage can be claimed until either:
   - a preserved bundle exists under an allowed snapshot dir, or
   - write-producing snapshot commands are allowed again.

## Audit Bundles And Scenarios

Use public audit-bundle recipes first. When a focused audit needs a specific deterministic state, build the snapshot binary and invoke the helper directly.

Agent-facing audit bundles:

- `full-audit`: representative desktop audit coverage across Dashboard, Chat, Runs, Approvals, Action Center, Audit, Status, Model Providers, Git Setup, and Git Remote. Agents must confirm this bundle coverage in the non-GUI summary before claiming a full audit was completed.
- `dashboard-audit`
- `action-center-audit`
- `runs-audit`
- `approvals-audit`
- `audit-route-audit`
- `status-audit`
- `setup-audit`
- `chat-audit`

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
2. Use the full audit bundle, not the raw `all` scenario group.
3. Generate and preserve the full audit bundle for inspection:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-full
```

4. Produce a non-GUI summary from the preserved artifact directory:

```sh
python3 ./tools/tui_snapshot_review.py --mode summary --require-bundle full-audit --require-viewport desktop ./.tui-snapshots
```

5. Print artifact paths for direct file inspection from the same preserved directory:

```sh
python3 ./tools/tui_snapshot_review.py --mode list --require-bundle full-audit --require-viewport desktop ./.tui-snapshots
```

6. Inspect the selected `.png` or `.svg` files directly with file-reading tools. Do not open GUI windows unless asked.
7. Confirm the review summary reports the expected bundle, route coverage, viewport coverage, and scenario coverage before claiming the audit is complete.
Route coverage expected today: Dashboard, Chat, Runs, Approvals, Action Center, Audit, Status, Model Providers, Git Setup, and Git Remote.
8. Review each captured route/state for route hierarchy, text density, sidebar behavior, footer/chrome weight, inspector layout, modal/overlay polish, color contrast, and raw debug-token leakage.
9. Compare desktop and mobile/compact behavior when the audit involves layout responsiveness:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-dashboard-multi
```

10. Record findings with bundle name, scenario name, viewport, artifact path, and concrete issue.
11. If code changes are made, regenerate the relevant bundle or focused route snapshots and compare the updated artifacts.
12. Run validation before handoff:

```sh
just tui-snapshot-ci
```

13. If snapshot plumbing, build tags, release safety, or dev/CI boundaries changed, also run:

```sh
just tui-release-safety
```

## Focused TUI Audit Workflow

Use this workflow when the human asks to audit one route, state, viewport, or a small polish change.

1. Identify the target surface.
2. Map the request to the smallest matching audit bundle first:

```sh
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-dashboard
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-action-center
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-runs
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-approvals
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-audit
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-status
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-setup
TUI_SNAPSHOT_KEEP=1 just tui-snapshot-audit-chat
```

3. Only fall back to a raw scenario when the request is truly about one exact deterministic state.
4. Use non-GUI review first against the preserved output directory:

```sh
python3 ./tools/tui_snapshot_review.py --mode summary --require-bundle dashboard-audit ./.tui-snapshots
python3 ./tools/tui_snapshot_review.py --mode list --require-bundle dashboard-audit ./.tui-snapshots
```

5. Confirm the summary/list output matches the requested focused bundle or focused scenario before analyzing it.
If the user asks for a route or area without a matching bundle yet, state exactly which existing bundle or explicit scenario you are using as the nearest proxy and what route coverage it actually provides.
6. Inspect only the relevant artifact paths. Ignore unrelated scenarios.
7. Analyze the focused area for the requested concern, such as wording, spacing, overflow, truncation, color, responsive layout, or debug-token exposure.
8. If a fix is implemented, regenerate only the focused bundle or focused scenario where possible.
9. Run at least the relevant targeted verification:

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
- Bundle or scenario requested
- Scenario(s) and viewport(s) reviewed
- Route coverage confirmed from manifest summary
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
