## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/global/local-product-lifecycle-and-attach-contract.md`
- `standards/global/project-substrate-contract-and-lifecycle.md`
- `standards/global/session-execution-contract-and-watch-families.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/trusted-run-plan-authority-and-selection.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/protocol-schema-invariants.md`

## Resolution Notes
This change exists to make RuneCode productively useful without creating a second built-in workflow system that later custom workflows would have to imitate or bypass.

That includes keeping first-party change/spec drafting and approved-change implementation on the same typed workflow, approval, audit, verification, and git foundations as the rest of the product.

It also includes inheriting the canonical repo-scoped RuneCode product lifecycle from `CHG-2026-047-c3e2-local-control-plane-bootstrap-persistent-session-lifecycle-v0` rather than inventing a built-in-only bootstrap, attach, or remediation path.

This change now also freezes these foundation decisions for `v0`:
- built-in workflow identities are product-shipped reviewed assets and are not repository-overridable
- drafting is artifact-first with explicit promote/apply semantics
- approved-change implementation binds exact reviewed input sets rather than ambient planning state
- direct CLI entrypoints remain thin adapters over the same broker-owned trigger and execution contracts as chat and autonomous operation
- performance work must optimize one shared topology-neutral architecture across constrained and scaled environments rather than introducing environment-specific workflow semantics
