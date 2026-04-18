## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/trust-boundary-change-checklist.md`
- `standards/security/runner-boundary-check.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/runner-durable-state-and-replay.md`
- `standards/global/control-plane-api-contract-shape.md`

## Resolution Notes
This optional follow-on change exists to keep any future LangGraph adoption explicitly subordinate to RuneCode's trust-boundary, broker-authority, approval-binding, runner-replay, and control-plane contract standards rather than treating a third-party runtime as the architectural source of truth.

This now explicitly includes preserving exact-action hard-floor approval semantics for lanes such as `git_remote_ops` and fail-closed remote-drift handling during wait, replay, and resume.
