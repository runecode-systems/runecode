# macOS Virtualization Polish — Post-MVP

User-visible outcome: RuneCode’s microVM experience on macOS is reliable and ergonomic, with good performance and clear system integration.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-macos-virtualization-polish/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: HVF Reliability + UX

- Harden HVF-based QEMU startup flows.
- Improve diagnostics for common host limitations.

## Task 3: Optional Virtualization.framework Backend

- Evaluate adopting Virtualization.framework for improved UX/perf.
- Keep capability model and audit semantics unchanged.

## Task 4: Packaging + Permissions

- Define installation and update UX (codesigning/notarization where needed).
- Ensure local IPC permissions remain least-privilege.

## Acceptance Criteria

- macOS users can run microVM-backed roles with minimal setup friction.
- The active backend and assurance level are always visible and audited.
