# Tasks

## Windows MicroVM Backend Implementation

- [ ] Implement QEMU acceleration via WHPX/Hyper-V.
- [ ] Ensure parity with Linux microVM backend contracts, including runtime-image identity, boot-profile handling, trusted-admission rules, verified-cache semantics, launch/session/attachment semantics, attestation evidence and verification semantics, hardening posture recording, terminal reporting, and isolate-session audit payload expectations.
- [ ] Require valid attestation for all supported production and user-facing Windows runtime paths.
- [ ] Disallow Windows-specific automatic fallback, manual override, default configuration, documented operator flow, CLI flag, TUI action, or platform exception that would permit TOFU trust decisions.

## Windows Service + Local IPC

- [ ] Define how launcher and broker run as services.
- [ ] Use named pipes with strict ACLs for the local API while preserving the same logical broker API and session semantics.
- [ ] Preserve one repo-scoped RuneCode product instance per authoritative repository root rather than redefining lifecycle around host-global service identity or named-pipe identity.
- [ ] Keep Windows service state, named-pipe reachability, and bootstrap-local artifacts non-authoritative for operator UX; broker-owned product lifecycle posture remains authoritative.
- [ ] Preserve canonical `runecode` attach/start/status/stop/restart semantics above Windows-specific service and IPC realization details.

## Packaging + Prereqs

- [ ] Define required host capabilities (virtualization enabled, Hyper-V availability).
- [ ] Provide clear diagnostics when prerequisites are missing.
- [ ] Ensure Windows packaging and bootstrap flows preserve the same published signed runtime-asset and verified local cache trust model used on other platforms.
- [ ] Ensure missing Windows prerequisites or missing attestation capability fail closed rather than downgrading to a supported TOFU posture.

## CI/Testing Strategy

- [ ] Keep Windows CI coverage strong for backend-agnostic components.
- [ ] Add microVM integration tests via self-hosted runners if required.

## Acceptance Criteria

- [ ] MicroVM roles can be launched on Windows and produce the same audit/artifact outputs and the same operator-visible runtime posture semantics.
- [ ] Windows microVM launch consumes the same signed immutable runtime assets, preserves the same launch-admission and launch-evidence semantics used on other platforms, and requires the same valid attestation posture for supported production and user-facing runtime use.
- [ ] Reduced-assurance container mode remains explicit opt-in.
- [ ] Windows preserves the same repo-scoped product-lifecycle semantics and canonical `runecode` user-surface behavior as other platforms.
- [ ] No Windows-specific supported path permits TOFU trust decisions.
