## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/security/trusted-runtime-evidence-and-broker-projection.md`

## Resolution Notes
This advanced TUI work remains bound to the same control-plane, local-first, trust-boundary, approval-binding, audit, and trusted-runtime-projection standards as the MVP foundation.

The richer workbench must be implemented by improving shell composition, typed reads, watch surfaces, and local convenience UX rather than by inventing client-local authority, scraping logs as the primary truth surface, or crossing trust boundaries for convenience.

Specific implications for this change:
- canonical object identity and queue semantics come from broker-visible models
- local workbench state such as sidebar visibility, pane ratios, themes, recents, and pinned sessions is non-authoritative
- inspection and live activity must rely on typed broker contracts and watch streams
- the trusted TUI shell in `cmd/` must preserve repo trust-boundary rules and must not reach into untrusted runner-only surfaces
