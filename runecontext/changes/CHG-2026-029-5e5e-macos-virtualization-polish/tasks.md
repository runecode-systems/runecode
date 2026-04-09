# Tasks

## HVF Reliability + UX

- [ ] Harden HVF-based QEMU startup flows.
- [ ] Improve diagnostics for common host limitations.

## Optional Virtualization.framework Backend

- [ ] Evaluate adopting Virtualization.framework for improved UX and performance.
- [ ] Keep the capability model, launch/session/attachment semantics, and audit semantics unchanged.

## Packaging + Permissions

- [ ] Define installation and update UX (codesigning/notarization where needed).
- [ ] Ensure local IPC permissions remain least-privilege.

## Acceptance Criteria

- [ ] macOS users can run microVM-backed roles with minimal setup friction.
- [ ] The active `backend_kind`, runtime isolation assurance, provisioning/binding posture, and audit posture are always visible and auditable.
- [ ] Those runtime posture dimensions remain visible through the same shared broker operator-facing run surfaces used on other platforms.
