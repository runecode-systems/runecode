# Tasks

## HVF Reliability + UX

- [ ] Harden HVF-based QEMU startup flows.
- [ ] Improve diagnostics for common host limitations.

## Optional Virtualization.framework Backend

- [ ] Evaluate adopting Virtualization.framework for improved UX and performance.
- [ ] Keep the capability model and audit semantics unchanged.

## Packaging + Permissions

- [ ] Define installation and update UX (codesigning/notarization where needed).
- [ ] Ensure local IPC permissions remain least-privilege.

## Acceptance Criteria

- [ ] macOS users can run microVM-backed roles with minimal setup friction.
- [ ] The active backend and assurance level are always visible and audited.
- [ ] The active backend and assurance level remain visible through the same shared broker operator-facing run surfaces used on other platforms.
