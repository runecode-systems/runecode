## Applicable Standards
- `standards/product/roadmap-conventions.md`
- `standards/security/trust-boundary-interfaces.md`
- `standards/security/trust-boundary-layered-enforcement.md`
- `standards/security/trust-boundary-change-checklist.md`
- `standards/security/policy-evaluation-foundations.md`
- `standards/security/audit-verification-scope-and-evidence-binding.md`
- `standards/global/control-plane-api-contract-shape.md`

## Resolution Notes
This change exists to make pre-beta direct-credential model access land on the same durable provider substrate that later OAuth and bridge-runtime features will reuse.

That includes preserving one shared provider-profile and auth-material model, one canonical typed model boundary, one secrets-custody posture, and one broker-projected readiness and compatibility surface.
