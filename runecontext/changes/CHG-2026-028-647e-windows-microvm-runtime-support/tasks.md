# Tasks

## Windows MicroVM Backend Implementation

- [ ] Implement QEMU acceleration via WHPX/Hyper-V.
- [ ] Ensure parity with Linux microVM backend contracts, including launch/session/attachment semantics, hardening posture recording, terminal reporting, and isolate-session audit payload expectations.

## Windows Service + Local IPC

- [ ] Define how launcher and broker run as services.
- [ ] Use named pipes with strict ACLs for the local API while preserving the same logical broker API and session semantics.

## Packaging + Prereqs

- [ ] Define required host capabilities (virtualization enabled, Hyper-V availability).
- [ ] Provide clear diagnostics when prerequisites are missing.

## CI/Testing Strategy

- [ ] Keep Windows CI coverage strong for backend-agnostic components.
- [ ] Add microVM integration tests via self-hosted runners if required.

## Acceptance Criteria

- [ ] MicroVM roles can be launched on Windows and produce the same audit/artifact outputs and the same operator-visible runtime posture semantics.
- [ ] Reduced-assurance container mode remains explicit opt-in.
