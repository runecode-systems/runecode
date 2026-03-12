# macOS Virtualization Polish — Post-MVP

User-visible outcome: RuneCode’s microVM experience on macOS is reliable and ergonomic, with good performance and clear system integration.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-macos-virtualization-polish/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

Parallelization: docs-only; safe to do anytime.

## Task 2: HVF Reliability + UX

- Harden HVF-based QEMU startup flows.
- Improve diagnostics for common host limitations.

Parallelization: can be developed in parallel with Linux microVM work; keep interfaces aligned.

## Task 3: Optional Virtualization.framework Backend

- Evaluate adopting Virtualization.framework for improved UX/perf.
- Keep capability model and audit semantics unchanged.

Parallelization: can be evaluated in parallel with other post-MVP platform work.

## Task 4: Packaging + Permissions

- Define installation and update UX (codesigning/notarization where needed).
- Ensure local IPC permissions remain least-privilege.

Parallelization: can be developed in parallel with TUI/CLI packaging work; coordinate on install/update UX.

## Acceptance Criteria

- macOS users can run microVM-backed roles with minimal setup friction.
- The active backend and assurance level are always visible and audited.
