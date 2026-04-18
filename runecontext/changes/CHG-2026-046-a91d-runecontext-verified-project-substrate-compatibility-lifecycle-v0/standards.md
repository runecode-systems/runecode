## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/security/trusted-local-artifact-persistence.md`
- `standards/global/control-plane-api-contract-shape.md`
- `standards/global/local-first-future-optionality.md`

## Resolution Notes
This change exists to turn the accepted RuneContext migration decisions into a real product capability: one canonical project substrate, one verified-mode compatibility lifecycle, and one auditable upgrade path.

That includes keeping hard compatibility enforcement in RuneCode, advisory generic compatibility in RuneContext, and assurance-binding tied to canonical project state rather than to local product-private mirrors.
