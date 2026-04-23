## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/runner-boundary-check.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/global/control-plane-api-contract-shape.md`

## Resolution Notes
This change exists to stop live chat and autonomous operation from drifting into separate execution systems.

That includes preserving one session-to-execution trigger model, one isolate-backed workflow path, one approval model, and one broker-owned lifecycle truth across all user-visible interaction modes.

This change also inherits the repo-scoped product instance, canonical `runecode` lifecycle surface, and diagnostics/remediation-only attach posture frozen by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`; execution orchestration must extend that foundation rather than redefining it.
