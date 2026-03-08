# Windows MicroVM Runtime Support — Post-MVP

User-visible outcome: RuneCode can run microVM-based roles on Windows with the same capability model and audit semantics.

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-03-08-1039-windows-microvm-runtime/` with:
- `plan.md` (this file)
- `shape.md`
- `standards.md`
- `references.md`
- `visuals/` (empty)

## Task 2: Windows MicroVM Backend Implementation

- Implement QEMU acceleration via WHPX/Hyper-V.
- Ensure parity with Linux microVM backend interfaces.

## Task 3: Windows Service + Local IPC

- Define how launcher/broker run as services.
- Use named pipes with strict ACLs for local API.

## Task 4: Packaging + Prereqs

- Define required host capabilities (virtualization enabled, Hyper-V availability).
- Provide clear diagnostics when prerequisites are missing.

## Task 5: CI/Testing Strategy

- Keep Windows CI coverage strong for backend-agnostic components.
- Add microVM integration tests via self-hosted runners if required.

## Acceptance Criteria

- MicroVM roles can be launched on Windows and produce the same audit/artifact outputs.
- Reduced-assurance container mode remains explicit opt-in.
