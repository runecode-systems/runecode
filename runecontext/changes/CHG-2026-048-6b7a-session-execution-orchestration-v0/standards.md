## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/runner-boundary-check.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-product-lifecycle-and-attach-contract.md`
- `standards/global/project-substrate-contract-and-lifecycle.md`

## Resolution Notes
This change exists to stop live chat and autonomous operation from drifting into separate execution systems.

That includes preserving one session-to-execution trigger model, one isolate-backed workflow path, one approval model, one broker-owned turn execution state model, and one broker-owned lifecycle truth across all user-visible interaction modes.

This change now also explicitly freezes the following clarifications for that foundation:
- plain transcript append remains distinct from execution-trigger submission
- validated project-substrate snapshot digest is the canonical execution binding for project-context-sensitive work
- transcript checkpoint durability remains distinct from in-flight execution watch state
- inspect-only access in diagnostics/remediation-only attach remains distinct from productive execution authorization
- formal approval profile and operator-question frequency remain separate controls, while hard-floor approvals stay outside both controls
- pending user input or approval remains dependency-aware partial blocking rather than a whole-system stop signal

This change also inherits the repo-scoped product instance, canonical `runecode` lifecycle surface, and diagnostics/remediation-only attach posture frozen by `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0`; execution orchestration must extend that foundation rather than redefining it.
