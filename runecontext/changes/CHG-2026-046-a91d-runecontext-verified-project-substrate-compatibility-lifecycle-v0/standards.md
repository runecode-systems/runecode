## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/security/approval-binding-and-verifier-identity.md`
- `standards/security/trusted-local-artifact-persistence.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`

## Resolution Notes
This change exists to turn the accepted RuneContext migration decisions into a real product capability: one canonical project substrate, one verified-mode compatibility lifecycle, and one auditable upgrade path.

That includes keeping hard compatibility enforcement in RuneCode, advisory generic compatibility in RuneContext, and assurance-binding tied to canonical project state rather than to local product-private mirrors.

This now also includes:
- compatibility evaluated against the repository's declared substrate contract rather than each developer's installed tool version
- one broker-owned typed project-posture authority surface rather than readiness-only or client-local heuristics
- validated snapshot-digest binding for later planning, audit, attestation, and verification features
- future dashboard/operator prompts staying subordinate to the same shared broker and approval contracts rather than creating a setup-only exception path
