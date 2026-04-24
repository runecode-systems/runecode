## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/policy-evaluation-foundations.md`

## Resolution Notes
This change remains bound to the same local-first, control-plane, and trust-boundary rules as the existing TUI foundation and advanced workbench shell.

The new leader and command-mode input system must be implemented as shell-owned interaction architecture and local-only convenience behavior, not as broker-owned policy truth, repository-shared preference state, or client-local authority over canonical control-plane meaning.

Specific implications for this change:
- keyboard entry behavior must not blur the shell/client boundary with broker authority
- local leader-key preference remains non-authoritative convenience state
- discoverability, help, and command execution must remain generated from real shell action definitions
- the trusted TUI shell in `cmd/` must preserve repo trust-boundary rules and must not reach into runner-only surfaces for input semantics or shortcut behavior
