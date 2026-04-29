# Tasks

## HVF Reliability + UX

- [ ] Harden HVF-based QEMU startup flows.
- [ ] Improve diagnostics for common host limitations.
- [ ] Preserve the same signed runtime-asset, boot-profile, trusted-admission, and verified-cache model used on other platforms.

## Optional Virtualization.framework Backend

- [ ] Evaluate adopting Virtualization.framework for improved UX and performance.
- [ ] Keep the capability model, runtime-image identity, launch/session/attachment semantics, launch-evidence semantics, and audit semantics unchanged.
- [ ] Preserve one repo-scoped RuneCode product instance per authoritative repository root rather than redefining lifecycle around platform-specific runtime helpers.

## Packaging + Permissions

- [ ] Define installation and update UX (codesigning/notarization where needed).
- [ ] Ensure local IPC permissions remain least-privilege.
- [ ] Keep macOS packaging state, bootstrap-local artifacts, and local IPC reachability non-authoritative for operator UX; broker-owned product lifecycle posture remains authoritative.
- [ ] Preserve canonical `runecode` attach/start/status/stop/restart semantics above macOS-specific runtime and packaging realization details.

## Acceptance Criteria

- [ ] macOS users can run microVM-backed roles with minimal setup friction.
- [ ] The active `backend_kind`, runtime isolation assurance, provisioning/binding posture, and audit posture are always visible and auditable.
- [ ] Those runtime posture dimensions remain visible through the same shared broker operator-facing run surfaces used on other platforms.
- [ ] macOS microVM launch consumes the same signed immutable runtime assets and preserves the same launch-admission and launch-evidence semantics used on other platforms.
- [ ] macOS preserves the same repo-scoped product-lifecycle semantics and canonical `runecode` user-surface behavior as other platforms.
