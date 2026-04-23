# Tasks

## Windows MicroVM Backend Implementation

- [ ] Implement QEMU acceleration via WHPX/Hyper-V.
- [ ] Ensure parity with Linux microVM backend contracts, including launch/session/attachment semantics, hardening posture recording, terminal reporting, and isolate-session audit payload expectations.

## Windows Service + Local IPC

- [ ] Define how launcher and broker run as services.
- [ ] Use named pipes with strict ACLs for the local API while preserving the same logical broker API and session semantics.
- [ ] Preserve one repo-scoped RuneCode product instance per authoritative repository root rather than redefining lifecycle around host-global service identity or named-pipe identity.
- [ ] Keep Windows service state, named-pipe reachability, and bootstrap-local artifacts non-authoritative for operator UX; broker-owned product lifecycle posture remains authoritative.
- [ ] Preserve canonical `runecode` attach/start/status/stop/restart semantics above Windows-specific service and IPC realization details.

## Packaging + Prereqs

- [ ] Define required host capabilities (virtualization enabled, Hyper-V availability).
- [ ] Provide clear diagnostics when prerequisites are missing.

## CI/Testing Strategy

- [ ] Keep Windows CI coverage strong for backend-agnostic components.
- [ ] Add microVM integration tests via self-hosted runners if required.

## Acceptance Criteria

- [ ] MicroVM roles can be launched on Windows and produce the same audit/artifact outputs and the same operator-visible runtime posture semantics.
- [ ] Reduced-assurance container mode remains explicit opt-in.
- [ ] Windows preserves the same repo-scoped product-lifecycle semantics and canonical `runecode` user-surface behavior as other platforms.
